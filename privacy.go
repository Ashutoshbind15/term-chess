package main

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Ashutoshbind15/ssh-chess/common"
)

type privacyStep int

const (
	privacyStepView privacyStep = iota
	privacyStepConfirm
	privacyStepDeleting
	privacyStepDone
)

type deletePlayerDataMsg struct {
	ok  bool
	err error
}

type privacyModel struct {
	ctx     *Context
	step    privacyStep
	spinner spinner.Model
	err     string
}

func newPrivacyModel(ctx *Context) privacyModel {
	return privacyModel{
		ctx:     ctx,
		spinner: common.InitSpinner(),
	}
}

func (m privacyModel) Init() tea.Cmd { return nil }

func (m privacyModel) Activate() (privacyModel, tea.Cmd) {
	m.step = privacyStepView
	m.err = ""
	return m, nil
}

func deletePlayerDataCmd(fingerprint string) tea.Cmd {
	return func() tea.Msg {
		snap, err := gameManager.EndActiveGameForDeletion(fingerprint)
		if err != nil {
			return deletePlayerDataMsg{err: err}
		}
		if snap != nil {
			finalizeGameAction(snap, fingerprint, "")
		}

		if err := dataManager.DeletePlayerData(fingerprint); err != nil {
			return deletePlayerDataMsg{err: err}
		}

		gameManager.RemovePlayer(fingerprint)
		botGameManager.RemovePlayer(fingerprint)
		return deletePlayerDataMsg{ok: true}
	}
}

func quitAfterDelayCmd() tea.Cmd {
	return tea.Tick(time.Second*2, func(time.Time) tea.Msg {
		return disconnectAfterDeleteMsg{}
	})
}

type disconnectAfterDeleteMsg struct{}

func (m privacyModel) startDelete() (privacyModel, tea.Cmd) {
	m.step = privacyStepDeleting
	m.err = ""
	return m, tea.Batch(m.spinner.Tick, deletePlayerDataCmd(m.ctx.fingerPrint))
}

func (m privacyModel) Update(msg tea.Msg) (privacyModel, tea.Cmd) {
	switch msg := msg.(type) {
	case deletePlayerDataMsg:
		if msg.err != nil {
			m.step = privacyStepView
			m.err = msg.err.Error()
			return m, nil
		}
		m.ctx.player = nil
		m.step = privacyStepDone
		return m, quitAfterDelayCmd()
	case disconnectAfterDeleteMsg:
		return m, tea.Quit
	case tea.KeyMsg:
		if m.step == privacyStepDeleting || m.step == privacyStepDone {
			return m, nil
		}
		switch msg.String() {
		case "esc":
			if m.step == privacyStepConfirm {
				m.step = privacyStepView
				m.err = ""
				return m, nil
			}
			return m, func() tea.Msg { return navigateMsg{page: PageIntro} }
		case "d":
			if m.step == privacyStepView && m.ctx.player != nil {
				m.step = privacyStepConfirm
				m.err = ""
			}
		case "y":
			if m.step == privacyStepConfirm {
				return m.startDelete()
			}
		case "n":
			if m.step == privacyStepConfirm {
				m.step = privacyStepView
				m.err = ""
			}
		}
	}

	if m.step == privacyStepDeleting {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m privacyModel) View() string {
	r := m.ctx.renderer
	titleStyle := r.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	bodyStyle := r.NewStyle().Foreground(lipgloss.Color("252"))
	mutedStyle := r.NewStyle().Foreground(lipgloss.Color("241"))
	errStyle := r.NewStyle().Foreground(lipgloss.Color("9"))

	lines := []string{
		titleStyle.Render("Privacy and data"),
		"",
		bodyStyle.Render("Policy: "+privacyURL),
		"",
		bodyStyle.Render("You can request deletion of data linked to your SSH key:"),
		mutedStyle.Render("  • your profile and generated display name"),
		mutedStyle.Render("  • your bot game history"),
		mutedStyle.Render("  • your fingerprint and username in stored multiplayer games"),
		mutedStyle.Render("    (your display name becomes \"deleteduser\"; opponents keep"),
		mutedStyle.Render("    the game until both sides have deleted, then it is removed)"),
		"",
	}

	switch m.step {
	case privacyStepConfirm:
		lines = append(lines,
			r.NewStyle().Foreground(lipgloss.Color("214")).Bold(true).Render("Delete all stored data for this SSH key?"),
			mutedStyle.Render("This cannot be undone."),
			mutedStyle.Render("Press y to confirm, n or esc to cancel."),
		)
	case privacyStepDeleting:
		lines = append(lines, m.spinner.View()+" deleting your data...")
	case privacyStepDone:
		lines = append(lines,
			r.NewStyle().Foreground(lipgloss.Color("42")).Render("Your stored data has been deleted."),
			mutedStyle.Render("Disconnecting..."),
		)
	default:
		if m.ctx.player == nil {
			lines = append(lines, mutedStyle.Render("No stored profile is linked to this SSH key yet."))
		} else {
			lines = append(lines, mutedStyle.Render("Press d to delete your stored data."))
		}
		lines = append(lines, "", mutedStyle.Render("You can also email "+contactEmail+"."))
		if m.err != "" {
			lines = append(lines, "", errStyle.Render(m.err))
		}
		lines = append(lines, "", mutedStyle.Render("esc · back to intro"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
