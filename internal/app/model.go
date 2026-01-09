package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattmezza/tlgram/internal/config"
	"github.com/mattmezza/tlgram/internal/keybind"
	"github.com/mattmezza/tlgram/internal/telegram"
	"github.com/mattmezza/tlgram/internal/ui/auth"
	"github.com/mattmezza/tlgram/internal/ui/statusbar"
)

// View represents the current view
type View int

const (
	ViewAuth View = iota
	ViewChatList
	ViewChat
	ViewSwitcher
)

// Model is the main application model
type Model struct {
	// Dimensions
	width  int
	height int

	// State
	vim         *keybind.VimState
	currentView View
	prevView    View
	quitting    bool

	// Target chat from CLI
	targetChat string

	// Configuration
	config *config.Config

	// Demo mode (runs without TDLib)
	demoMode bool

	// Sub-components
	authView  auth.Model
	statusBar statusbar.Model
	input     textinput.Model

	// Telegram client (nil in demo mode)
	telegram *telegram.Client

	// Chat state
	chats       []*Chat
	messages    []*Message
	selectedIdx int
	chatIdx     int
	currentChat *Chat

	// Reply state
	replyingTo *Message
}

// Chat represents a chat in the list
type Chat struct {
	ID          int64
	Name        string
	UnreadCount int
	LastMessage string
	LastTime    time.Time
}

// Message represents a chat message
type Message struct {
	ID         int64
	Sender     string
	Text       string
	IsOwn      bool
	Time       time.Time
	ReplyToMsg *Message
}

// New creates a new application model
func New(cfg *config.Config, targetChat string) Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message... (i to focus, Esc to blur)"
	ti.CharLimit = 4096

	vim := keybind.NewVimState()

	// Check if we have API credentials - if not, we're in demo mode
	demoMode := cfg.Telegram.APIID == 0 || cfg.Telegram.APIHash == ""

	m := Model{
		config:     cfg,
		targetChat: targetChat,
		vim:        vim,
		demoMode:   demoMode,
		authView:   auth.New(),
		statusBar:  statusbar.New(),
		input:      ti,
		chats:      make([]*Chat, 0),
		messages:   make([]*Message, 0),
	}

	// In demo mode, start with chat view directly
	if demoMode {
		m.currentView = ViewChat
		m.statusBar.SetConnectionState(telegram.ConnectionStateReady)
		m.loadDemoData()
	} else {
		m.currentView = ViewAuth
	}

	return m
}

