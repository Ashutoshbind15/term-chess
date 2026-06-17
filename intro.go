package main

import (
	"strings"

	"github.com/Ashutoshbind15/ssh-chess/common"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type introModel struct {
	ctx *Context

	usernameSpinner spinner.Model
	gamesTable      table.Model

	introLoading bool
	introSaving  bool
	introErr     string

	gamesLoading bool
	gamesErr     string
}

func newIntroModel(ctx *Context) introModel {
	return introModel{
		ctx:             ctx,
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
		)
	}
	return nil
}

func (m introModel) Activate() (introModel, tea.Cmd) {
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

func (m introModel) createPlayer() (introModel, tea.Cmd) {
	username := common.UsernameForFingerprint(m.ctx.fingerPrint)
	p := common.Player{Fingerprint: m.ctx.fingerPrint, Username: username}
	m.introErr = ""
	m.introSaving = true
	return m, tea.Batch(m.usernameSpinner.Tick, savePlayerCmd(p, m.ctx.fingerPrint))
}

func navigateToGameCmd() tea.Cmd {
	return func() tea.Msg { return navigateMsg{page: PageGame} }
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
			return m, nil
		}
		m.introErr = ""
		if msg.player != nil {
			m.ctx.player = msg.player
			gameManager.SetPlayer(m.ctx.fingerPrint, msg.player.Username)
			return m.startGamesLoad()
		}
		return m.createPlayer()
	case savePlayerMsg:
		m.introSaving = false
		if msg.err != nil {
			m.introErr = msg.err.Error()
			return m, nil
		}
		m.introErr = ""
		player := msg.player
		m.ctx.player = &player
		loaded, loadCmd := m.startGamesLoad()
		return loaded, tea.Batch(loadCmd, navigateToGameCmd())
	}

	if m.busy() {
		var spCmd tea.Cmd
		m.usernameSpinner, spCmd = m.usernameSpinner.Update(msg)
		return m, spCmd
	}

	if m.ctx.player != nil {
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
			return m, navigateToGameCmd()
		}
	}

	return m, nil
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
	case m.introSaving:
		lines = append(lines, m.usernameSpinner.View()+" setting up profile...")
	case m.ctx.player == nil:
		if m.introErr != "" {
			lines = append(lines, r.NewStyle().Foreground(lipgloss.Color("9")).Render(m.introErr))
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
