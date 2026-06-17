package main

import (
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/davecgh/go-spew/spew"
	zone "github.com/lrstanley/bubblezone"

	"github.com/Ashutoshbind15/ssh-chess/common"
)

// Context holds state that every page model needs to read or share. Models
// receive a pointer to the same Context, so mutations made by one model
// (e.g. intro creating a Player) become visible to the others without
// having to be plumbed through messages.
type Context struct {
	fingerPrint string
	renderer    *lipgloss.Renderer
	zone        *zone.Manager
	dump        io.Writer
	player      *common.Player
	width       int
	height      int
	pieceMode   common.BoardPieceMode
}

// navigateMsg is emitted by a child model to ask the root to switch pages.
// The root intercepts it and never forwards it down.
type navigateMsg struct {
	page Page
}

// closeMenuMsg is emitted by the menu model when the user backs out without
// picking an item. The root pops back to the previous page.
type closeMenuMsg struct{}

// appModel is the top-level (root) model. Its job is message routing and
// screen composition: it holds a child model per page, dispatches messages
// to the relevant child, and stitches the rendered View() of the current
// page into a header + content + footer layout.
type appModel struct {
	ctx          *Context
	page         Page
	previousPage *Page

	intro introModel
	menu  menuModel
	game  gameModel
	bot   botModel
}

func (m appModel) Init() tea.Cmd {
	return m.intro.Init()
}

func (m appModel) introBusy() bool {
	return m.intro.busy()
}

func (m appModel) openPageSelect() appModel {
	if m.page == PageSelect {
		return m
	}
	prev := m.page
	m.previousPage = &prev
	m.page = PageSelect
	return m
}

func (m appModel) closePageSelect() appModel {
	if m.previousPage == nil {
		return m
	}
	m.page = *m.previousPage
	m.previousPage = nil
	return m
}

// effectivePage returns the "real" page the user is on, treating the
// transient menu overlay (PageSelect) as the page underneath it.
func (m appModel) effectivePage() Page {
	if m.page == PageSelect && m.previousPage != nil {
		return *m.previousPage
	}
	return m.page
}

// navigateTo validates the destination, then activates the target page so
// the new page can register any follow-up command (focus, refresh, etc).
func (m appModel) navigateTo(page Page) (appModel, tea.Cmd) {
	// todo: add a toast or some sort of feedback for
	// an unexpected action
	if (page == PageGame || page == PageBot) && m.ctx.player == nil {
		page = PageIntro
	}
	var cmds []tea.Cmd
	m.page = page
	m.previousPage = nil
	var activateCmd tea.Cmd
	m, activateCmd = m.activateCurrentPage()
	if activateCmd != nil {
		cmds = append(cmds, activateCmd)
	}
	return m, tea.Batch(cmds...)
}

func (m appModel) activateCurrentPage() (appModel, tea.Cmd) {
	switch m.page {
	case PageIntro:
		var cmd tea.Cmd
		m.intro, cmd = m.intro.Activate()
		return m, cmd
	case PageGame:
		var cmd tea.Cmd
		m.game, cmd = m.game.Activate()
		return m, cmd
	case PageBot:
		var cmd tea.Cmd
		m.bot, cmd = m.bot.Activate()
		return m, cmd
	}
	return m, nil
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.ctx.dump != nil {
		spew.Fdump(m.ctx.dump, msg)
	}

	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.ctx.width = ws.Width
		m.ctx.height = ws.Height
		return m, nil
	}

	// Handle global commands
	switch msg := msg.(type) {
	case navigateMsg:
		return m.navigateTo(msg.page)
	case closeMenuMsg:
		m = m.closePageSelect()
		return m.activateCurrentPage()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.ctx.zone != nil {
				m.ctx.zone.Close()
			}
			return m, tea.Quit
		case "ctrl+b":
			m.ctx.pieceMode = m.ctx.pieceMode.Toggle()
			return m, nil
		case "tab":
			if !m.introBusy() {
				m = m.openPageSelect()
				return m.activateCurrentPage()
			}
		}
	case opponentJoinedGameMsg, gameUpdatedMsg, ClockUpdateMsg, TimeForfeitMsg, moveAppliedMsg, gameLobbyResultMsg:
		var cmd tea.Cmd
		m.game, cmd = m.game.Update(msg)
		return m, cmd
	case botMoveMsg, botAPIHealthMsg:
		var cmd tea.Cmd
		m.bot, cmd = m.bot.Update(msg)
		return m, cmd
	case gamesRefreshMsg, loadGamesMsg, loadPlayerMsg, savePlayerMsg:
		var cmd tea.Cmd
		m.intro, cmd = m.intro.Update(msg)
		return m, cmd
	case botGamesRefreshMsg, loadBotGamesMsg:
		var cmd tea.Cmd
		m.bot, cmd = m.bot.Update(msg)
		return m, cmd
	}

	// Route to page-specific update handlers
	switch m.page {
	case PageIntro:
		var cmd tea.Cmd
		m.intro, cmd = m.intro.Update(msg)
		return m, cmd
	case PageSelect:
		var cmd tea.Cmd
		m.menu, cmd = m.menu.Update(msg)
		return m, cmd
	case PageGame:
		var cmd tea.Cmd
		m.game, cmd = m.game.Update(msg)
		return m, cmd
	case PageBot:
		var cmd tea.Cmd
		m.bot, cmd = m.bot.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m appModel) headerText() string {
	if m.ctx.player != nil {
		return m.ctx.player.Username
	}
	return "Guest"
}

func (m appModel) View() string {
	r := m.ctx.renderer

	headerStyle := r.NewStyle().
		Align(lipgloss.Center).
		Width(m.ctx.width).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("62")).
		Foreground(lipgloss.Color("229")).
		Bold(true)
	header := headerStyle.Render("♜ >_ Term Chess | " + m.headerText())

	footerStyle := r.NewStyle().
		Align(lipgloss.Center).
		Width(m.ctx.width).
		Foreground(lipgloss.Color("241"))
	footer := footerStyle.Render("Page: " + string(m.page) + " | tab · menu | ctrl+c · quit")

	var pageContent string
	switch m.page {
	case PageIntro:
		pageContent = m.intro.View()
	case PageSelect:
		pageContent = m.menu.View()
	case PageGame:
		pageContent = m.game.View()
	case PageBot:
		pageContent = m.bot.View()
	default:
		pageContent = "Unknown page"
	}

	content := r.NewStyle().
		Width(m.ctx.width).
		Height(m.ctx.height - lipgloss.Height(header) - lipgloss.Height(footer)).
		Render(pageContent)

	output := lipgloss.JoinVertical(lipgloss.Top, header, content, footer)
	if m.ctx.zone != nil {
		return m.ctx.zone.Scan(output)
	}
	return output
}