// loadDemoData loads sample data for demo mode
func (m *Model) loadDemoData() {
	// Demo chats
	m.chats = []*Chat{
		{ID: 1, Name: "@john_doe", UnreadCount: 3, LastMessage: "Hey, how's the project going?", LastTime: time.Now().Add(-5 * time.Minute)},
		{ID: 2, Name: "Project Alpha", UnreadCount: 0, LastMessage: "Build succeeded!", LastTime: time.Now().Add(-1 * time.Hour)},
		{ID: 3, Name: "@jane_smith", UnreadCount: 1, LastMessage: "Can you review my PR?", LastTime: time.Now().Add(-2 * time.Hour)},
		{ID: 4, Name: "Team Chat", UnreadCount: 15, LastMessage: "Meeting at 3pm", LastTime: time.Now().Add(-30 * time.Minute)},
	}

	// Resolve target chat or use first chat
	if m.targetChat != "" {
		resolved := m.config.ResolveAlias(m.targetChat)
		for i, chat := range m.chats {
			if strings.EqualFold(chat.Name, resolved) || strings.EqualFold(chat.Name, m.targetChat) {
				m.chatIdx = i
				m.currentChat = chat
				break
			}
		}
	}
	if m.currentChat == nil && len(m.chats) > 0 {
		m.currentChat = m.chats[0]
	}

	// Demo messages for current chat
	if m.currentChat != nil {
		m.statusBar.SetChatName(m.currentChat.Name)
		m.messages = []*Message{
			{ID: 1, Sender: "john_doe", Text: "Hey! 👋", IsOwn: false, Time: time.Now().Add(-10 * time.Minute)},
			{ID: 2, Sender: "You", Text: "Hi John! What's up?", IsOwn: true, Time: time.Now().Add(-9 * time.Minute)},
			{ID: 3, Sender: "john_doe", Text: "Working on the new feature. Have you seen the latest PR?", IsOwn: false, Time: time.Now().Add(-8 * time.Minute)},
			{ID: 4, Sender: "You", Text: "Not yet, I'll check it out now", IsOwn: true, Time: time.Now().Add(-7 * time.Minute)},
			{ID: 5, Sender: "john_doe", Text: "Great! Let me know if you have any questions.", IsOwn: false, Time: time.Now().Add(-6 * time.Minute)},
			{ID: 6, Sender: "john_doe", Text: "Also, don't forget about the team meeting at 3pm", IsOwn: false, Time: time.Now().Add(-5 * time.Minute)},
			{ID: 7, Sender: "You", Text: "Thanks for the reminder! I'll be there.", IsOwn: true, Time: time.Now().Add(-4 * time.Minute)},
		}
		if len(m.messages) > 0 {
			m.selectedIdx = len(m.messages) - 1
		}
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("tlgram"),
		textinput.Blink,
	)
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.authView.SetSize(msg.Width, msg.Height)
		m.statusBar.SetSize(msg.Width, 1)
		m.input.Width = msg.Width - 4
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case statusbar.ClearNotificationMsg:
		m.statusBar, _ = m.statusBar.Update(msg)
		return m, nil

	case messageSentMsg:
		// Add the sent message to the list
		m.messages = append(m.messages, msg.message)
		m.selectedIdx = len(m.messages) - 1
		cmds = append(cmds, m.statusBar.ShowNotification("Sent!", 2*time.Second))
		return m, tea.Batch(cmds...)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global quit with Ctrl+C
	if key == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}

	// Handle based on current view
	switch m.currentView {
	case ViewAuth:
		return m.handleAuthKey(msg)
	case ViewChatList:
		return m.handleChatListKey(msg)
	case ViewChat:
		return m.handleChatKey(msg)
	case ViewSwitcher:
		return m.handleSwitcherKey(msg)
	}

	return m, nil
}

func (m Model) handleAuthKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "q" {
		m.quitting = true
		return m, tea.Quit
	}

	// In demo mode with no credentials, show info
	if m.demoMode {
		if key == "d" {
			// Switch to demo chat view
			m.currentView = ViewChat
			m.loadDemoData()
			return m, nil
		}
	}

	// Forward to auth view
	var cmd tea.Cmd
	m.authView, cmd = m.authView.Update(msg)
	return m, cmd
}

func (m Model) handleChatListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	result := m.vim.ProcessKey(key)

	switch result.Action {
	case keybind.ActionQuit:
		m.quitting = true
		return m, tea.Quit

	case keybind.ActionMoveDown:
		for i := 0; i < result.Count; i++ {
			if m.chatIdx < len(m.chats)-1 {
				m.chatIdx++
			}
		}

	case keybind.ActionMoveUp:
		for i := 0; i < result.Count; i++ {
			if m.chatIdx > 0 {
				m.chatIdx--
			}
		}

	case keybind.ActionJumpTop:
		m.chatIdx = 0

	case keybind.ActionJumpBottom:
		if len(m.chats) > 0 {
			m.chatIdx = len(m.chats) - 1
		}

	case keybind.ActionSelect:
		if len(m.chats) > 0 {
			m.currentChat = m.chats[m.chatIdx]
			m.currentView = ViewChat
			m.statusBar.SetChatName(m.currentChat.Name)
			m.loadDemoData() // Load messages for selected chat
		}

	case keybind.ActionOpenSwitcher:
		m.prevView = m.currentView
		m.currentView = ViewSwitcher
	}

	return m, nil
}

