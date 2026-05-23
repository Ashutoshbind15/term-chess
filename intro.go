package main

import (
	"strings"

	"github.com/Ashutoshbind15/ssh-chess/common"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type introModel struct {
	ctx *Context

	usernameInput   textinput.Model
	usernameSpinner spinner.Model
	gamesTable      table.Model

	introLoading bool
	introSaving  bool
	introErr     string

	gamesLoading bool
	gamesErr     string
}

func newIntroModel(ctx *Context) introModel {
	usernameInput := common.InitTextInput()
	applyRendererTextInputStyles(&usernameInput, ctx.renderer)
	usernameInput.Width = textInputViewWidth

	return introModel{
		ctx:             ctx,
		usernameInput:   usernameInput,
		usernameSpinner: common.InitSpinner(),
		gamesTable:      newGamesTable(),
		introLoading:    true,
	}
}

func (m introModel) busy() bool {
	return m.introLoading || m.introSaving
}

func (m introModel) Init() tea.Cmd {
	if m.introLoading {
		return tea.Batch(
			m.usernameSpinner.Tick,
			loadPlayerCmd(m.ctx.fingerPrint),
			m.usernameInput.Focus(),
		)
	}
	return nil
}

func (m introModel) Activate() (introModel, tea.Cmd) {
	if m.ctx.player == nil && !m.busy() {
		return m, m.usernameInput.Focus()
	}
	return m, nil
}

func viewOnlyTableStyles() table.Styles {
	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	// Unfocused table still tracks cursor on row 0; empty style avoids a false selection highlight.
	styles.Selected = lipgloss.NewStyle()
	return styles
}

func newGamesTable() table.Model {
	columns := []table.Column{
		{Title: "Date", Width: 16},
		{Title: "Color", Width: 6},
		{Title: "Opponent", Width: 18},
		{Title: "Outcome", Width: 8},
		{Title: "Method", Width: 18},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(false),
		table.WithHeight(8),
	)
	t.SetStyles(viewOnlyTableStyles())
	return t
}

func loadPlayerCmd(fingerPrint string) tea.Cmd {
	return func() tea.Msg {
		player, err := dataManager.GetPlayer(fingerPrint)
		return loadPlayerMsg{player: player, err: err}
	}
}

func savePlayerCmd(p common.Player, fingerPrint string) tea.Cmd {
	return func() tea.Msg {
		err := dataManager.AddPlayer(p)
		if err == nil {
			gameManager.SetPlayer(fingerPrint, p.Username)
		}
		return savePlayerMsg{player: p, err: err}
	}
}

func loadGamesCmd(fingerPrint string) tea.Cmd {
	return func() tea.Msg {
		games, err := dataManager.GetGamesForPlayer(fingerPrint)
		return loadGamesMsg{games: games, err: err}
	}
}

func gameRowsFor(fingerPrint string, games []common.Game) []table.Row {
	rows := make([]table.Row, 0, len(games))
	for _, g := range games {
		var color, opponent string
		if g.WhiteFingerprint == fingerPrint {
			color = "white"
			opponent = g.BlackUsername
		} else {
			color = "black"
			opponent = g.WhiteUsername
		}
		if opponent == "" {
			opponent = "?"
		}
		rows = append(rows, table.Row{
			g.CreatedAt.Format("2006-01-02 15:04"),
			color,
			opponent,
			common.PlayerOutcomeLabel(g.Outcome, color),
			g.Method,
		})
	}
	return rows
}

func (m introModel) startGamesLoad() (introModel, tea.Cmd) {
	if m.ctx.player == nil {
		return m, nil
	}
	m.gamesLoading = true
	m.gamesErr = ""
	return m, tea.Batch(m.usernameSpinner.Tick, loadGamesCmd(m.ctx.fingerPrint))
}

func navigateToChatCmd() tea.Cmd {
	return func() tea.Msg { return navigateMsg{page: PageChat} }
}

func (m introModel) Update(msg tea.Msg) (introModel, tea.Cmd) {
	switch msg := msg.(type) {
	case gamesRefreshMsg:
		if m.ctx.player == nil {
			return m, nil
		}
		return m.startGamesLoad()
	case loadGamesMsg:
		m.gamesLoading = false
		if msg.err != nil {
			m.gamesErr = msg.err.Error()
			return m, nil
		}
		m.gamesErr = ""
		m.gamesTable.SetRows(gameRowsFor(m.ctx.fingerPrint, msg.games))
		return m, nil
	case loadPlayerMsg:
		m.introLoading = false
		if msg.err != nil {
			m.introErr = msg.err.Error()
			return m, m.usernameInput.Focus()
		}
		m.introErr = ""
		if msg.player != nil {
			m.ctx.player = msg.player
			gameManager.SetPlayer(m.ctx.fingerPrint, msg.player.Username)
			m.usernameInput.SetValue(msg.player.Username)
			return m.startGamesLoad()
		}
		return m, m.usernameInput.Focus()
	case savePlayerMsg:
		m.introSaving = false
		if msg.err != nil {
			m.introErr = msg.err.Error()
			return m, m.usernameInput.Focus()
		}
		m.introErr = ""
		player := msg.player
		m.ctx.player = &player
		loaded, loadCmd := m.startGamesLoad()
		return loaded, tea.Batch(loadCmd, navigateToChatCmd())
	}

	if m.busy() {
		var spCmd, tiCmd tea.Cmd
		m.usernameSpinner, spCmd = m.usernameSpinner.Update(msg)
		if _, ok := msg.(tea.KeyMsg); !ok {
			m.usernameInput, tiCmd = m.usernameInput.Update(msg)
		}
		return m, tea.Batch(spCmd, tiCmd)
	}

	if m.ctx.player != nil {
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
			return m, navigateToChatCmd()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.usernameInput, cmd = m.usernameInput.Update(msg)

	if key, ok := msg.(tea.KeyMsg); ok {
		if m.introErr != "" && key.String() != "enter" {
			m.introErr = ""
		}
		if key.String() == "enter" {
			username := strings.TrimSpace(m.usernameInput.Value())
			if m.ctx.player == nil && username != "" {
				m.usernameInput.SetValue(username)
				p := common.Player{Fingerprint: m.ctx.fingerPrint, Username: username}
				m.introErr = ""
				m.introSaving = true
				m.usernameInput.Blur()
				return m, tea.Batch(m.usernameSpinner.Tick, savePlayerCmd(p, m.ctx.fingerPrint))
			}
		}
	}

	return m, cmd
}

func (m introModel) View() string {
	r := m.ctx.renderer

	rookArt := `
       .::.
      _|||||_
     | || || |
     |_______|
     \__ ___ /
      |___|_| 
      |_|___| 
      |___|_| 
     (_______)
     /_______\`

	termArt := `  ____
 | >_ |
 |____|`

	textArt := `
  _____ _____ ____  __  __       ____ _   _ _____ ____ ____ 
 |_   _| ____|  _ \|  \/  |     / ___| | | | ____/ ___/ ___|
   | | |  _| | |_) | |\/| |____| |   | |_| |  _| \___ \___ \
   | | | |___|  _ <| |  | |____| |___|  _  | |___ ___) |__) |
   |_| |_____|_| \_\_|  |_|     \____|_| |_|_____|____/____/`

	rookBlock := r.NewStyle().Foreground(lipgloss.Color("252")).Bold(true).Render(strings.Trim(rookArt, "\n"))

	termBlock := strings.Trim(termArt, "\n")
	termStyle := r.NewStyle().Foreground(lipgloss.Color("42")).Bold(true).MarginTop(4).MarginLeft(2)
	termBlock = termStyle.Render(termBlock)

	topBlock := lipgloss.JoinHorizontal(lipgloss.Top, rookBlock, termBlock)
	textBlock := r.NewStyle().Foreground(lipgloss.Color("38")).Bold(true).Render(strings.Trim(textArt, "\n"))

	artBlock := lipgloss.JoinVertical(lipgloss.Center, topBlock, "", "", textBlock)
	lines := []string{artBlock, "", ""}

	switch {
	case m.introLoading:
		lines = append(lines, m.usernameSpinner.View()+" loading profile...")
	case m.ctx.player == nil:
		lines = append(lines, m.usernameInput.View())
		if m.introErr != "" {
			lines = append(lines, r.NewStyle().Foreground(lipgloss.Color("9")).Render(m.introErr))
		}
		if m.introSaving {
			lines = append(lines, m.usernameSpinner.View()+" saving profile...")
		}
	default:
		welcomeStyle := r.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		infoStyle := r.NewStyle().Foreground(lipgloss.Color("252"))
		lines = append(lines, welcomeStyle.Render("Welcome, "+m.ctx.player.Username), "", infoStyle.Render("Your games:"))
		switch {
		case m.gamesLoading:
			lines = append(lines, m.usernameSpinner.View()+" loading games...")
		case m.gamesErr != "":
			lines = append(lines, r.NewStyle().Foreground(lipgloss.Color("9")).Render(m.gamesErr))
		case len(m.gamesTable.Rows()) == 0:
			lines = append(lines, r.NewStyle().Faint(true).Render("No games yet."))
		default:
			lines = append(lines, m.gamesTable.View())
		}
	}

	return lipgloss.JoinVertical(lipgloss.Center, lines...)
}
