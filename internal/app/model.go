package app

import (
	"fmt"
	"path/filepath"
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
	telegramClient *telegram.Client

	// Chat state
	chats       []*Chat
	messages    []*Message
	selectedIdx int
	chatIdx     int
	currentChat *Chat

	// Switcher state
	switcherInput    textinput.Model
	switcherFiltered []*Chat
	switcherIdx      int

	// Reply state
	replyingTo *Message

	// Error state
	lastError string
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

	// Switcher search input
	switcherTi := textinput.New()
	switcherTi.Placeholder = "Search chats..."
	switcherTi.CharLimit = 100
	switcherTi.Width = 40

	vim := keybind.NewVimState()

	// Check if we have API credentials - if not, we're in demo mode
	demoMode := cfg.Telegram.APIID == 0 || cfg.Telegram.APIHash == ""

	m := Model{
		config:        cfg,
		targetChat:   targetChat,
		vim:          vim,
		demoMode:     demoMode,
		authView:     auth.New(),
		statusBar:    statusbar.New(),
		input:        ti,
		switcherInput: switcherTi,
		chats:        make([]*Chat, 0),
		messages:     make([]*Message, 0),
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
			{ID: 1, Sender: "john_doe", Text: "Hey!", IsOwn: false, Time: time.Now().Add(-10 * time.Minute)},
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

// Message types
type authErrorMsg struct {
	err error
}

type telegramUpdateMsg struct {
	update telegram.Update
}

type chatsLoadedMsg struct {
	chats []*Chat
}

type messagesLoadedMsg struct {
	messages []*Message
}

type messageSentMsg struct {
	message *Message
}

type clientStartedMsg struct {
	client *telegram.Client
}

type clientErrorMsg struct {
	err error
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tea.SetWindowTitle("tlgram"),
		textinput.Blink,
	}

	// If not in demo mode, start the Telegram client
	if !m.demoMode {
		cmds = append(cmds, m.startTelegramClient())
	}

	return tea.Batch(cmds...)
}

func (m *Model) startTelegramClient() tea.Cmd {
	return func() tea.Msg {
		cfg := telegram.DefaultClientConfig(m.config.ConfigDir)
		cfg.APIID = m.config.Telegram.APIID
		cfg.APIHash = m.config.Telegram.APIHash
		cfg.SessionDir = filepath.Join(m.config.ConfigDir, "session")

		client, err := telegram.NewClient(cfg)
		if err != nil {
			return clientErrorMsg{err: err}
		}

		if err := client.Start(); err != nil {
			return clientErrorMsg{err: err}
		}

		return clientStartedMsg{client: client}
	}
}

func (m Model) listenForUpdates() tea.Cmd {
	if m.telegramClient == nil {
		return nil
	}

	return func() tea.Msg {
		update, ok := <-m.telegramClient.Updates()
		if !ok {
			return nil
		}
		return telegramUpdateMsg{update: update}
	}
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

	case clientStartedMsg:
		// Client started, store it and start listening for updates
		m.telegramClient = msg.client
		// Set auth view to wait for phone (client will update via AuthStateUpdate if already authed)
		m.authView.SetState(telegram.AuthStateWaitPhoneNumber)
		return m, m.listenForUpdates()

	case clientErrorMsg:
		m.lastError = msg.err.Error()
		m.authView.SetError(msg.err.Error())
		return m, nil

	case authErrorMsg:
		m.authView.SetError(msg.err.Error())
		m.authView.SetLoading(false)
		return m, nil

	case auth.PhoneSubmitMsg:
		if m.telegramClient == nil {
			m.authView.SetError("client not initialized")
			m.authView.SetLoading(false)
			return m, nil
		}
		return m, func() tea.Msg {
			err := m.telegramClient.SendPhoneNumber(msg.Phone)
			if err != nil {
				return authErrorMsg{err: err}
			}
			return nil
		}

	case auth.CodeSubmitMsg:
		if m.telegramClient == nil {
			m.authView.SetError("client not initialized")
			m.authView.SetLoading(false)
			return m, nil
		}
		return m, func() tea.Msg {
			err := m.telegramClient.SendAuthCode(msg.Code)
			if err != nil {
				return authErrorMsg{err: err}
			}
			return nil
		}

	case auth.PasswordSubmitMsg:
		if m.telegramClient == nil {
			m.authView.SetError("client not initialized")
			m.authView.SetLoading(false)
			return m, nil
		}
		return m, func() tea.Msg {
			err := m.telegramClient.Send2FAPassword(msg.Password)
			if err != nil {
				return authErrorMsg{err: err}
			}
			return nil
		}

	case telegramUpdateMsg:
		newModel, cmd := m.handleTelegramUpdate(msg.update)
		// Continue listening for more updates
		cmds = append(cmds, cmd, m.listenForUpdates())
		return newModel, tea.Batch(cmds...)

	case chatsLoadedMsg:
		m.chats = msg.chats
		if len(m.chats) > 0 {
			m.currentChat = m.chats[0]
			m.statusBar.SetChatName(m.currentChat.Name)
		}
		return m, nil

	case messagesLoadedMsg:
		m.messages = msg.messages
		if len(m.messages) > 0 {
			m.selectedIdx = len(m.messages) - 1
		}
		return m, nil

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

func (m Model) handleTelegramUpdate(update telegram.Update) (Model, tea.Cmd) {
	switch u := update.(type) {
	case telegram.AuthStateUpdate:
		return m.handleAuthStateUpdate(u)

	case telegram.ConnectionStateUpdate:
		m.statusBar.SetConnectionState(u.State)
		return m, nil

	case telegram.NewMessageUpdate:
		// Convert and add new message if it's for the current chat
		if m.currentChat != nil && u.Message.ChatID == m.currentChat.ID {
			msg := convertTelegramMessage(u.Message)
			m.messages = append(m.messages, msg)
			// Auto-scroll if at bottom
			if m.selectedIdx >= len(m.messages)-2 {
				m.selectedIdx = len(m.messages) - 1
			}
		}
		return m, nil

	case telegram.ChatUpdate:
		// Update chat in list
		chat := convertTelegramChat(u.Chat)
		found := false
		for i, c := range m.chats {
			if c.ID == chat.ID {
				m.chats[i] = chat
				found = true
				break
			}
		}
		if !found {
			m.chats = append(m.chats, chat)
		}
		return m, nil

	case telegram.ChatReadUpdate:
		// Update unread count
		for i, c := range m.chats {
			if c.ID == u.ChatID {
				m.chats[i].UnreadCount = 0
				break
			}
		}
		return m, nil

	case telegram.ErrorUpdate:
		// Show error in auth view or status bar
		m.authView.SetError(u.Message)
		m.authView.SetLoading(false)
		m.lastError = u.Message
		return m, nil
	}

	return m, nil
}

func (m Model) handleAuthStateUpdate(u telegram.AuthStateUpdate) (Model, tea.Cmd) {
	switch u.State {
	case telegram.AuthStateWaitPhoneNumber:
		m.authView.SetState(telegram.AuthStateWaitPhoneNumber)

	case telegram.AuthStateWaitCode:
		m.authView.SetState(telegram.AuthStateWaitCode)

	case telegram.AuthStateWaitPassword:
		m.authView.SetState(telegram.AuthStateWaitPassword)

	case telegram.AuthStateReady:
		// Authentication complete, load chats
		m.currentView = ViewChatList
		m.statusBar.SetConnectionState(telegram.ConnectionStateReady)
		return m, m.loadChats()
	}

	return m, nil
}

func (m Model) loadChats() tea.Cmd {
	return func() tea.Msg {
		if m.telegramClient == nil {
			return nil
		}

		chats, err := m.telegramClient.GetChats(100)
		if err != nil {
			return clientErrorMsg{err: err}
		}

		result := make([]*Chat, len(chats))
		for i, c := range chats {
			result[i] = convertTelegramChat(c)
		}

		return chatsLoadedMsg{chats: result}
	}
}

func (m Model) loadMessages(chatID int64) tea.Cmd {
	return func() tea.Msg {
		if m.telegramClient == nil {
			return nil
		}

		messages, err := m.telegramClient.GetChatHistory(chatID, 0, 50)
		if err != nil {
			return clientErrorMsg{err: err}
		}

		result := make([]*Message, len(messages))
		for i, msg := range messages {
			result[i] = convertTelegramMessage(msg)
		}

		return messagesLoadedMsg{messages: result}
	}
}

func convertTelegramChat(c *telegram.Chat) *Chat {
	name := c.Title
	if c.Username != "" {
		name = "@" + c.Username
	}

	lastMsg := ""
	var lastTime time.Time
	if c.LastMessage != nil {
		lastMsg = c.LastMessage.Text
		lastTime = c.LastMessage.Date
	}

	return &Chat{
		ID:          c.ID,
		Name:        name,
		UnreadCount: c.UnreadCount,
		LastMessage: lastMsg,
		LastTime:    lastTime,
	}
}

func convertTelegramMessage(m *telegram.Message) *Message {
	sender := m.SenderName
	if m.IsOutgoing {
		sender = "You"
	}

	return &Message{
		ID:     m.ID,
		Sender: sender,
		Text:   m.Text,
		IsOwn:  m.IsOutgoing,
		Time:   m.Date,
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global quit with Ctrl+C
	if key == "ctrl+c" {
		m.quitting = true
		if m.telegramClient != nil {
			_ = m.telegramClient.Close()
		}
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
		if m.telegramClient != nil {
			_ = m.telegramClient.Close()
		}
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
		if m.telegramClient != nil {
			_ = m.telegramClient.Close()
		}
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

			if m.demoMode {
				m.loadDemoData()
				return m, nil
			}
			return m, m.loadMessages(m.currentChat.ID)
		}

	case keybind.ActionOpenSwitcher:
		m.prevView = m.currentView
		m.currentView = ViewSwitcher
		m.switcherInput.Reset()
		m.switcherInput.Focus()
		m.switcherIdx = 0
	}

	return m, nil
}

// getFilteredChats returns chats filtered by the switcher search input
func (m Model) getFilteredChats() []*Chat {
	query := strings.ToLower(strings.TrimSpace(m.switcherInput.Value()))
	if query == "" {
		return m.chats
	}

	var filtered []*Chat
	for _, chat := range m.chats {
		if strings.Contains(strings.ToLower(chat.Name), query) {
			filtered = append(filtered, chat)
		}
	}
	return filtered
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
		if m.telegramClient != nil {
			_ = m.telegramClient.Close()
		}
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

func (m Model) sendMessage() (Model, tea.Cmd) {
	text := strings.TrimSpace(m.input.Value())
	if text == "" {
		return m, nil
	}

	var replyToID int64
	if m.replyingTo != nil {
		replyToID = m.replyingTo.ID
	}

	// Clear input and reply state
	m.input.Reset()
	replyingTo := m.replyingTo
	m.replyingTo = nil

	// Return to normal mode
	m.vim.SetMode(keybind.ModeNormal)
	m.statusBar.SetMode(keybind.ModeNormal)
	m.input.Blur()

	// Send message
	if m.demoMode {
		// Demo mode: just add to list
		newMsg := &Message{
			ID:     int64(len(m.messages) + 1),
			Sender: "You",
			Text:   text,
			IsOwn:  true,
			Time:   time.Now(),
		}
		if replyingTo != nil {
			newMsg.ReplyToMsg = replyingTo
		}
		return m, func() tea.Msg {
			return messageSentMsg{message: newMsg}
		}
	}

	// Real mode: send via client
	chatID := m.currentChat.ID
	return m, func() tea.Msg {
		if m.telegramClient == nil {
			return nil
		}
		sent, err := m.telegramClient.SendMessage(chatID, text, replyToID)
		if err != nil {
			return clientErrorMsg{err: err}
		}
		return messageSentMsg{message: convertTelegramMessage(sent)}
	}
}

func (m Model) handleSwitcherKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc", "escape":
		m.currentView = m.prevView
		m.switcherInput.Reset()
		m.switcherFiltered = nil
		m.switcherIdx = 0
		return m, nil

	case "enter":
		// Select chat and switch to it
		filtered := m.getFilteredChats()
		if len(filtered) > 0 && m.switcherIdx < len(filtered) {
			m.currentChat = filtered[m.switcherIdx]
			m.currentView = ViewChat
			m.statusBar.SetChatName(m.currentChat.Name)
			m.switcherInput.Reset()
			m.switcherFiltered = nil
			m.switcherIdx = 0

			// Find the index in the main chat list
			for i, c := range m.chats {
				if c.ID == m.currentChat.ID {
					m.chatIdx = i
					break
				}
			}

			if m.demoMode {
				m.loadDemoData()
				return m, nil
			}
			return m, m.loadMessages(m.currentChat.ID)
		}
		return m, nil

	case "down", "ctrl+n":
		filtered := m.getFilteredChats()
		if m.switcherIdx < len(filtered)-1 {
			m.switcherIdx++
		}
		return m, nil

	case "up", "ctrl+p":
		if m.switcherIdx > 0 {
			m.switcherIdx--
		}
		return m, nil

	case "backspace":
		// Let the text input handle backspace
		var cmd tea.Cmd
		m.switcherInput, cmd = m.switcherInput.Update(msg)
		m.switcherIdx = 0 // Reset selection when search changes
		return m, cmd

	default:
		// Forward other keys to text input for searching
		var cmd tea.Cmd
		m.switcherInput, cmd = m.switcherInput.Update(msg)
		m.switcherIdx = 0 // Reset selection when search changes
		return m, cmd
	}
}

// View implements tea.Model
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	// Debug: show current view
	viewName := map[View]string{
		ViewAuth:     "AUTH",
		ViewChatList: "CHATLIST",
		ViewChat:     "CHAT",
		ViewSwitcher: "SWITCHER",
	}[m.currentView]
	debugLine := fmt.Sprintf("[%s | chats:%d | idx:%d]", viewName, len(m.chats), m.chatIdx)

	switch m.currentView {
	case ViewAuth:
		return m.viewAuth()
	case ViewChatList:
		return m.viewChatList() + "\n" + debugLine
	case ViewChat:
		return m.viewChat() + "\n" + debugLine
	case ViewSwitcher:
		return m.viewSwitcher() + "\n" + debugLine
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
	b.WriteString(fmt.Sprintf(" (%d)", len(m.chats)))
	b.WriteString("\n\n")

	// Calculate visible window (show ~20 items centered on selection)
	visibleCount := 20
	if m.height > 10 {
		visibleCount = m.height - 8
	}

	start := m.chatIdx - visibleCount/2
	if start < 0 {
		start = 0
	}
	end := start + visibleCount
	if end > len(m.chats) {
		end = len(m.chats)
		start = end - visibleCount
		if start < 0 {
			start = 0
		}
	}

	// Chat list with scrolling
	for i := start; i < end; i++ {
		chat := m.chats[i]
		style := lipgloss.NewStyle()
		prefix := "  "
		if i == m.chatIdx {
			style = style.Background(lipgloss.Color("237"))
			prefix = "> "
		}

		unread := ""
		if chat.UnreadCount > 0 {
			unread = fmt.Sprintf(" (%d)", chat.UnreadCount)
			style = style.Bold(true)
		}

		line := fmt.Sprintf("%s%s%s", prefix, chat.Name, unread)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.statusBar.View())
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("j/k: navigate • Enter: open • Ctrl+p: search • :q quit"))

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
		Foreground(lipgloss.Color("39"))

	var content strings.Builder
	content.WriteString(titleStyle.Render("Switch Chat"))
	content.WriteString("\n\n")

	// Search input
	content.WriteString(m.switcherInput.View())
	content.WriteString("\n\n")

	// Get filtered chats
	filtered := m.getFilteredChats()

	// Show max 15 chats with scrolling
	visibleCount := 15
	start := m.switcherIdx - visibleCount/2
	if start < 0 {
		start = 0
	}
	end := start + visibleCount
	if end > len(filtered) {
		end = len(filtered)
		start = end - visibleCount
		if start < 0 {
			start = 0
		}
	}

	if len(filtered) == 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("No matching chats"))
		content.WriteString("\n")
	} else {
		for i := start; i < end; i++ {
			chat := filtered[i]
			style := lipgloss.NewStyle()
			prefix := "  "
			if i == m.switcherIdx {
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
	}

	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Type to search • ↑/↓: navigate • Enter: select • Esc: close"))

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
