package app

import (
	"fmt"
	"os/exec"
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

	// Cursor state (line-based navigation)
	cursorLine    int // Which visual line the cursor is on (0-indexed)
	viewportStart int // First visible line in viewport

	// Display preferences
	showUsernames bool // Toggle between full name and @username

	// Loading state
	loadingMore bool

	// New messages indicator (when scrolled up)
	newMsgCount int

	// Error state
	lastError string
}

// Chat represents a chat in the list
type Chat struct {
	ID          int64
	Type        telegram.ChatType
	Name        string
	Username    string
	UnreadCount int
	MemberCount int
	LastMessage string
	LastTime    time.Time
}

// Message represents a chat message
type Message struct {
	ID             int64
	Sender         string // Full name
	SenderUsername string // @username (may be empty)
	Text           string
	IsOwn          bool
	Time           time.Time
	ReplyToMsg     *Message
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
		showUsernames: cfg.Appearance.AuthorDisplay == "username",
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
		m.updateStatusBarForChat()
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

type moreMessagesLoadedMsg struct {
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

type chatSearchedMsg struct {
	chat *Chat
	err  error
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

		// If targetChat is specified, try to find and open it
		if m.targetChat != "" {
			resolved := m.config.ResolveAlias(m.targetChat)
			// Remove @ prefix if present for comparison
			searchName := strings.TrimPrefix(resolved, "@")

			for i, chat := range m.chats {
				chatUsername := strings.TrimPrefix(chat.Name, "@")
				if strings.EqualFold(chatUsername, searchName) ||
					strings.EqualFold(chat.Name, resolved) {
					m.chatIdx = i
					m.currentChat = chat
					m.currentView = ViewChat
					m.updateStatusBarForChat()
					m.input.Blur()
					m.vim.SetMode(keybind.ModeNormal)
					m.statusBar.SetMode(keybind.ModeNormal)
					return m, m.loadMessages(chat.ID)
				}
			}
			// Chat not found in dialogs, try to search for it
			return m, m.searchAndOpenChat(resolved)
		}

		// No targetChat - stay in switcher mode, set first chat as fallback
		if len(m.chats) > 0 {
			m.currentChat = m.chats[0]
			m.updateStatusBarForChat()
		}
		// Focus the switcher input
		m.switcherInput.Focus()
		return m, nil

	case messagesLoadedMsg:
		// Merge instead of replace: keep any messages newer than what was loaded
		// This prevents losing messages that arrived via updates while loading
		if len(m.messages) > 0 && len(msg.messages) > 0 {
			// Find the highest ID in the loaded messages
			maxLoadedID := msg.messages[len(msg.messages)-1].ID

			// Keep messages from current array that are newer (higher ID)
			var newerMessages []*Message
			for _, existing := range m.messages {
				if existing.ID > maxLoadedID {
					newerMessages = append(newerMessages, existing)
				}
			}

			// Merge: loaded messages + newer messages
			if len(newerMessages) > 0 {
				m.messages = append(msg.messages, newerMessages...)
			} else {
				m.messages = msg.messages
			}
		} else {
			m.messages = msg.messages
		}

		if len(m.messages) > 0 {
			m.selectedIdx = len(m.messages) - 1
			// Set cursor to last line (bottom of chat)
			m.cursorLine = m.getTotalLines() - 1
			if m.cursorLine < 0 {
				m.cursorLine = 0
			}
			// Set viewport to show cursor at bottom
			m.adjustViewport()
		}
		m.newMsgCount = 0
		return m, nil

	case moreMessagesLoadedMsg:
		m.loadingMore = false
		if len(msg.messages) > 0 {
			// Calculate lines added by new messages before prepending
			contentWidth := m.width - 4
			if contentWidth < 40 {
				contentWidth = 40
			}
			linesAdded := 0
			for _, newMsg := range msg.messages {
				if newMsg.ReplyToMsg != nil {
					linesAdded++
				}
				wrappedLines := wrapText(newMsg.Text, contentWidth-len(newMsg.Sender)-4)
				linesAdded += len(wrappedLines)
			}

			// Prepend older messages
			m.messages = append(msg.messages, m.messages...)
			// Adjust cursorLine and viewportStart to keep viewing the same content
			m.cursorLine += linesAdded
			m.viewportStart += linesAdded
		}
		return m, nil

	case statusbar.ClearNotificationMsg:
		m.statusBar, _ = m.statusBar.Update(msg)
		return m, nil

	case chatSearchedMsg:
		if msg.err != nil {
			m.lastError = fmt.Sprintf("Chat not found: %s", msg.err.Error())
			// Stay in switcher mode
			return m, nil
		}
		if msg.chat != nil {
			// Add to chats if not already present
			found := false
			for i, c := range m.chats {
				if c.ID == msg.chat.ID {
					m.chatIdx = i
					found = true
					break
				}
			}
			if !found {
				m.chats = append([]*Chat{msg.chat}, m.chats...)
				m.chatIdx = 0
			}
			m.currentChat = msg.chat
			m.currentView = ViewChat
			m.updateStatusBarForChat()
			m.input.Blur()
			m.vim.SetMode(keybind.ModeNormal)
			m.statusBar.SetMode(keybind.ModeNormal)
			return m, m.loadMessages(msg.chat.ID)
		}
		return m, nil

	case messageSentMsg:
		// Add the sent message to the list
		m.messages = append(m.messages, msg.message)
		m.selectedIdx = len(m.messages) - 1
		// Scroll cursor to the sent message (last line)
		m.cursorLine = m.getTotalLines() - 1
		if m.cursorLine < 0 {
			m.cursorLine = 0
		}
		// Adjust viewport to show the sent message
		m.adjustViewport()
		m.newMsgCount = 0
		var cmd tea.Cmd
		m.statusBar, cmd = m.statusBar.ShowNotification("Sent!", 2*time.Second)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case clipboardResultMsg:
		var cmd tea.Cmd
		if msg.success {
			m.statusBar, cmd = m.statusBar.ShowNotification("Copied!", 2*time.Second)
		} else {
			m.statusBar, cmd = m.statusBar.ShowNotification("Copy failed: "+msg.err, 2*time.Second)
		}
		return m, cmd
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

			// Check for duplicate message ID
			isDuplicate := false
			for _, existing := range m.messages {
				if existing.ID == msg.ID {
					isDuplicate = true
					break
				}
			}

			if !isDuplicate {
				// Check if cursor is at the bottom before adding message
				totalLinesBefore := m.getTotalLines()
				wasAtBottom := m.cursorLine >= totalLinesBefore-1

				m.messages = append(m.messages, msg)

				// Only auto-scroll if cursor was already at the bottom
				if wasAtBottom {
					m.cursorLine = m.getTotalLines() - 1
					m.newMsgCount = 0 // Reset counter when at bottom
				} else {
					// Track new message count when scrolled up
					m.newMsgCount++
				}
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
		// Authentication complete, load chats and go to switcher
		m.currentView = ViewSwitcher
		m.switcherInput.Focus()
		m.switcherIdx = 0
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

func (m Model) loadMoreMessages() tea.Cmd {
	if m.telegramClient == nil || m.currentChat == nil || len(m.messages) == 0 {
		return nil
	}

	// Get the oldest message ID to load messages before it
	oldestMsgID := m.messages[0].ID
	chatID := m.currentChat.ID

	return func() tea.Msg {
		messages, err := m.telegramClient.GetChatHistory(chatID, oldestMsgID, 50)
		if err != nil {
			return clientErrorMsg{err: err}
		}

		result := make([]*Message, len(messages))
		for i, msg := range messages {
			result[i] = convertTelegramMessage(msg)
		}

		return moreMessagesLoadedMsg{messages: result}
	}
}

func (m Model) searchAndOpenChat(username string) tea.Cmd {
	if m.telegramClient == nil {
		return nil
	}

	// Remove @ prefix if present
	username = strings.TrimPrefix(username, "@")

	return func() tea.Msg {
		chat, err := m.telegramClient.SearchPublicChat(username)
		if err != nil {
			return chatSearchedMsg{err: err}
		}
		return chatSearchedMsg{chat: convertTelegramChat(chat)}
	}
}

func convertTelegramChat(c *telegram.Chat) *Chat {
	name := c.Title
	// For private chats, use the title (full name) as the primary name
	// For other chats, also use the title

	lastMsg := ""
	var lastTime time.Time
	if c.LastMessage != nil {
		lastMsg = c.LastMessage.Text
		lastTime = c.LastMessage.Date
	}

	return &Chat{
		ID:          c.ID,
		Type:        c.Type,
		Name:        name,
		Username:    c.Username,
		UnreadCount: c.UnreadCount,
		MemberCount: c.MemberCount,
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
		ID:             m.ID,
		Sender:         sender,
		SenderUsername: m.SenderUsername,
		Text:           m.Text,
		IsOwn:          m.IsOutgoing,
		Time:           m.Date,
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// updateStatusBarForChat updates the status bar with the current chat info
func (m *Model) updateStatusBarForChat() {
	if m.currentChat == nil {
		m.statusBar.SetChatName("tlgram")
		return
	}
	m.statusBar.SetChatInfo(
		m.currentChat.Name,
		m.currentChat.Username,
		m.currentChat.Type,
		m.currentChat.MemberCount,
		m.currentChat.UnreadCount,
	)
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

	case keybind.ActionPageDown:
		pageSize := m.height - 8
		if pageSize < 1 {
			pageSize = 10
		}
		for i := 0; i < pageSize*result.Count; i++ {
			if m.chatIdx < len(m.chats)-1 {
				m.chatIdx++
			}
		}

	case keybind.ActionPageUp:
		pageSize := m.height - 8
		if pageSize < 1 {
			pageSize = 10
		}
		for i := 0; i < pageSize*result.Count; i++ {
			if m.chatIdx > 0 {
				m.chatIdx--
			}
		}

	case keybind.ActionHalfPageDown:
		pageSize := (m.height - 8) / 2
		if pageSize < 1 {
			pageSize = 5
		}
		for i := 0; i < pageSize*result.Count; i++ {
			if m.chatIdx < len(m.chats)-1 {
				m.chatIdx++
			}
		}

	case keybind.ActionHalfPageUp:
		pageSize := (m.height - 8) / 2
		if pageSize < 1 {
			pageSize = 5
		}
		for i := 0; i < pageSize*result.Count; i++ {
			if m.chatIdx > 0 {
				m.chatIdx--
			}
		}

	case keybind.ActionSelect:
		if len(m.chats) > 0 {
			m.currentChat = m.chats[m.chatIdx]
			m.currentView = ViewChat
			m.updateStatusBarForChat()

			// Reset input state for new chat
			m.input.Reset()
			m.input.Blur()
			m.replyingTo = nil
			m.vim.SetMode(keybind.ModeNormal)
			m.statusBar.SetMode(keybind.ModeNormal)
			m.selectedIdx = 0

			// Clear messages and reset cursor for new chat
			m.messages = nil
			m.cursorLine = 0

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
		// Search by name
		if strings.Contains(strings.ToLower(chat.Name), query) {
			filtered = append(filtered, chat)
			continue
		}
		// Also search by username for DMs
		if chat.Username != "" && strings.Contains(strings.ToLower(chat.Username), query) {
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
		// Move cursor down by lines
		m.cursorLine += result.Count
		m.adjustViewport()
		// Reset new message counter if at bottom
		totalLines := m.getTotalLines()
		if m.cursorLine >= totalLines-1 {
			m.newMsgCount = 0
		}

	case keybind.ActionMoveUp:
		// Move cursor up by lines
		m.cursorLine -= result.Count
		m.adjustViewport()
		// Load more messages when reaching the top
		if m.cursorLine == 0 && !m.loadingMore && !m.demoMode {
			m.loadingMore = true
			return m, m.loadMoreMessages()
		}

	case keybind.ActionJumpTop:
		m.cursorLine = 0
		m.adjustViewport()
		// Load more messages when jumping to top
		if !m.loadingMore && !m.demoMode {
			m.loadingMore = true
			return m, m.loadMoreMessages()
		}

	case keybind.ActionJumpBottom:
		// Jump to last line
		totalLines := m.getTotalLines()
		if totalLines > 0 {
			m.cursorLine = totalLines - 1
			m.newMsgCount = 0 // Reset new message counter
		}
		m.adjustViewport()

	case keybind.ActionHalfPageDown:
		pageSize := m.getViewportHeight() / 2
		if pageSize < 1 {
			pageSize = 1
		}
		m.cursorLine += pageSize * result.Count
		m.adjustViewport()
		// Reset new message counter if at bottom
		totalLines := m.getTotalLines()
		if m.cursorLine >= totalLines-1 {
			m.newMsgCount = 0
		}

	case keybind.ActionHalfPageUp:
		pageSize := m.getViewportHeight() / 2
		if pageSize < 1 {
			pageSize = 1
		}
		m.cursorLine -= pageSize * result.Count
		m.adjustViewport()
		// Load more messages when reaching the top
		if m.cursorLine == 0 && !m.loadingMore && !m.demoMode {
			m.loadingMore = true
			return m, m.loadMoreMessages()
		}

	case keybind.ActionPageDown:
		pageSize := m.getViewportHeight()
		if pageSize < 1 {
			pageSize = 1
		}
		m.cursorLine += pageSize * result.Count
		m.adjustViewport()
		// Reset new message counter if at bottom
		totalLines := m.getTotalLines()
		if m.cursorLine >= totalLines-1 {
			m.newMsgCount = 0
		}

	case keybind.ActionPageUp:
		pageSize := m.getViewportHeight()
		if pageSize < 1 {
			pageSize = 1
		}
		m.cursorLine -= pageSize * result.Count
		m.adjustViewport()
		// Load more messages when reaching the top
		if m.cursorLine == 0 && !m.loadingMore && !m.demoMode {
			m.loadingMore = true
			return m, m.loadMoreMessages()
		}

	case keybind.ActionViewportTop:
		// H - move cursor to top of visible viewport
		m.cursorLine = m.viewportStart
		// Don't call adjustViewport - we're moving within viewport

	case keybind.ActionViewportBottom:
		// L - move cursor to bottom of visible viewport
		viewportHeight := m.getViewportHeight()
		totalLines := m.getTotalLines()
		lastVisibleLine := m.viewportStart + viewportHeight - 1
		if lastVisibleLine >= totalLines {
			lastVisibleLine = totalLines - 1
		}
		m.cursorLine = lastVisibleLine
		// Reset new message counter if at bottom
		if m.cursorLine >= totalLines-1 {
			m.newMsgCount = 0
		}
		// Don't call adjustViewport - we're moving within viewport

	case keybind.ActionEnterInsert:
		m.statusBar.SetMode(keybind.ModeInsert)
		return m, m.input.Focus()

	case keybind.ActionReply:
		// Reply to message at cursor position
		msgIdx := m.getMessageAtCursor()
		if msgIdx >= 0 && msgIdx < len(m.messages) {
			m.replyingTo = m.messages[msgIdx]
			m.vim.SetMode(keybind.ModeInsert)
			m.statusBar.SetMode(keybind.ModeInsert)
			return m, m.input.Focus()
		}

	case keybind.ActionCopy:
		// Copy message at cursor to clipboard
		msgIdx := m.getMessageAtCursor()
		if msgIdx >= 0 && msgIdx < len(m.messages) {
			msg := m.messages[msgIdx]
			if msg.Text != "" {
				// Copy to clipboard using xclip (Linux)
				return m, m.copyToClipboard(msg.Text)
			}
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification("Cannot copy: no text", 2*time.Second)
			return m, cmd
		}

	case keybind.ActionOpenSwitcher:
		m.prevView = m.currentView
		m.currentView = ViewSwitcher

	case keybind.ActionToggleUsername:
		m.showUsernames = !m.showUsernames
		var cmd tea.Cmd
		if m.showUsernames {
			m.statusBar, cmd = m.statusBar.ShowNotification("Showing @usernames", 1*time.Second)
		} else {
			m.statusBar, cmd = m.statusBar.ShowNotification("Showing full names", 1*time.Second)
		}
		return m, cmd

	case keybind.ActionExitToNormal:
		m.input.Blur()
		m.replyingTo = nil
		m.statusBar.SetMode(keybind.ModeNormal)
	}

	m.statusBar.SetMode(m.vim.Mode())
	return m, nil
}

func (m Model) handleInsertMode(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	// Check for escape first
	if key == "esc" || key == "escape" {
		m.vim.SetMode(keybind.ModeNormal)
		m.statusBar.SetMode(keybind.ModeNormal)
		m.input.Blur()
		m.replyingTo = nil
		return m, nil
	}

	// Check for send key
	shouldSend := false
	sendKey := m.config.General.SendKey

	// Handle "enter" send key
	if sendKey == "enter" && (key == "enter" || msg.Type == tea.KeyEnter) {
		shouldSend = true
	}

	// Handle "ctrl-enter" send key
	// Note: Ctrl+Enter is not consistently supported across terminals
	// Check for various representations
	if sendKey == "ctrl-enter" || sendKey == "ctrl+enter" {
		// Check string representation
		if key == "ctrl+enter" || key == "ctrl-enter" {
			shouldSend = true
		}
		// Also check if it's Enter with Alt modifier (some terminals send this)
		if msg.Type == tea.KeyEnter && msg.Alt {
			shouldSend = true
		}
	}

	if shouldSend && strings.TrimSpace(m.input.Value()) != "" {
		return m.sendMessage()
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

	// Safety check
	if m.currentChat == nil {
		var cmd tea.Cmd
		m.statusBar, cmd = m.statusBar.ShowNotification("No chat selected", 2*time.Second)
		return m, cmd
	}

	var replyToID int64
	if m.replyingTo != nil {
		replyToID = m.replyingTo.ID
	}

	// Clear input and reply state
	m.input.Reset()
	replyingTo := m.replyingTo
	m.replyingTo = nil

	// Stay in insert mode to allow sending multiple messages

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
			m.updateStatusBarForChat()
			m.switcherInput.Reset()
			m.switcherFiltered = nil
			m.switcherIdx = 0

			// Reset input state for new chat
			m.input.Reset()
			m.input.Blur()
			m.replyingTo = nil
			m.vim.SetMode(keybind.ModeNormal)
			m.statusBar.SetMode(keybind.ModeNormal)
			m.selectedIdx = 0

			// Clear messages and reset cursor for new chat
			m.messages = nil
			m.cursorLine = 0

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

	case "ctrl+n":
		filtered := m.getFilteredChats()
		if m.switcherIdx < len(filtered)-1 {
			m.switcherIdx++
		}
		return m, nil

	case "ctrl+p":
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
	// Box width is about 60% of screen width, min 40, max 80
	boxWidth := m.width * 60 / 100
	if boxWidth < 40 {
		boxWidth = 40
	}
	if boxWidth > 80 {
		boxWidth = 80
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(boxWidth)

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
		usernameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
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

			// For DMs, also show username
			displayName := chat.Name
			if chat.Type == telegram.ChatTypePrivate && chat.Username != "" {
				displayName = chat.Name + " " + usernameStyle.Render("@"+chat.Username)
			}

			content.WriteString(style.Render(prefix + displayName + unread))
			content.WriteString("\n")
		}
	}

	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Type to search • Ctrl-n/p: navigate • Enter: select • Esc: close"))

	box := boxStyle.Render(content.String())

	// Center the box in the screen
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// messageLine represents a line and which message it belongs to
type messageLine struct {
	msgIdx    int
	lineType  int    // 0=reply, 1=firstLine, 2=continuation
	sender    string // raw sender name (for first line)
	content   string // message content or reply text
	timestamp string // formatted timestamp (for first line)
	isOwn     bool   // whether this is own message
}

// formatRelativeTime formats a time as relative to now
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	msgDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())

	daysDiff := int(today.Sub(msgDay).Hours() / 24)

	switch {
	case daysDiff == 0:
		// Today - just show time
		return t.Format("15:04")
	case daysDiff == 1:
		// Yesterday
		return "Yesterday " + t.Format("15:04")
	case daysDiff < 7:
		// Within a week
		return fmt.Sprintf("%d days ago %s", daysDiff, t.Format("15:04"))
	default:
		// Older - show date
		return t.Format("2 Jan 15:04")
	}
}

func (m Model) renderMessages(maxHeight int) string {
	if len(m.messages) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Height(maxHeight)
		return emptyStyle.Render("No messages yet. Press 'i' to start typing.")
	}

	// Calculate available width for message text (leave room for sender prefix)
	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	// Build all lines with raw data (styling applied at render time)
	var allLines []messageLine

	for msgIdx, msg := range m.messages {
		// Reply context (if any)
		if msg.ReplyToMsg != nil {
			replySender := msg.ReplyToMsg.Sender
			if m.showUsernames && msg.ReplyToMsg.SenderUsername != "" {
				replySender = msg.ReplyToMsg.SenderUsername
			}
			replyText := msg.ReplyToMsg.Text
			if len(replyText) > 30 {
				replyText = replyText[:30] + "..."
			}
			allLines = append(allLines, messageLine{
				msgIdx:   msgIdx,
				lineType: 0, // reply
				content:  fmt.Sprintf("↳ %s: %s", replySender, replyText),
			})
		}

		// Determine which name to show
		senderName := m.getSenderName(msg)

		// Format timestamp with relative date
		timestamp := formatRelativeTime(msg.Time)

		// Wrap long messages
		wrappedLines := wrapText(msg.Text, contentWidth-len(senderName)-4)

		for i, wline := range wrappedLines {
			if i == 0 {
				// First line with sender and timestamp
				allLines = append(allLines, messageLine{
					msgIdx:    msgIdx,
					lineType:  1, // firstLine
					sender:    senderName,
					content:   wline,
					timestamp: timestamp,
					isOwn:     msg.IsOwn,
				})
			} else {
				// Continuation line
				allLines = append(allLines, messageLine{
					msgIdx:   msgIdx,
					lineType: 2, // continuation
					sender:   senderName,
					content:  wline,
				})
			}
		}
	}

	totalLines := len(allLines)
	viewportHeight := maxHeight - 1 // Leave room for scroll indicator
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	// Use cursor and viewport state from model (local vars for View safety)
	cursorLine := m.cursorLine
	if cursorLine >= totalLines {
		cursorLine = totalLines - 1
	}
	if cursorLine < 0 {
		cursorLine = 0
	}

	// Use viewportStart from model (already adjusted by navigation handlers)
	viewportStart := m.viewportStart

	// Ensure viewport doesn't go past end
	maxViewportStart := totalLines - viewportHeight
	if maxViewportStart < 0 {
		maxViewportStart = 0
	}
	if viewportStart > maxViewportStart {
		viewportStart = maxViewportStart
	}
	if viewportStart < 0 {
		viewportStart = 0
	}

	// Render visible lines
	var visibleLines []string
	viewportEnd := viewportStart + viewportHeight
	if viewportEnd > totalLines {
		viewportEnd = totalLines
	}

	for i := viewportStart; i < viewportEnd; i++ {
		line := allLines[i]
		isCursor := (i == cursorLine)
		renderedLine := m.renderLine(line, contentWidth, isCursor)
		visibleLines = append(visibleLines, renderedLine)
	}

	// Pad with empty lines if needed
	for len(visibleLines) < viewportHeight {
		visibleLines = append(visibleLines, "")
	}

	// Add scroll indicator showing cursor position
	scrollInfo := fmt.Sprintf("─ line %d/%d ", cursorLine+1, totalLines)
	if viewportStart > 0 {
		scrollInfo = "↑ " + scrollInfo
	}
	if viewportEnd < totalLines {
		scrollInfo = scrollInfo + "↓"
	}

	// Add new messages indicator if there are unread messages below
	if m.newMsgCount > 0 {
		newMsgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		scrollInfo = scrollInfo + " " + newMsgStyle.Render(fmt.Sprintf("(%d new)", m.newMsgCount))
	}

	visibleLines = append(visibleLines, lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(scrollInfo))

	return strings.Join(visibleLines, "\n")
}

// renderLine renders a single message line with appropriate styling
func (m Model) renderLine(line messageLine, contentWidth int, isCursor bool) string {
	// Base styles - cursor adds background to all
	cursorBg := lipgloss.Color("237")

	// Build styles based on cursor state
	var senderStyle, ownSenderStyle, timeStyle, replyStyle, textStyle lipgloss.Style

	if isCursor {
		senderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Background(cursorBg)
		ownSenderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")).Background(cursorBg)
		timeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Background(cursorBg)
		replyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true).Background(cursorBg)
		textStyle = lipgloss.NewStyle().Background(cursorBg)
	} else {
		senderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
		ownSenderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
		timeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		replyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
		textStyle = lipgloss.NewStyle()
	}

	var result string

	switch line.lineType {
	case 0: // Reply line
		result = replyStyle.Render(line.content)

	case 1: // First line with sender and timestamp
		var styledSender string
		if line.isOwn {
			styledSender = ownSenderStyle.Render(line.sender)
		} else {
			styledSender = senderStyle.Render(line.sender)
		}

		// Calculate padding for right-aligned timestamp
		baseLen := len(line.sender) + 2 + len(line.content)
		padding := contentWidth - baseLen - len(line.timestamp)
		if padding < 1 {
			padding = 1
		}

		result = styledSender + textStyle.Render(": "+line.content+strings.Repeat(" ", padding)) + timeStyle.Render(line.timestamp)

	case 2: // Continuation line
		indent := strings.Repeat(" ", len(line.sender)+2)
		result = textStyle.Render(indent + line.content)
	}

	// Pad to full width for cursor line
	if isCursor {
		lineWidth := lipgloss.Width(result)
		if lineWidth < m.width {
			result = result + textStyle.Render(strings.Repeat(" ", m.width-lineWidth))
		}
	}

	return result
}

// getSenderName returns the appropriate sender name based on display toggle
func (m Model) getSenderName(msg *Message) string {
	if msg.IsOwn {
		return "You"
	}
	if m.showUsernames && msg.SenderUsername != "" {
		return msg.SenderUsername
	}
	return msg.Sender
}

// getMessageAtCursor returns the message index that contains the cursor line
func (m Model) getMessageAtCursor() int {
	if len(m.messages) == 0 {
		return -1
	}

	// Build line mapping (same logic as renderMessages)
	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	var lineToMsg []int // maps line index to message index
	for msgIdx, msg := range m.messages {
		// Account for reply line
		if msg.ReplyToMsg != nil {
			lineToMsg = append(lineToMsg, msgIdx)
		}
		// Account for message lines
		senderName := m.getSenderName(msg)
		wrappedLines := wrapText(msg.Text, contentWidth-len(senderName)-4)
		for range wrappedLines {
			lineToMsg = append(lineToMsg, msgIdx)
		}
	}

	// Clamp cursorLine
	cursorLine := m.cursorLine
	if cursorLine >= len(lineToMsg) {
		cursorLine = len(lineToMsg) - 1
	}
	if cursorLine < 0 {
		cursorLine = 0
	}

	if cursorLine < len(lineToMsg) {
		return lineToMsg[cursorLine]
	}
	return len(m.messages) - 1 // Default to last message
}

// getTotalLines returns the total number of visual lines for all messages
func (m Model) getTotalLines() int {
	if len(m.messages) == 0 {
		return 0
	}

	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	totalLines := 0
	for _, msg := range m.messages {
		if msg.ReplyToMsg != nil {
			totalLines++
		}
		senderName := m.getSenderName(msg)
		wrappedLines := wrapText(msg.Text, contentWidth-len(senderName)-4)
		totalLines += len(wrappedLines)
	}
	return totalLines
}

// getViewportHeight returns the number of visible lines in the message viewport
func (m Model) getViewportHeight() int {
	// Height minus status bar (1), input area (~3), scroll indicator (1)
	vh := m.height - 6
	if vh < 1 {
		vh = 1
	}
	return vh
}

// adjustViewport ensures cursor is visible and adjusts viewportStart if needed
func (m *Model) adjustViewport() {
	totalLines := m.getTotalLines()
	viewportHeight := m.getViewportHeight()

	// Clamp cursor to valid range
	if m.cursorLine < 0 {
		m.cursorLine = 0
	}
	if m.cursorLine >= totalLines {
		m.cursorLine = totalLines - 1
	}
	if m.cursorLine < 0 {
		m.cursorLine = 0
	}

	// Only scroll viewport if cursor is outside visible area
	if m.cursorLine < m.viewportStart {
		// Cursor above viewport - scroll up
		m.viewportStart = m.cursorLine
	} else if m.cursorLine >= m.viewportStart+viewportHeight {
		// Cursor below viewport - scroll down
		m.viewportStart = m.cursorLine - viewportHeight + 1
	}

	// Clamp viewport to valid range
	maxViewportStart := totalLines - viewportHeight
	if maxViewportStart < 0 {
		maxViewportStart = 0
	}
	if m.viewportStart > maxViewportStart {
		m.viewportStart = maxViewportStart
	}
	if m.viewportStart < 0 {
		m.viewportStart = 0
	}
}

// clipboardResultMsg is sent when clipboard operation completes
type clipboardResultMsg struct {
	success bool
	err     string
}

// copyToClipboard copies text to the system clipboard
func (m Model) copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		// Try xclip first (X11), then xsel, then wl-copy (Wayland)
		cmds := []struct {
			name string
			args []string
		}{
			{"xclip", []string{"-selection", "clipboard"}},
			{"xsel", []string{"--clipboard", "--input"}},
			{"wl-copy", []string{}},
		}

		for _, c := range cmds {
			cmd := exec.Command(c.name, c.args...)
			cmd.Stdin = strings.NewReader(text)
			if err := cmd.Run(); err == nil {
				return clipboardResultMsg{success: true}
			}
		}

		return clipboardResultMsg{success: false, err: "no clipboard tool found"}
	}
}

// wrapText wraps text to the specified width, handling explicit newlines
func wrapText(text string, width int) []string {
	if width <= 0 {
		width = 80
	}

	// First split by explicit newlines
	paragraphs := strings.Split(text, "\n")
	var lines []string

	for _, para := range paragraphs {
		// Handle empty lines (preserve blank lines in messages)
		if para == "" {
			lines = append(lines, "")
			continue
		}

		// Wrap this paragraph
		for len(para) > 0 {
			if len(para) <= width {
				lines = append(lines, para)
				break
			}

			// Find a good break point
			breakAt := width
			for breakAt > width/2 {
				if para[breakAt] == ' ' {
					break
				}
				breakAt--
			}
			if breakAt <= width/2 {
				breakAt = width // No good break point, just cut
			}

			lines = append(lines, para[:breakAt])
			para = strings.TrimLeft(para[breakAt:], " ")
		}
	}

	// Ensure at least one line
	if len(lines) == 0 {
		lines = append(lines, "")
	}

	return lines
}

func (m Model) viewInput() string {
	placeholder := "Type a message..."
	if m.vim.Mode() == keybind.ModeNormal {
		placeholder = "i: type, r: reply, yy: copy, u: toggle names, Ctrl-p: switcher"
	}
	m.input.Placeholder = placeholder

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("241")).
		Width(m.width - 2)

	return inputStyle.Render(m.input.View())
}
