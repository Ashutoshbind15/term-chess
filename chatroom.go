package main

import (
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// ChatRoom is an in-memory, ephemeral broadcast channel. Membership is tied
// to the user actively viewing the chat page (Join on enter, Leave on exit
// or disconnect). Messages are only delivered to current members and are
// kept in a small ring buffer so newcomers see a bit of recent context.
// Nothing is persisted across server restarts.

const (
	chatBacklogSize    = 30
	chatMaxMessageLen  = 200
	chatMaxClientLines = 100
)

type chatSub struct {
	program  *tea.Program
	username string
}

type ChatRoom struct {
	mu          sync.RWMutex
	subscribers map[string]*chatSub
	recent      []message
}

func NewChatRoom() *ChatRoom {
	return &ChatRoom{
		subscribers: make(map[string]*chatSub),
	}
}

// Join registers the fingerprint as a current viewer of the chat page.
// It returns the recent backlog so the joiner can render some context.
// If username is non-empty and this is a fresh join, a system message is
// broadcast to the rest of the room. Idempotent: rejoining (e.g. after
// closing the menu) does not re-broadcast a join notice.
func (c *ChatRoom) Join(fp string, p *tea.Program, username string) []message {
	c.mu.Lock()
	_, already := c.subscribers[fp]
	c.subscribers[fp] = &chatSub{program: p, username: username}
	backlog := append([]message(nil), c.recent...)
	subs := c.snapshotLocked()
	count := len(subs)
	c.mu.Unlock()

	if !already && username != "" {
		sys := message{system: true, content: username + " joined the room"}
		fanout(subs, fp, sys)
	}
	broadcastPresence(subs, count)
	return backlog
}

// Leave removes the fingerprint from the room and notifies others.
// Safe to call when the user was never a member.
func (c *ChatRoom) Leave(fp string) {
	c.mu.Lock()
	prev, was := c.subscribers[fp]
	delete(c.subscribers, fp)
	subs := c.snapshotLocked()
	count := len(subs)
	c.mu.Unlock()

	if !was {
		return
	}
	if prev.username != "" {
		sys := message{system: true, content: prev.username + " left the room"}
		fanout(subs, "", sys)
	}
	broadcastPresence(subs, count)
}

// Broadcast publishes a user-authored message to all subscribers. The
// returned tea.Cmd echoes the message back to the sender's own program.
// Empty / whitespace-only content is dropped.
func (c *ChatRoom) Broadcast(senderFP string, msg message) tea.Cmd {
	msg.content = strings.TrimSpace(msg.content)
	if msg.content == "" {
		return nil
	}
	if len(msg.content) > chatMaxMessageLen {
		msg.content = msg.content[:chatMaxMessageLen]
	}
	msg.system = false

	c.mu.Lock()
	c.recent = append(c.recent, msg)
	if len(c.recent) > chatBacklogSize {
		c.recent = c.recent[len(c.recent)-chatBacklogSize:]
	}
	subs := c.snapshotLocked()
	c.mu.Unlock()

	fanout(subs, senderFP, msg)
	return func() tea.Msg { return msg }
}

func (c *ChatRoom) snapshotLocked() map[string]*chatSub {
	subs := make(map[string]*chatSub, len(c.subscribers))
	for k, v := range c.subscribers {
		subs[k] = v
	}
	return subs
}

// fanout delivers msg to every subscriber whose fingerprint != skipFP via
// the bubbletea Program. Sends are launched as goroutines so a slow or
// closing program never blocks the broadcaster.
func fanout(subs map[string]*chatSub, skipFP string, msg tea.Msg) {
	for fp, sub := range subs {
		if fp == skipFP || sub == nil || sub.program == nil {
			continue
		}
		prog := sub.program
		go prog.Send(msg)
	}
}

func broadcastPresence(subs map[string]*chatSub, count int) {
	pm := presenceMsg{count: count}
	fanout(subs, "", pm)
}
