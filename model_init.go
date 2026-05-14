package main

import (
	"io"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

// textInputViewWidth must be > 0: bubbles v0.21 textinput placeholderView uses
// make([]rune, m.Width+1) and early-returns when Width is 0, so a zero width
// truncates the placeholder to a single character.
const textInputViewWidth = 40

func applyRendererTextareaStyles(ta *textarea.Model, r *lipgloss.Renderer) {
	focused := textarea.Style{
		Base:             r.NewStyle(),
		CursorLine:       r.NewStyle().Background(lipgloss.AdaptiveColor{Light: "255", Dark: "0"}),
		CursorLineNumber: r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "240"}),
		EndOfBuffer:      r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "254", Dark: "0"}),
		LineNumber:       r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "249", Dark: "7"}),
		Placeholder:      r.NewStyle().Foreground(lipgloss.Color("240")),
		Prompt:           r.NewStyle().Foreground(lipgloss.Color("7")),
		Text:             r.NewStyle(),
	}
	blurred := textarea.Style{
		Base:             r.NewStyle(),
		CursorLine:       r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "245", Dark: "7"}),
		CursorLineNumber: r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "249", Dark: "7"}),
		EndOfBuffer:      r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "254", Dark: "0"}),
		LineNumber:       r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "249", Dark: "7"}),
		Placeholder:      r.NewStyle().Foreground(lipgloss.Color("240")),
		Prompt:           r.NewStyle().Foreground(lipgloss.Color("7")),
		Text:             r.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "245", Dark: "7"}),
	}
	ta.FocusedStyle = focused
	ta.BlurredStyle = blurred
	ta.Cursor.Style = r.NewStyle()
	ta.Cursor.TextStyle = r.NewStyle()
}

func applyRendererTextInputStyles(ti *textinput.Model, r *lipgloss.Renderer) {
	ti.PromptStyle = r.NewStyle()
	ti.TextStyle = r.NewStyle()
	ti.PlaceholderStyle = r.NewStyle().Foreground(lipgloss.Color("240"))
	ti.CompletionStyle = r.NewStyle().Foreground(lipgloss.Color("240"))
	ti.Cursor.Style = r.NewStyle()
	ti.Cursor.TextStyle = r.NewStyle()
}

func newAppModel(fingerPrint string, renderer *lipgloss.Renderer, dump io.Writer) appModel {
	ctx := &Context{
		fingerPrint: fingerPrint,
		renderer:    renderer,
		zone:        zone.New(),
		dump:        dump,
	}

	return appModel{
		ctx:   ctx,
		page:  PageIntro,
		intro: newIntroModel(ctx),
		chat:  newChatModel(ctx),
		menu:  newMenuModel(ctx),
		game:  newGameModel(ctx),
		bot:   newBotModel(ctx),
	}
}
