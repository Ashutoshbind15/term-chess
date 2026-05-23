package main

import (
	"strconv"
	"strings"

	"github.com/Ashutoshbind15/ssh-chess/common"
	"github.com/Ashutoshbind15/ssh-chess/managers"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/notnil/chess"
)

const (
	botPageTitle  = "Play vs Computer"
	botHelpLevels = "Select level: [1] 1100  [3] 1300  [5] 1500  [7] 1700  [9] 1900"
	botHelpColors = "Select color: [w] white  [b] black  [r] random"
	botHelpStart  = "Start a game: ctrl+n"
	botHelpMove   = "Make a move: click a piece, then click its destination square."
	botHelpResign   = "Resign: ctrl+x"
	botHelpFinished = "Play again: ctrl+n   Bot lobby: esc"
)

type botMoveMsg struct {
	gameID string
	move   string
	err    error
}

// botMoveAppliedMsg is the response to a MakePlayerMove call from a mouse click.
type botMoveAppliedMsg struct {
	game *managers.BotGame
	move string
	err  error
}

type botAPIHealthMsg struct {
	err error
}

type loadBotGamesMsg struct {
	games []common.BotGame
	err   error
}

type botGamesRefreshMsg struct{}

// botPageKind selects which subsection of the bot page handles input/rendering,
// analogous to routing between lobby vs in-progress game updates.
type botPageKind int

const (
	botPageHealthChecking botPageKind = iota
	botPageHealthFailed
	botPageLobby
	botPageGameInProgress
	botPageGameFinished
)

type botModel struct {
	ctx *Context

	currentBotGame   *managers.BotGame
	botGamesTable    table.Model
	botSelectedLevel int
	botSelectedColor chess.Color
	botNotice        string
	botMoving        bool
	botMovePending   bool
	botSpinner       spinner.Model

	selected      string
	possibleMoves []string

	botGamesLoading bool
	botGamesErr     string

	botAPICheckInFlight bool
	botAPIServiceReady  bool
	botAPIServiceErr    string
}

// botPageKind returns the active subsection derived from bot API readiness and game state.
func (m botModel) botPageKind() botPageKind {
	if m.botAPICheckInFlight {
		return botPageHealthChecking
	}
	if !m.botAPIServiceReady {
		return botPageHealthFailed
	}
	if m.currentBotGame == nil {
		return botPageLobby
	}
	switch m.currentBotGame.Status() {
	case managers.GameStatusInProgress:
		return botPageGameInProgress
	case managers.GameStatusFinished:
		return botPageGameFinished
	default:
		return botPageLobby
	}
}

func newBotModel(ctx *Context) botModel {
	return botModel{
		ctx:            ctx,
		currentBotGame: botGameManager.GameForPlayer(ctx.fingerPrint),
		botGamesTable:  newBotGamesTable(),
		botSpinner:     common.InitSpinner(),
	}
}

func (m botModel) Init() tea.Cmd { return nil }

func (m botModel) Activate() (botModel, tea.Cmd) {
	m.botNotice = ""
	m.botAPICheckInFlight = true
	m.botAPIServiceReady = false
	m.botAPIServiceErr = ""
	return m, tea.Batch(m.botSpinner.Tick, checkBotAPIServiceCmd())
}

func checkBotAPIServiceCmd() tea.Cmd {
	return func() tea.Msg {
		err := botAPIManager.HealthCheck()
		return botAPIHealthMsg{err: err}
	}
}

func navigateToChatCmdBot() tea.Cmd {
	return func() tea.Msg { return navigateMsg{page: PageChat} }
}

func loadBotGamesCmd(fingerprint string) tea.Cmd {
	return func() tea.Msg {
		games, err := dataManager.GetBotGamesForPlayer(fingerprint)
		return loadBotGamesMsg{games: games, err: err}
	}
}

