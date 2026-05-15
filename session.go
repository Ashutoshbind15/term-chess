package main

import (
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

type SessionManager struct {
	mu                   sync.RWMutex
	fingerPrintToProgram map[string]*tea.Program
}

func (s *SessionManager) SetProgram(fingerPrint string, program *tea.Program) {
	log.Info("Setting program for fingerprint", "fingerprint", fingerPrint)
	s.mu.Lock()
	s.fingerPrintToProgram[fingerPrint] = program
	s.mu.Unlock()
}

func (s *SessionManager) GetProgram(fingerPrint string) *tea.Program {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fingerPrintToProgram[fingerPrint]
}

func (s *SessionManager) RemoveProgram(fingerPrint string) {
	s.mu.Lock()
	delete(s.fingerPrintToProgram, fingerPrint)
	s.mu.Unlock()
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		fingerPrintToProgram: make(map[string]*tea.Program),
	}
}
