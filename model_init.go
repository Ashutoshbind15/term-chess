package main

import (
	"github.com/Ashutoshbind15/ssh-chess/common"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

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

	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	styles.Selected = styles.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(styles)

	return t
}

func initModel(fingerPrint string) model {
	chatTa := common.InitTextArea()
	usernameInputTa := common.InitTextInput()
	gameJoinInput := common.InitTextInput()
	moveInput := common.InitTextInput()
	gameJoinInput.Prompt = "game id> "
	gameJoinInput.Placeholder = "Enter game ID"
	moveInput.Prompt = "move> "
	moveInput.Placeholder = "e2e4"

	return model{
		counter:         0,
		messages:        []message{},
		fingerPrint:     fingerPrint,
		chatTextarea:    chatTa,
		usernameInput:   usernameInputTa,
		usernameSpinner: common.InitSpinner(),
		gameJoinInput:   gameJoinInput,
		moveInput:       moveInput,
		page:            PageIntro,
		introLoading:    true,
		pageList:        newPageList(80, 22),
		currentGame:     gameManager.GameForPlayer(fingerPrint),
		gamesTable:      newGamesTable(),
	}
}