func requestBotMoveCmd(gameID, fen string, level int) tea.Cmd {
	return func() tea.Msg {
		move, err := botAPIManager.BestMove(fen, level)
		return botMoveMsg{gameID: gameID, move: move, err: err}
	}
}

// applyBotMouseMoveCmd is the mouse-click variant for bot games. Like PvP,
// manager work runs in the cmd goroutine and lands as botMoveAppliedMsg in
// Update().
func applyBotMouseMoveCmd(fingerprint, move string) tea.Cmd {
	return func() tea.Msg {
		game, err := botGameManager.MakePlayerMove(fingerprint, move)
		if err != nil {
			if game2, err2 := botGameManager.MakePlayerMove(fingerprint, move+"q"); err2 == nil {
				game = game2
				move += "q"
				err = nil
			}
		}
		return botMoveAppliedMsg{game: game, move: move, err: err}
	}
}

func botGameRowsFor(games []common.BotGame) []table.Row {
	rows := make([]table.Row, 0, len(games))
	for _, g := range games {
		color := g.PlayerColor
		if color == "" {
			color = "?"
		}
		rows = append(rows, table.Row{
			g.CreatedAt.Format("2006-01-02 15:04"),
			color,
			strconv.Itoa(g.BotLevel),
			common.PlayerOutcomeLabel(g.Outcome, color),
			g.Method,
		})
	}
	return rows
}

func newBotGamesTable() table.Model {
	columns := []table.Column{
		{Title: "Date", Width: 16},
		{Title: "You", Width: 6},
		{Title: "Level", Width: 6},
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

func (m botModel) startBotRematch() (botModel, tea.Cmd) {
	if m.currentBotGame == nil {
		return m, nil
	}
	level := m.currentBotGame.BotLevel()
	color := m.currentBotGame.PlayerColor()
	if existing := botGameManager.GameForPlayer(m.ctx.fingerPrint); existing != nil {
		botGameManager.RemoveBotGame(existing.ID())
	}
	m.currentBotGame = nil
	m.selected = ""
	m.possibleMoves = nil
	m.botNotice = ""

	username := ""
	if m.ctx.player != nil {
		username = m.ctx.player.Username
	}
	game, err := botGameManager.CreateBotGame(m.ctx.fingerPrint, username, color, level)
	if err != nil {
		m.botNotice = err.Error()
		return m, nil
	}
	m.currentBotGame = game
	m.botNotice = "Rematch on. You are " + strings.ToLower(game.PlayerColor().Name()) + " vs level " + strconv.Itoa(game.BotLevel()) + "."
	if !game.IsPlayersTurn() {
		m.botMoving = true
		return m, requestBotMoveCmd(game.ID(), game.FEN(), game.BotLevel())
	}
	return m, nil
}

func (m botModel) startBotGamesLoad() (botModel, tea.Cmd) {
	if m.ctx.player == nil {
		return m, nil
	}
	m.botGamesLoading = true
	m.botGamesErr = ""
	return m, tea.Batch(m.botSpinner.Tick, loadBotGamesCmd(m.ctx.fingerPrint))
}

// persistAndReloadBotGameCmd persists a finished bot game and then reloads
// the player's bot games list. Both DB operations run inside the goroutine
// so Update() returns immediately and the event loop stays responsive. The
// reload is chained after the persist (rather than fired as a separate cmd
// in parallel) so the new game is guaranteed to be visible in the result.
func persistAndReloadBotGameCmd(gameID, fingerprint string) tea.Cmd {
	return func() tea.Msg {
		if record, ok := botGameManager.BuildBotGameRecord(gameID); ok {
			if err := dataManager.AddBotGame(record); err != nil {
				log.Error("failed to persist bot game", "id", gameID, "error", err)
			} else {
				botGameManager.RemoveBotGame(gameID)
			}
		}
		games, err := dataManager.GetBotGamesForPlayer(fingerprint)
		return loadBotGamesMsg{games: games, err: err}
	}
}

func (m botModel) updateBotHealthChecking(msg tea.Msg) (botModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && key.String() == "esc" {
		m.botAPICheckInFlight = false
		m.botAPIServiceErr = ""
		return m, navigateToChatCmdBot()
	}
	return m, nil
}

func (m botModel) updateBotHealthFailed(msg tea.Msg) (botModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.botAPICheckInFlight = false
			m.botAPIServiceErr = ""
			return m, navigateToChatCmdBot()
		case "ctrl+r":
			m.botAPICheckInFlight = true
			m.botAPIServiceErr = ""
			return m, tea.Batch(m.botSpinner.Tick, checkBotAPIServiceCmd())
		}
	}
	return m, nil
}