func (m Model) handleChatKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle insert mode
	if m.vim.Mode() == keybind.ModeInsert {
		return m.handleInsertMode(msg, key)
	}

	// Normal mode
	result := m.vim.ProcessKey(key)

	switch result.Action {
	case keybind.ActionQuit:
		m.quitting = true
		return m, tea.Quit

	case keybind.ActionMoveDown:
		for i := 0; i < result.Count; i++ {
			if m.selectedIdx < len(m.messages)-1 {
				m.selectedIdx++
			}
		}

	case keybind.ActionMoveUp:
		for i := 0; i < result.Count; i++ {
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		}

	case keybind.ActionJumpTop:
		m.selectedIdx = 0

	case keybind.ActionJumpBottom:
		if len(m.messages) > 0 {
			m.selectedIdx = len(m.messages) - 1
		}

	case keybind.ActionHalfPageDown:
		pageSize := (m.height - 6) / 2
		for i := 0; i < pageSize*result.Count; i++ {
			if m.selectedIdx < len(m.messages)-1 {
				m.selectedIdx++
			}
		}

	case keybind.ActionHalfPageUp:
		pageSize := (m.height - 6) / 2
		for i := 0; i < pageSize*result.Count; i++ {
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		}

	case keybind.ActionEnterInsert:
		m.statusBar.SetMode(keybind.ModeInsert)
		return m, m.input.Focus()

	case keybind.ActionReply:
		if len(m.messages) > 0 && m.selectedIdx < len(m.messages) {
			m.replyingTo = m.messages[m.selectedIdx]
			m.vim.SetMode(keybind.ModeInsert)
			m.statusBar.SetMode(keybind.ModeInsert)
			return m, m.input.Focus()
		}

	case keybind.ActionCopy:
		if len(m.messages) > 0 && m.selectedIdx < len(m.messages) {
			// In a real app, copy to clipboard
			return m, m.statusBar.ShowNotification("Copied!", 2*time.Second)
		}

	case keybind.ActionOpenSwitcher:
		m.prevView = m.currentView
		m.currentView = ViewSwitcher

	case keybind.ActionExitToNormal:
		m.input.Blur()
		m.replyingTo = nil
		m.statusBar.SetMode(keybind.ModeNormal)
	}

	m.statusBar.SetMode(m.vim.Mode())
	return m, nil
}

func (m Model) handleInsertMode(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	// Check for send key
	shouldSend := false
	if m.config.General.SendKey == "enter" && key == "enter" {
		shouldSend = true
	} else if m.config.General.SendKey == "ctrl-enter" && key == "ctrl+enter" {
		shouldSend = true
	}

	if shouldSend {
		return m.sendMessage()
	}

	// Check for escape
	if key == "esc" || key == "escape" {
		m.vim.SetMode(keybind.ModeNormal)
		m.statusBar.SetMode(keybind.ModeNormal)
		m.input.Blur()
		m.replyingTo = nil
		return m, nil
	}

	// Forward to text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

type messageSentMsg struct {
	message *Message
}

func (m Model) sendMessage() (Model, tea.Cmd) {
	text := strings.TrimSpace(m.input.Value())
	if text == "" {
		return m, nil
	}

	// Create new message
	newMsg := &Message{
		ID:     int64(len(m.messages) + 1),
		Sender: "You",
		Text:   text,
		IsOwn:  true,
		Time:   time.Now(),
	}

	if m.replyingTo != nil {
		newMsg.ReplyToMsg = m.replyingTo
	}

	// Clear input and reply state
	m.input.Reset()
	m.replyingTo = nil

	// Return to normal mode
	m.vim.SetMode(keybind.ModeNormal)
	m.statusBar.SetMode(keybind.ModeNormal)
	m.input.Blur()

	// Send message (in demo mode, just add to list)
	return m, func() tea.Msg {
		return messageSentMsg{message: newMsg}
	}
}

func (m Model) handleSwitcherKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc", "escape", "ctrl+p":
		m.currentView = m.prevView
		return m, nil

	case "enter":
		// Select chat and switch to it
		if len(m.chats) > 0 && m.chatIdx < len(m.chats) {
			m.currentChat = m.chats[m.chatIdx]
			m.currentView = ViewChat
			m.statusBar.SetChatName(m.currentChat.Name)
			m.loadDemoData()
		}
		return m, nil

	case "j", "down", "ctrl+n":
		if m.chatIdx < len(m.chats)-1 {
			m.chatIdx++
		}

	case "k", "up":
		if m.chatIdx > 0 {
			m.chatIdx--
		}
	}

	return m, nil
}

// View implements tea.Model
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	switch m.currentView {
	case ViewAuth:
		return m.viewAuth()
	case ViewChatList:
		return m.viewChatList()
	case ViewChat:
		return m.viewChat()
	case ViewSwitcher:
		return m.viewSwitcher()
	}

	return "Unknown view"
}

func (m Model) viewAuth() string {
	if m.demoMode {
		return m.viewDemoWelcome()
	}
	return m.authView.View()
}

