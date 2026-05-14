package main

import (
	"github.com/Ashutoshbind15/ssh-chess/common"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type chatModel struct {
	ctx          *Context
	chatTextarea textarea.Model
	messages     []message
}

func newChatModel(ctx *Context) chatModel {
	ta := common.InitTextArea()
	applyRendererTextareaStyles(&ta, ctx.renderer)
	return chatModel{ctx: ctx, chatTextarea: ta}
}

func (m chatModel) Init() tea.Cmd { return nil }

func (m chatModel) Activate() (chatModel, tea.Cmd) {
	return m, m.chatTextarea.Focus()
}

// UpdateChat handles chat-specific update logic
func (m chatModel) Update(msg tea.Msg) (chatModel, tea.Cmd) {
	var tiCmd tea.Cmd
	m.chatTextarea, tiCmd = m.chatTextarea.Update(msg)
	var rescmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			if m.ctx.player == nil {
				return m, tiCmd
			}
			cmds := sessionManager.SendMessage(m.ctx.fingerPrint, message{
				sender:  m.ctx.player.Username,
				content: m.chatTextarea.Value(),
			})
			rescmds = append(rescmds, cmds...)
			m.chatTextarea.Reset()
		}
	case message:
		m.messages = append(m.messages, msg)
	}

	rescmds = append(rescmds, tiCmd)
	return m, tea.Batch(rescmds...)
}

// ViewChat renders the chat view
func (m chatModel) View() string {
	r := m.ctx.renderer

	titleStyle := r.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		Padding(0, 1).
		MarginBottom(1)

	senderStyle := r.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	msgStyle := r.NewStyle().
		Foreground(lipgloss.Color("252"))

	rows := []string{titleStyle.Render("Lobby Chat")}

	if len(m.messages) == 0 {
		rows = append(rows, r.NewStyle().Faint(true).Render("No messages yet. Be the first to say hi!"))
	} else {
		for _, msg := range m.messages {
			sender := senderStyle.Render(msg.sender + ":")
			content := msgStyle.Render(msg.content)
			rows = append(rows, sender+" "+content)
		}
	}

	rows = append(rows, "", m.chatTextarea.View())
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