// Update is the entry point for the bot page. Mirrors the structure of
// UpdateGame but with no clocks and no opponent messaging.
func (m botModel) Update(msg tea.Msg) (botModel, tea.Cmd) {
	if spin, ok := msg.(spinner.TickMsg); ok && m.botPageKind() == botPageHealthChecking {
		var cmd tea.Cmd
		m.botSpinner, cmd = m.botSpinner.Update(spin)
		return m, cmd
	}

	switch msg := msg.(type) {
	case botAPIHealthMsg:
		m.botAPICheckInFlight = false
		if msg.err != nil {
			m.botAPIServiceReady = false
			m.botAPIServiceErr = msg.err.Error()
			return m, nil
		}
		m.botAPIServiceReady = true
		m.botAPIServiceErr = ""
		if m.currentBotGame == nil {
			return m.startBotGamesLoad()
		}
		return m, nil
	case botGamesRefreshMsg:
		return m.startBotGamesLoad()
	case loadBotGamesMsg:
		m.botGamesLoading = false
		if msg.err != nil {
			m.botGamesErr = msg.err.Error()
			return m, nil
		}
		m.botGamesErr = ""
		m.botGamesTable.SetRows(botGameRowsFor(msg.games))
		return m, nil
	case botMoveMsg:
		return m.handleBotMove(msg)
	case botMoveAppliedMsg:
		return m.handleBotMoveApplied(msg)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.botSpinner, cmd = m.botSpinner.Update(msg)
		return m, cmd
	}

	switch m.botPageKind() {
	case botPageHealthChecking:
		return m.updateBotHealthChecking(msg)
	case botPageHealthFailed:
		return m.updateBotHealthFailed(msg)
	case botPageLobby:
		return m.updateBotLobby(msg)
	case botPageGameInProgress:
		return m.updateBotInProgress(msg)
	case botPageGameFinished:
		return m.updateBotFinished(msg)
	default:
		return m, nil
	}
}

func (m botModel) updateBotLobby(msg tea.Msg) (botModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			return m, navigateToChatCmdBot()
		case "1":
			m.botSelectedLevel = 1100
			return m, nil
		case "3":
			m.botSelectedLevel = 1300
			return m, nil
		case "5":
			m.botSelectedLevel = 1500
			return m, nil
		case "7":
			m.botSelectedLevel = 1700
			return m, nil
		case "9":
			m.botSelectedLevel = 1900
			return m, nil
		case "w":
			m.botSelectedColor = chess.White
			return m, nil
		case "b":
			m.botSelectedColor = chess.Black
			return m, nil
		case "r":
			m.botSelectedColor = chess.NoColor
			return m, nil
		case "ctrl+n":
			if m.botSelectedLevel == 0 {
				m.botNotice = "Pick a level first (1/3/5/7/9)."
				return m, nil
			}
			username := ""
			if m.ctx.player != nil {
				username = m.ctx.player.Username
			}
			game, err := botGameManager.CreateBotGame(m.ctx.fingerPrint, username, m.botSelectedColor, m.botSelectedLevel)
			if err != nil {
				m.botNotice = err.Error()
				return m, nil
			}
			m.currentBotGame = game
			m.botNotice = "Game on. You are " + strings.ToLower(game.PlayerColor().Name()) + " vs level " + strconv.Itoa(game.BotLevel()) + "."
			m.selected = ""
			m.possibleMoves = nil

			if !game.IsPlayersTurn() {
				m.botMoving = true
				return m, requestBotMoveCmd(game.ID(), game.FEN(), game.BotLevel())
			}
			return m, nil
		}
	}

	return m, nil
}

