package main

import tea "github.com/charmbracelet/bubbletea"

func (m model) ViewIntro() string {
	return "Intro Page"
}

func (m model) UpdateIntro(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		msgType := msg.Type
		if(msgType == tea.KeyEsc) {
			m.page = PageChat
		}
	}
	
	return m, nil
}