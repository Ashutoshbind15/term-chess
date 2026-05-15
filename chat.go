package main

import (
	"fmt"

	"github.com/Ashutoshbind15/ssh-chess/common"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type chatModel struct {
	ctx          *Context
	chatTextarea textarea.Model
	messages     []message
	onlineCount  int
	joined       bool
}

func newChatModel(ctx *Context) chatModel {
	ta := common.InitTextArea()
	applyRendererTextareaStyles(&ta, ctx.renderer)
	return chatModel{ctx: ctx, chatTextarea: ta}
}

func (m chatModel) Init() tea.Cmd { return nil }

// Activate is called every time the user lands on the chat page (including
// returning from the menu). It joins the room on the first activation and
// seeds the local view with whatever recent backlog the room has.
func (m chatModel) Activate() (chatModel, tea.Cmd) {
	if m.ctx.player == nil {
		return m, m.chatTextarea.Focus()
	}
	if !m.joined {
		prog := sessionManager.GetProgram(m.ctx.fingerPrint)
		backlog := chatRoom.Join(m.ctx.fingerPrint, prog, m.ctx.player.Username)
		m.messages = backlog
		m.joined = true
	}
	return m, m.chatTextarea.Focus()
}

// Deactivate is called by the root model when the user navigates away from
// the chat page (but not when temporarily opening the menu). It leaves the
// room so further messages are not delivered, and clears the local buffer.
func (m chatModel) Deactivate() chatModel {
	if !m.joined {
		return m
	}
	chatRoom.Leave(m.ctx.fingerPrint)
	m.joined = false
	m.messages = nil
	m.onlineCount = 0
	return m
}

func (m chatModel) Update(msg tea.Msg) (chatModel, tea.Cmd) {
	var tiCmd tea.Cmd
	m.chatTextarea, tiCmd = m.chatTextarea.Update(msg)
	var rescmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			if m.ctx.player == nil || !m.joined {
				return m, tiCmd
			}
			cmd := chatRoom.Broadcast(m.ctx.fingerPrint, message{
				sender:  m.ctx.player.Username,
				content: m.chatTextarea.Value(),
			})
			if cmd != nil {
				rescmds = append(rescmds, cmd)
			}
			m.chatTextarea.Reset()
		}
	case message:
		m.messages = append(m.messages, msg)
		if len(m.messages) > chatMaxClientLines {
			m.messages = m.messages[len(m.messages)-chatMaxClientLines:]
		}
	case presenceMsg:
		m.onlineCount = msg.count
	}

	rescmds = append(rescmds, tiCmd)
	return m, tea.Batch(rescmds...)
}

func (m chatModel) View() string {
	r := m.ctx.renderer

	titleStyle := r.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		Padding(0, 1).
		MarginBottom(1)

	presenceStyle := r.NewStyle().
		Foreground(lipgloss.Color("245")).
		Faint(true)

	senderStyle := r.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	msgStyle := r.NewStyle().
		Foreground(lipgloss.Color("252"))

	systemStyle := r.NewStyle().
		Foreground(lipgloss.Color("244")).
		Italic(true)

	title := titleStyle.Render("Chatroom")
	presence := presenceStyle.Render(fmt.Sprintf("%d online", m.onlineCount))
	rows := []string{lipgloss.JoinHorizontal(lipgloss.Left, title, " ", presence)}

	if len(m.messages) == 0 {
		rows = append(rows, r.NewStyle().Faint(true).Render("It's quiet in here. Say hi!"))
	} else {
		for _, msg := range m.messages {
			if msg.system {
				rows = append(rows, systemStyle.Render("• "+msg.content))
				continue
			}
			sender := senderStyle.Render(msg.sender + ":")
			content := msgStyle.Render(msg.content)
			rows = append(rows, sender+" "+content)
		}
	}

	rows = append(rows, "", m.chatTextarea.View())
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