// updateBotInProgress only handles mouse input plus a couple of keyboard
// shortcuts. Bot games are intentionally mouse-only — there is no UCI text
// input on this page, which keeps the layout simple and avoids interfering
// with bubblezone's column tracking.
func (m botModel) updateBotInProgress(msg tea.Msg) (botModel, tea.Cmd) {
	if mouseMsg, ok := msg.(tea.MouseMsg); ok {
		return m.handleBotBoardMouse(mouseMsg)
	}

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.selected = ""
			m.possibleMoves = nil
			return m, navigateToChatCmdBot()
		case "ctrl+x":
			game, err := botGameManager.Resign(m.ctx.fingerPrint)
			if err != nil {
				m.botNotice = err.Error()
				return m, nil
			}
			m.currentBotGame = game
			m.botNotice = ""
			return m, persistAndReloadBotGameCmd(game.ID(), m.ctx.fingerPrint)
		}
	}
	return m, nil
}

func (m botModel) updateBotFinished(msg tea.Msg) (botModel, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc":
			m.currentBotGame = nil
			m.selected = ""
			m.possibleMoves = nil
			m.botNotice = ""
			return m.startBotGamesLoad()
		case "ctrl+n":
			return m.startBotRematch()
		}
	}
	return m, nil
}

func (m botModel) handleBotMove(msg botMoveMsg) (botModel, tea.Cmd) {
	if m.currentBotGame == nil || m.currentBotGame.ID() != msg.gameID {
		return m, nil
	}
	m.botMoving = false
	if msg.err != nil {
		m.botNotice = "Bot error: " + msg.err.Error()
		return m, nil
	}
	game, err := botGameManager.ApplyBotMove(msg.gameID, msg.move)
	if err != nil {
		m.botNotice = "Bot error: " + err.Error()
		return m, nil
	}
	m.currentBotGame = game
	if game.Status() == managers.GameStatusFinished {
		m.botNotice = ""
		return m, persistAndReloadBotGameCmd(game.ID(), m.ctx.fingerPrint)
	}
	m.botNotice = "Bot played " + msg.move + "."
	return m, nil
}

func (m botModel) handleBotMoveApplied(msg botMoveAppliedMsg) (botModel, tea.Cmd) {
	m.botMovePending = false
	if msg.err != nil {
		m.botNotice = "Move rejected: " + msg.err.Error()
		m.selected = ""
		m.possibleMoves = nil
		return m, nil
	}
	if msg.game == nil {
		return m, nil
	}
	m.currentBotGame = msg.game
	m.selected = ""
	m.possibleMoves = nil
	if msg.game.Status() == managers.GameStatusFinished {
		m.botNotice = ""
		return m, persistAndReloadBotGameCmd(msg.game.ID(), m.ctx.fingerPrint)
	}
	m.botNotice = "You played " + msg.move + "."

	m.botMoving = true
	return m, requestBotMoveCmd(msg.game.ID(), msg.game.FEN(), msg.game.BotLevel())
}