func (m Model) viewDemoWelcome() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	highlightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("tlgram - Terminal Telegram Client"),
		"",
		highlightStyle.Render("Demo Mode"),
		"",
		"No Telegram API credentials found.",
		"",
		infoStyle.Render("To use with Telegram:"),
		infoStyle.Render("1. Go to https://my.telegram.org/apps"),
		infoStyle.Render("2. Create an application"),
		infoStyle.Render("3. Add api_id and api_hash to ~/.config/tlgram/config.toml"),
		"",
		highlightStyle.Render("Press 'd' to try demo mode"),
		infoStyle.Render("Press 'q' to quit"),
	)
}

func (m Model) viewChatList() string {
	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("Chats"))
	b.WriteString("\n\n")

	// Chat list
	for i, chat := range m.chats {
		style := lipgloss.NewStyle()
		if i == m.chatIdx {
			style = style.Background(lipgloss.Color("237"))
		}

		unread := ""
		if chat.UnreadCount > 0 {
			unread = fmt.Sprintf(" (%d)", chat.UnreadCount)
			style = style.Bold(true)
		}

		line := fmt.Sprintf("  %s%s", chat.Name, unread)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.statusBar.View())

	return b.String()
}

func (m Model) viewChat() string {
	var parts []string

	// Status bar at top
	parts = append(parts, m.statusBar.View())

	// Messages
	msgHeight := m.height - 5 // Room for status bar, input, borders
	if m.replyingTo != nil {
		msgHeight--
	}
	parts = append(parts, m.renderMessages(msgHeight))

	// Reply indicator
	if m.replyingTo != nil {
		replyText := m.replyingTo.Text
		if len(replyText) > 40 {
			replyText = replyText[:40] + "..."
		}
		replyStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Width(m.width)
		parts = append(parts, replyStyle.Render(fmt.Sprintf("↳ Replying to %s: %s", m.replyingTo.Sender, replyText)))
	}

	// Input
	parts = append(parts, m.viewInput())

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) viewSwitcher() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(m.width - 4)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	var content strings.Builder
	content.WriteString(titleStyle.Render("Chat Switcher (Ctrl-p)"))
	content.WriteString("\n\n")

	for i, chat := range m.chats {
		style := lipgloss.NewStyle()
		prefix := "  "
		if i == m.chatIdx {
			style = style.Background(lipgloss.Color("237")).Bold(true)
			prefix = "> "
		}

		unread := ""
		if chat.UnreadCount > 0 {
			unread = fmt.Sprintf(" (%d)", chat.UnreadCount)
		}

		content.WriteString(style.Render(prefix + chat.Name + unread))
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("j/k: navigate • Enter: select • Esc: close"))

	return boxStyle.Render(content.String())
}

func (m Model) renderMessages(maxHeight int) string {
	if len(m.messages) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Height(maxHeight)
		return emptyStyle.Render("No messages yet. Press 'i' to start typing.")
	}

	var lines []string

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("237"))

	senderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39"))

	ownSenderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("42"))

	timeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	replyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	for i, msg := range m.messages {
		// Reply context
		if msg.ReplyToMsg != nil {
			replyText := msg.ReplyToMsg.Text
			if len(replyText) > 30 {
				replyText = replyText[:30] + "..."
			}
			lines = append(lines, replyStyle.Render(fmt.Sprintf("  ↳ %s: %s", msg.ReplyToMsg.Sender, replyText)))
		}

		// Sender
		var sender string
		if msg.IsOwn {
			sender = ownSenderStyle.Render("You")
		} else {
			sender = senderStyle.Render(msg.Sender)
		}

		// Message line
		line := sender + ": " + msg.Text

		// Add time if selected
		if i == m.selectedIdx {
			line += " " + timeStyle.Render(msg.Time.Format("15:04"))
			line = selectedStyle.Render(line)
		}

		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")

	// Ensure it fits in the available height
	return lipgloss.NewStyle().
		Height(maxHeight).
		MaxHeight(maxHeight).
		Render(content)
}

func (m Model) viewInput() string {
	placeholder := "Type a message..."
	if m.vim.Mode() == keybind.ModeNormal {
		placeholder = "Press 'i' to type, 'r' to reply, Ctrl-p for switcher"
	}
	m.input.Placeholder = placeholder

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("241")).
		Width(m.width - 2)

	return inputStyle.Render(m.input.View())
}