func (m botModel) handleBotBoardMouse(msg tea.MouseMsg) (botModel, tea.Cmd) {
	if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	if m.botMoving || m.botMovePending {
		return m, nil
	}
	if !m.currentBotGame.IsPlayersTurn() {
		m.selected = ""
		m.possibleMoves = nil
		return m, nil
	}

	colorIsWhite := m.currentBotGame.PlayerColor() == chess.White
	playerColor := m.currentBotGame.PlayerColor()
	doesClick := false

	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			pos := convertToChessboardPosition(j, i, colorIsWhite)
			if !m.ctx.zone.Get(pos).InBounds(msg) {
				continue
			}
			doesClick = true

			selected, possibleMoves, moveUCI := boardClickResult(m.currentBotGame.Game().FEN(), pos, m.selected, m.possibleMoves, playerColor)
			if moveUCI != "" {
				m.selected = ""
				m.possibleMoves = nil
				m.botMovePending = true
				return m, applyBotMouseMoveCmd(m.ctx.fingerPrint, moveUCI)
			}

			m.selected = selected
			m.possibleMoves = possibleMoves
			return m, nil
		}
	}

	if !doesClick {
		m.selected = ""
		m.possibleMoves = nil
	}

	return m, nil
}

func (m botModel) renderBotBoardFromFEN() string {
	fen := m.currentBotGame.Game().FEN()
	colorIsWhite := m.currentBotGame.PlayerColor() != chess.Black
	return renderChessBoard(m.ctx.renderer, m.ctx.zone, fen, colorIsWhite, m.selected, m.possibleMoves)
}

func botStatusLine(g *managers.BotGame) string {
	if g == nil {
		return ""
	}
	switch g.Status() {
	case managers.GameStatusInProgress:
		return "Status: in progress."
	case managers.GameStatusFinished:
		return "Status: finished."
	}
	return ""
}

func botTurnLine(g *managers.BotGame, botMoving bool) string {
	if g == nil {
		return ""
	}
	you := strings.ToLower(g.PlayerColor().Name())
	if g.Status() != managers.GameStatusInProgress {
		return ""
	}
	if g.IsPlayersTurn() {
		return "You are " + you + ". Your turn."
	}
	if botMoving {
		return "You are " + you + ". Bot is thinking..."
	}
	return "You are " + you + ". Bot to move."
}

func botResultLine(g *managers.BotGame) string {
	if g == nil || g.Status() != managers.GameStatusFinished {
		return ""
	}
	return common.GameResultSummary(g.Outcome(), g.Method(), strings.ToLower(g.PlayerColor().Name()))
}

func (m botModel) View() string {
	switch m.botPageKind() {
	case botPageHealthChecking:
		return m.viewBotHealthCheck()
	case botPageHealthFailed:
		return m.viewBotAPIUnavailable()
	case botPageLobby:
		return m.viewBotLobby()
	case botPageGameInProgress:
		return m.viewBotInProgress()
	case botPageGameFinished:
		return m.viewBotFinished()
	default:
		return ""
	}
}

func (m botModel) viewBotHealthCheck() string {
	r := m.ctx.renderer
	title := r.NewStyle().Bold(true).Foreground(lipgloss.Color("62")).Padding(0, 1).Render(botPageTitle)
	body := r.NewStyle().Foreground(lipgloss.Color("241")).Render("Checking bot engine (GET /health)…")
	urlLine := r.NewStyle().Foreground(lipgloss.Color("252")).Render(botAPIManager.BaseURL())
	help := r.NewStyle().Foreground(lipgloss.Color("241")).Render("esc · return to lobby")
	return lipgloss.JoinVertical(lipgloss.Left, title, "", m.botSpinner.View()+" "+body, "", urlLine, "", help)
}

func (m botModel) viewBotAPIUnavailable() string {
	r := m.ctx.renderer
	title := r.NewStyle().Bold(true).Foreground(lipgloss.Color("62")).Padding(0, 1).Render(botPageTitle)
	head := r.NewStyle().Foreground(lipgloss.Color("9")).Render("Bot engine is not reachable.")
	detail := r.NewStyle().Foreground(lipgloss.Color("252")).Render(m.botAPIServiceErr)
	base := r.NewStyle().Foreground(lipgloss.Color("241")).Render("BOT_API_URL: " + botAPIManager.BaseURL())
	help := r.NewStyle().Foreground(lipgloss.Color("241")).Render("ctrl+r · retry health check   esc · return to lobby")
	return lipgloss.JoinVertical(lipgloss.Left, title, "", head, "", detail, "", base, "", help)
}

func (m botModel) viewBotLobby() string {
	r := m.ctx.renderer

	titleStyle := r.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		Padding(0, 1)

	helpStyle := r.NewStyle().Foreground(lipgloss.Color("241"))
	highlightStyle := r.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	infoStyle := r.NewStyle().Foreground(lipgloss.Color("252"))
	noticeStyle := r.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Padding(0, 1)

	rows := []string{
		titleStyle.Render(botPageTitle),
		"",
	}
	if m.botSelectedLevel == 0 {
		rows = append(rows, helpStyle.Render(botHelpLevels))
	} else {
		rows = append(rows, infoStyle.Render("Level: ")+highlightStyle.Render(strconv.Itoa(m.botSelectedLevel))+helpStyle.Render("  (press 1/3/5/7/9 to change)"))
	}
	colorChoice := "random"
	switch m.botSelectedColor {
	case chess.White:
		colorChoice = "white"
	case chess.Black:
		colorChoice = "black"
	}
	rows = append(rows, infoStyle.Render("Color: ")+highlightStyle.Render(colorChoice)+helpStyle.Render("  ("+botHelpColors+")"))
	rows = append(rows, "", helpStyle.Render(botHelpStart))

	if m.botNotice != "" {
		rows = append(rows, "", noticeStyle.Render(m.botNotice))
	}
	rows = append(rows, "", infoStyle.Render("Your bot games:"))
	switch {
	case m.botGamesLoading:
		rows = append(rows, m.botSpinner.View()+" loading...")
	case m.botGamesErr != "":
		rows = append(rows, r.NewStyle().Foreground(lipgloss.Color("9")).Render(m.botGamesErr))
	case len(m.botGamesTable.Rows()) == 0:
		rows = append(rows, r.NewStyle().Faint(true).Render("No bot games yet."))
	default:
		rows = append(rows, m.botGamesTable.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m botModel) botHeaderRows() []string {
	r := m.ctx.renderer

	titleStyle := r.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		Padding(0, 1)

	infoStyle := r.NewStyle().Foreground(lipgloss.Color("252"))
	highlightStyle := r.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	noticeStyle := r.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Padding(0, 1)

	rows := []string{
		titleStyle.Render(botPageTitle),
		"",
		infoStyle.Render("Game ID: ") + highlightStyle.Render(m.currentBotGame.ID()),
		infoStyle.Render("Level: ") + highlightStyle.Render(strconv.Itoa(m.currentBotGame.BotLevel())),
		infoStyle.Render(botStatusLine(m.currentBotGame)),
	}
	if turn := botTurnLine(m.currentBotGame, m.botMoving); turn != "" {
		rows = append(rows, highlightStyle.Render(turn))
	}
	if m.botNotice != "" {
		rows = append(rows, noticeStyle.Render(m.botNotice))
	}
	return rows
}

func (m botModel) viewBotInProgress() string {
	helpStyle := m.ctx.renderer.NewStyle().Foreground(lipgloss.Color("241"))
	rows := append(m.botHeaderRows(), "", m.renderBotBoardFromFEN(), "", helpStyle.Render(botHelpMove), helpStyle.Render(botHelpResign))
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m botModel) viewBotFinished() string {
	r := m.ctx.renderer
	helpStyle := r.NewStyle().Foreground(lipgloss.Color("241"))
	resultStyle := r.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Bold(true).Padding(0, 1)

	rows := append(m.botHeaderRows(), "")
	if result := botResultLine(m.currentBotGame); result != "" {
		rows = append(rows, resultStyle.Render(result))
	}
	rows = append(rows, "", m.renderBotBoardFromFEN(), "", helpStyle.Render(botHelpFinished))
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
