package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
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
	ViewSetup View = iota // No credentials configured
	ViewAuth              // Waiting for authentication
	ViewChatList
	ViewChat
	ViewSwitcher
	ViewHelp
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

	// Sub-components
	authView  auth.Model
	statusBar statusbar.Model
	input     textarea.Model

	// Telegram client
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

	// Edit state
	editingMessage *Message // Message being edited (nil if composing new)

	// Cursor state (line-based navigation)
	cursorLine    int // Which visual line the cursor is on (0-indexed)
	viewportStart int // First visible line in viewport

	// Jump history for navigating to original messages and back
	jumpStack []int // Stack of cursor positions to return to

	// Display preferences
	showUsernames bool // Toggle between full name and @username
	showChatID    bool // Toggle chat ID display in header

	// Notification state (all runtime-only, not persisted)
	globalMute   bool           // Mute all notifications (toggle with 'M')
	mutedChats   map[int64]bool // Muted chats (toggle with 'm')
	watchedChats map[int64]bool // Watched chats - always notify (toggle with 'w')

	// Loading state
	loadingMore bool

	// New messages indicator (when scrolled up)
	newMsgCount   int   // count of new messages
	firstNewMsgID int64 // ID of first "new" message (to track which are new)

	// Auto-mark-as-read state
	autoMarkPending     bool  // Timer is pending for auto-mark
	autoMarkPendingChat int64 // Chat ID for pending auto-mark

	// Error state
	lastError string
}

// Chat represents a chat in the list
type Chat struct {
	ID              int64
	Type            telegram.ChatType
	Name            string
	Username        string
	UnreadCount     int
	MemberCount     int
	LastMessage     string
	LastTime        time.Time
	LastReadInboxID int64 // Last read message ID (for unread styling)
	MarkedUnread    bool  // UI flag for "remind me" unread status
}

// Message represents a chat message
type Message struct {
	ID             int64
	Sender         string // Full name
	SenderUsername string // @username (may be empty)
	Text           string
	IsOwn          bool
	Time           time.Time
	ReplyToID      int64               // ID of message this is replying to (0 if not a reply)
	Reactions      []telegram.Reaction // Reactions on this message
}

// New creates a new application model
func New(cfg *config.Config, targetChat string) Model {
	// Message input (multi-line textarea)
	ti := textarea.New()
	ti.Placeholder = "Type a message..."
	ti.CharLimit = 4096
	ti.SetHeight(3) // Fixed height, scrolls internally for longer messages
	ti.ShowLineNumbers = false
	// Disable cursor line highlighting (too bright by default)
	ti.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ti.BlurredStyle.CursorLine = lipgloss.NewStyle()

	// Switcher search input
	switcherTi := textinput.New()
	switcherTi.Placeholder = "Search chats..."
	switcherTi.CharLimit = 100
	switcherTi.Width = 40

	vim := keybind.NewVimState()

	// Check if we have API credentials
	hasCredentials := cfg.Telegram.APIID != 0 && cfg.Telegram.APIHash != ""

	m := Model{
		config:        cfg,
		targetChat:    targetChat,
		vim:           vim,
		authView:      auth.New(),
		statusBar:     statusbar.New(),
		input:         ti,
		switcherInput: switcherTi,
		chats:         make([]*Chat, 0),
		messages:      make([]*Message, 0),
		showUsernames: cfg.Appearance.AuthorDisplay == "username",
		showChatID:    cfg.Appearance.ShowChatID,
		globalMute:    cfg.Notification.StartMuted,
		mutedChats:    make(map[int64]bool),
		watchedChats:  make(map[int64]bool),
	}

	// Show setup screen if no credentials, otherwise auth screen
	if hasCredentials {
		m.currentView = ViewAuth
	} else {
		m.currentView = ViewSetup
	}

	return m
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

type markReadCompleteMsg struct {
	chatID   int64
	maxMsgID int64
	err      error
}

type markUnreadCompleteMsg struct {
	chatID int64
	err    error
}

type markFullyReadCompleteMsg struct {
	chatID int64
	err    error
}

type autoMarkReadTickMsg struct {
	chatID int64 // Which chat this tick is for (to ignore stale ticks)
}

type editMessageCompleteMsg struct {
	chatID    int64
	messageID int64
	newText   string
	err       error
}

type deleteMessageCompleteMsg struct {
	chatID    int64
	messageID int64
	err       error
}

type reactionSentMsg struct {
	chatID    int64
	messageID int64
	emoji     string
	added     bool // True if reaction was added, false if removed
	err       error
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
		textarea.Blink,
	}

	// Only start the Telegram client if we have credentials
	if m.currentView != ViewSetup {
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
		m.input.SetWidth(msg.Width - 4)
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
		// Sort chats by most recent activity
		sort.Slice(m.chats, func(i, j int) bool {
			return m.chats[i].LastTime.After(m.chats[j].LastTime)
		})

		// If targetChat is specified, try to find and open it
		if m.targetChat != "" {
			resolved := m.config.ResolveAlias(m.targetChat)

			// Check if resolved is a numeric chat ID
			var targetID int64
			if id, err := strconv.ParseInt(resolved, 10, 64); err == nil {
				targetID = id
			}

			// Remove @ prefix if present for comparison
			searchName := strings.TrimPrefix(resolved, "@")

			for i, chat := range m.chats {
				// Match by numeric ID
				if targetID != 0 && chat.ID == targetID {
					m.chatIdx = i
					m.currentChat = chat
					m.currentView = ViewChat
					m.updateStatusBarForChat()
					m.input.Blur()
					m.vim.SetMode(keybind.ModeNormal)
					m.statusBar.SetMode(keybind.ModeNormal)
					return m, m.loadMessages(chat.ID)
				}
				// Match by username or name
				chatUsername := strings.TrimPrefix(chat.Username, "@")
				if strings.EqualFold(chatUsername, searchName) ||
					strings.EqualFold(chat.Name, resolved) ||
					strings.EqualFold(chat.Username, searchName) {
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
			// Chat not found in dialogs, try to search for it (only works for usernames)
			if targetID == 0 {
				return m, m.searchAndOpenChat(resolved)
			}
			// Numeric ID not found - show error
			m.statusBar.ShowNotification("Chat not found: "+resolved, 3*time.Second)
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
		m.firstNewMsgID = 0

		// Start auto-mark-as-read timer if configured
		cmd := m.startAutoMarkReadTimer()
		return m, cmd

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
				if newMsg.ReplyToID > 0 {
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
		m.firstNewMsgID = 0
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

	case markReadCompleteMsg:
		var cmd tea.Cmd
		if msg.err != nil {
			m.statusBar, cmd = m.statusBar.ShowNotification("Error: "+msg.err.Error(), 3*time.Second)
		} else {
			// Count remaining unread messages (after the read position, incoming only)
			remainingUnread := 0
			for _, message := range m.messages {
				if message.ID > msg.maxMsgID && !message.IsOwn {
					remainingUnread++
				}
			}

			// Update local state
			for i, c := range m.chats {
				if c.ID == msg.chatID {
					m.chats[i].LastReadInboxID = msg.maxMsgID
					m.chats[i].UnreadCount = remainingUnread
					m.chats[i].MarkedUnread = false // Clear "marked unread" flag when reading
					if m.currentChat != nil && m.currentChat.ID == msg.chatID {
						m.currentChat.LastReadInboxID = msg.maxMsgID
						m.currentChat.UnreadCount = remainingUnread
						m.currentChat.MarkedUnread = false
					}
					break
				}
			}

			// Update status bar with new unread count
			m.updateStatusBarForChat()

			// Also update new message counter if we were tracking new messages
			if m.newMsgCount > 0 {
				m.newMsgCount = remainingUnread
				if m.newMsgCount == 0 {
					m.firstNewMsgID = 0
				}
			}

			m.statusBar, cmd = m.statusBar.ShowNotification("Marked as read", 2*time.Second)
		}
		return m, cmd

	case markUnreadCompleteMsg:
		var cmd tea.Cmd
		if msg.err != nil {
			m.statusBar, cmd = m.statusBar.ShowNotification("Error: "+msg.err.Error(), 3*time.Second)
		} else {
			m.statusBar, cmd = m.statusBar.ShowNotification("Marked as unread", 2*time.Second)
			// Update local state
			for i, c := range m.chats {
				if c.ID == msg.chatID {
					m.chats[i].MarkedUnread = true
					if m.currentChat != nil && m.currentChat.ID == msg.chatID {
						m.currentChat.MarkedUnread = true
					}
					break
				}
			}
		}
		return m, cmd

	case markFullyReadCompleteMsg:
		var cmd tea.Cmd
		if msg.err != nil {
			m.statusBar, cmd = m.statusBar.ShowNotification("Error: "+msg.err.Error(), 3*time.Second)
		} else {
			m.statusBar, cmd = m.statusBar.ShowNotification("Marked as read", 2*time.Second)
			// Find max message ID to update LastReadInboxID
			var maxMsgID int64
			for _, message := range m.messages {
				if message.ID > maxMsgID {
					maxMsgID = message.ID
				}
			}
			// Update local state - clear unread count and marked unread flag
			for i, c := range m.chats {
				if c.ID == msg.chatID {
					m.chats[i].UnreadCount = 0
					m.chats[i].MarkedUnread = false
					m.chats[i].LastReadInboxID = maxMsgID
					if m.currentChat != nil && m.currentChat.ID == msg.chatID {
						m.currentChat.UnreadCount = 0
						m.currentChat.MarkedUnread = false
						m.currentChat.LastReadInboxID = maxMsgID
					}
					break
				}
			}
		}
		return m, cmd

	case autoMarkReadTickMsg:
		// Ignore stale tick messages (from a different chat or already canceled)
		if !m.autoMarkPending || msg.chatID != m.autoMarkPendingChat {
			return m, nil
		}
		m.autoMarkPending = false

		// Mark all messages in the current chat as read
		if m.currentChat != nil && m.currentChat.ID == msg.chatID {
			return m, m.markChatFullyRead(msg.chatID)
		}
		return m, nil

	case editMessageCompleteMsg:
		var cmd tea.Cmd
		if msg.err != nil {
			m.statusBar, cmd = m.statusBar.ShowNotification("Edit failed: "+msg.err.Error(), 3*time.Second)
		} else {
			m.statusBar, cmd = m.statusBar.ShowNotification("Message edited", 2*time.Second)
			// Update local message text
			for i, message := range m.messages {
				if message.ID == msg.messageID {
					m.messages[i].Text = msg.newText
					break
				}
			}
		}
		return m, cmd

	case deleteMessageCompleteMsg:
		var cmd tea.Cmd
		if msg.err != nil {
			m.statusBar, cmd = m.statusBar.ShowNotification("Delete failed: "+msg.err.Error(), 3*time.Second)
		} else {
			m.statusBar, cmd = m.statusBar.ShowNotification("Message deleted", 2*time.Second)
			// Remove message from local list
			for i, message := range m.messages {
				if message.ID == msg.messageID {
					m.messages = append(m.messages[:i], m.messages[i+1:]...)
					// Adjust cursor if needed
					if m.cursorLine >= m.getTotalLines() && m.cursorLine > 0 {
						m.cursorLine = m.getTotalLines() - 1
					}
					break
				}
			}
		}
		return m, cmd

	case reactionSentMsg:
		var cmd tea.Cmd
		if msg.err != nil {
			m.statusBar, cmd = m.statusBar.ShowNotification("Reaction failed: "+msg.err.Error(), 3*time.Second)
		} else {
			// Update local message state with the reaction change
			// NOTE: Non-Premium users can only have ONE reaction per message.
			// Adding a new reaction REPLACES any existing chosen reaction.
			for i, message := range m.messages {
				if message.ID == msg.messageID {
					hadReactions := len(m.messages[i].Reactions) > 0
					if msg.added && msg.emoji != "" {
						// Add/replace reaction - first remove any existing chosen reaction
						newReactions := make([]telegram.Reaction, 0, len(m.messages[i].Reactions))
						for _, r := range m.messages[i].Reactions {
							if r.Chosen {
								// Remove our previous reaction
								if r.Count > 1 {
									r.Count--
									r.Chosen = false
									newReactions = append(newReactions, r)
								}
								// else: was only us, remove entirely
							} else {
								newReactions = append(newReactions, r)
							}
						}
						// Now add the new reaction
						found := false
						for j, r := range newReactions {
							if r.Emoji == msg.emoji {
								newReactions[j].Count++
								newReactions[j].Chosen = true
								found = true
								break
							}
						}
						if !found {
							newReactions = append(newReactions, telegram.Reaction{
								Emoji:  msg.emoji,
								Count:  1,
								Chosen: true,
							})
						}
						m.messages[i].Reactions = newReactions
					} else if !msg.added {
						// Remove reaction (toggle off or 0 key)
						newReactions := make([]telegram.Reaction, 0, len(m.messages[i].Reactions))
						for _, r := range m.messages[i].Reactions {
							shouldRemove := false
							if msg.emoji != "" {
								// Remove specific emoji
								shouldRemove = r.Emoji == msg.emoji && r.Chosen
							} else {
								// Remove all chosen (0 key)
								shouldRemove = r.Chosen
							}
							if shouldRemove {
								if r.Count > 1 {
									r.Count--
									r.Chosen = false
									newReactions = append(newReactions, r)
								}
								// else: remove entirely
							} else {
								newReactions = append(newReactions, r)
							}
						}
						m.messages[i].Reactions = newReactions
					}
					// Adjust viewport if reactions changed on this message
					hasReactions := len(m.messages[i].Reactions) > 0
					if hadReactions != hasReactions {
						// Reaction line was added or removed, need to adjust
						totalLines := m.getTotalLines()
						viewportHeight := m.getViewportHeight()
						maxViewportStart := totalLines - viewportHeight
						if maxViewportStart < 0 {
							maxViewportStart = 0
						}
						// If we're at the bottom, stay at bottom
						if m.viewportStart > maxViewportStart {
							m.viewportStart = maxViewportStart
						}
						// Ensure cursor is visible
						m.adjustViewport()
					}
					break
				}
			}
			if msg.added {
				m.statusBar, cmd = m.statusBar.ShowNotification("Reacted with "+msg.emoji, 1*time.Second)
			} else {
				m.statusBar, cmd = m.statusBar.ShowNotification("Reaction removed", 1*time.Second)
			}
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
		msg := convertTelegramMessage(u.Message)
		isCurrentChat := m.currentChat != nil && u.Message.ChatID == m.currentChat.ID

		// Update chat list with new message info
		for i, c := range m.chats {
			if c.ID == u.Message.ChatID {
				// Truncate message for preview
				preview := msg.Text
				if len(preview) > 50 {
					preview = preview[:47] + "..."
				}
				m.chats[i].LastMessage = preview
				m.chats[i].LastTime = msg.Time

				// Increment unread count if not the current open chat and not own message
				if !isCurrentChat && !msg.IsOwn {
					m.chats[i].UnreadCount++
				}
				break
			}
		}

		// Re-sort chats by most recent activity
		m.sortChatsByRecent()

		// Send notification if:
		// - Watched chats: always notify (even if currently viewing)
		// - Other chats: notify if not current chat, not globally muted, not per-chat muted
		// - Never notify for own messages
		isWatched := m.watchedChats[u.Message.ChatID]
		isMuted := m.mutedChats[u.Message.ChatID]
		shouldNotify := !msg.IsOwn && (isWatched || (!isCurrentChat && !m.globalMute && !isMuted))
		if shouldNotify {
			m.sendNotification(msg.Sender, msg.Text)
		}

		// Add to message list if it's for the current chat
		if isCurrentChat {
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
					m.firstNewMsgID = 0
					m.adjustViewport() // Actually scroll the viewport
				} else {
					// Track new message count when scrolled up
					if m.newMsgCount == 0 {
						m.firstNewMsgID = msg.ID // Track where "new" messages start
					}
					m.newMsgCount++
				}

				// (Re)start auto-mark timer for incoming messages
				if !msg.IsOwn {
					cmd := m.startAutoMarkReadTimer()
					return m, cmd
				}
			}
		}
		return m, nil

	case telegram.MessageEditedUpdate:
		// Update message text and/or reactions
		if m.currentChat != nil && m.currentChat.ID == u.ChatID {
			for i, msg := range m.messages {
				if msg.ID == u.MessageID {
					// Update text if changed
					if u.NewText != "" && u.NewText != m.messages[i].Text {
						m.messages[i].Text = u.NewText
					}
					// Update reactions if provided
					if u.Reactions != nil {
						hadReactions := len(m.messages[i].Reactions) > 0
						m.messages[i].Reactions = u.Reactions
						hasReactions := len(m.messages[i].Reactions) > 0
						// Adjust viewport if reactions changed (line count may have changed)
						if hadReactions != hasReactions {
							totalLines := m.getTotalLines()
							viewportHeight := m.getViewportHeight()
							maxViewportStart := totalLines - viewportHeight
							if maxViewportStart < 0 {
								maxViewportStart = 0
							}
							if m.viewportStart > maxViewportStart {
								m.viewportStart = maxViewportStart
							}
							m.adjustViewport()
						}
					}
					break
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

	case telegram.DialogUnreadUpdate:
		// Update unread mark state
		for i, c := range m.chats {
			if c.ID == u.ChatID {
				m.chats[i].MarkedUnread = u.MarkedUnread
				if m.currentChat != nil && m.currentChat.ID == u.ChatID {
					m.currentChat.MarkedUnread = u.MarkedUnread
				}
				break
			}
		}
		return m, nil

	case telegram.MessageReactionsUpdate:
		// Update reactions on a message (from server, e.g., other clients)
		if m.currentChat != nil && m.currentChat.ID == u.ChatID {
			for i, msg := range m.messages {
				if msg.ID == u.MessageID {
					hadReactions := len(m.messages[i].Reactions) > 0
					m.messages[i].Reactions = u.Reactions
					hasReactions := len(m.messages[i].Reactions) > 0
					// Adjust viewport if reactions changed (line count may have changed)
					if hadReactions != hasReactions {
						totalLines := m.getTotalLines()
						viewportHeight := m.getViewportHeight()
						maxViewportStart := totalLines - viewportHeight
						if maxViewportStart < 0 {
							maxViewportStart = 0
						}
						if m.viewportStart > maxViewportStart {
							m.viewportStart = maxViewportStart
						}
						m.adjustViewport()
					}
					break
				}
			}
		}
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

func (m Model) markMessagesAsRead(chatID int64, maxMsgID int64) tea.Cmd {
	if m.telegramClient == nil {
		return nil
	}

	return func() tea.Msg {
		// Clear the "marked unread" flag when marking as read
		_ = m.telegramClient.MarkDialogUnread(chatID, false)
		err := m.telegramClient.MarkAsRead(chatID, maxMsgID)
		return markReadCompleteMsg{chatID: chatID, maxMsgID: maxMsgID, err: err}
	}
}

func (m Model) markDialogUnread(chatID int64) tea.Cmd {
	if m.telegramClient == nil {
		return nil
	}

	return func() tea.Msg {
		err := m.telegramClient.MarkDialogUnread(chatID, true)
		return markUnreadCompleteMsg{chatID: chatID, err: err}
	}
}

func (m Model) markChatFullyRead(chatID int64) tea.Cmd {
	if m.telegramClient == nil {
		return nil
	}

	return func() tea.Msg {
		// Clear the "marked unread" flag first
		_ = m.telegramClient.MarkDialogUnread(chatID, false)
		// Mark all messages as read using max int32 as message ID
		err := m.telegramClient.MarkAsRead(chatID, int64(^uint32(0)>>1))
		return markFullyReadCompleteMsg{chatID: chatID, err: err}
	}
}

// sortChatsByRecent sorts chats by most recent activity (LastTime descending)
func (m *Model) sortChatsByRecent() {
	sort.Slice(m.chats, func(i, j int) bool {
		return m.chats[i].LastTime.After(m.chats[j].LastTime)
	})
}

// sendNotification executes the notification command with sender and message
func (m Model) sendNotification(senderName, messagePreview string) {
	cmd := m.config.Notification.Command
	if cmd == "" {
		return // Notifications disabled
	}

	// Truncate message preview
	if len(messagePreview) > 50 {
		messagePreview = messagePreview[:47] + "..."
	}

	// Escape single quotes in sender and message to prevent shell injection
	senderName = strings.ReplaceAll(senderName, "'", "'\\''")
	messagePreview = strings.ReplaceAll(messagePreview, "'", "'\\''")

	// Replace template variables
	cmd = strings.ReplaceAll(cmd, "%s", senderName)
	cmd = strings.ReplaceAll(cmd, "%m", messagePreview)

	// Execute command asynchronously, wiring stdout/stderr to current process
	// This allows commands like "echo -e '\a'" to trigger the terminal bell
	go func() {
		c := exec.Command("sh", "-c", cmd)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		_ = c.Run()
	}()
}

func (m Model) editMessage(chatID int64, messageID int64, newText string) tea.Cmd {
	if m.telegramClient == nil {
		return nil
	}

	return func() tea.Msg {
		err := m.telegramClient.EditMessage(chatID, messageID, newText)
		return editMessageCompleteMsg{chatID: chatID, messageID: messageID, newText: newText, err: err}
	}
}

func (m Model) deleteMessage(chatID int64, messageID int64) tea.Cmd {
	if m.telegramClient == nil {
		return nil
	}

	return func() tea.Msg {
		err := m.telegramClient.DeleteMessages(chatID, []int64{messageID}, true)
		return deleteMessageCompleteMsg{chatID: chatID, messageID: messageID, err: err}
	}
}

func (m Model) sendReaction(chatID int64, messageID int64, emoji string) tea.Cmd {
	if m.telegramClient == nil {
		return nil
	}

	return func() tea.Msg {
		result, err := m.telegramClient.SendReaction(chatID, int(messageID), emoji)
		msg := reactionSentMsg{chatID: chatID, messageID: messageID, err: err}
		if result != nil {
			msg.emoji = result.Emoji
			msg.added = result.Added
		}
		return msg
	}
}

// startAutoMarkReadTimer starts a timer to auto-mark messages as read
// Returns nil if auto-mark is disabled or there are no unread messages
func (m *Model) startAutoMarkReadTimer() tea.Cmd {
	delay := m.config.General.AutoMarkReadSecs
	if delay < 0 || m.currentChat == nil {
		return nil // Disabled or no chat
	}

	// Check if there are any unread messages
	if m.currentChat.UnreadCount == 0 && !m.currentChat.MarkedUnread {
		return nil // No unread messages
	}

	chatID := m.currentChat.ID
	m.autoMarkPending = true
	m.autoMarkPendingChat = chatID

	// Instant mode (0 delay)
	if delay == 0 {
		return func() tea.Msg {
			return autoMarkReadTickMsg{chatID: chatID}
		}
	}

	// Delayed mode
	return tea.Tick(time.Duration(delay)*time.Second, func(t time.Time) tea.Msg {
		return autoMarkReadTickMsg{chatID: chatID}
	})
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
		ID:              c.ID,
		Type:            c.Type,
		Name:            name,
		Username:        c.Username,
		UnreadCount:     c.UnreadCount,
		MemberCount:     c.MemberCount,
		LastMessage:     lastMsg,
		LastTime:        lastTime,
		LastReadInboxID: c.LastReadInboxID,
		MarkedUnread:    c.MarkedUnread,
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
		ReplyToID:      m.ReplyToID,
		Reactions:      m.Reactions,
	}
}

// updateStatusBarForChat updates the status bar with the current chat info
func (m *Model) updateStatusBarForChat() {
	if m.currentChat == nil {
		m.statusBar.SetChatName("tlgram")
		m.statusBar.SetChatID(0)
		m.statusBar.SetShowChatID(false)
		m.statusBar.SetNotifyState(statusbar.NotifyStateNone)
		return
	}
	m.statusBar.SetChatInfo(
		m.currentChat.Name,
		m.currentChat.Username,
		m.currentChat.Type,
		m.currentChat.MemberCount,
		m.currentChat.UnreadCount,
	)
	m.statusBar.SetChatID(m.currentChat.ID)
	m.statusBar.SetShowChatID(m.showChatID)
	m.statusBar.SetNotifyState(m.getNotifyState(m.currentChat.ID))
}

// getNotifyState returns the notification state for a chat
func (m *Model) getNotifyState(chatID int64) statusbar.NotifyState {
	if m.watchedChats[chatID] {
		return statusbar.NotifyStateWatched
	}
	if m.mutedChats[chatID] {
		return statusbar.NotifyStateMuted
	}
	return statusbar.NotifyStateNone
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
	case ViewSetup:
		return m.handleSetupKey(msg)
	case ViewAuth:
		return m.handleAuthKey(msg)
	case ViewChatList:
		return m.handleChatListKey(msg)
	case ViewChat:
		return m.handleChatKey(msg)
	case ViewSwitcher:
		return m.handleSwitcherKey(msg)
	case ViewHelp:
		return m.handleHelpKey(msg)
	}

	return m, nil
}

func (m Model) handleSetupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "q" {
		m.quitting = true
		return m, tea.Quit
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
		if m.chatIdx < len(m.chats)-1 {
			m.chatIdx++
		}

	case keybind.ActionMoveUp:
		if m.chatIdx > 0 {
			m.chatIdx--
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
		m.chatIdx += pageSize
		if m.chatIdx >= len(m.chats) {
			m.chatIdx = len(m.chats) - 1
		}

	case keybind.ActionPageUp:
		pageSize := m.height - 8
		if pageSize < 1 {
			pageSize = 10
		}
		m.chatIdx -= pageSize
		if m.chatIdx < 0 {
			m.chatIdx = 0
		}

	case keybind.ActionHalfPageDown:
		pageSize := (m.height - 8) / 2
		if pageSize < 1 {
			pageSize = 5
		}
		m.chatIdx += pageSize
		if m.chatIdx >= len(m.chats) {
			m.chatIdx = len(m.chats) - 1
		}

	case keybind.ActionHalfPageUp:
		pageSize := (m.height - 8) / 2
		if pageSize < 1 {
			pageSize = 5
		}
		m.chatIdx -= pageSize
		if m.chatIdx < 0 {
			m.chatIdx = 0
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
			m.newMsgCount = 0
			m.firstNewMsgID = 0

			return m, m.loadMessages(m.currentChat.ID)
		}

	case keybind.ActionOpenSwitcher:
		m.prevView = m.currentView
		m.currentView = ViewSwitcher
		m.switcherInput.Reset()
		m.switcherInput.Focus()
		m.switcherIdx = 0

	case keybind.ActionShowHelp:
		m.prevView = m.currentView
		m.currentView = ViewHelp
		return m, nil
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
		// Move cursor down by one line
		m.cursorLine++
		m.adjustViewport()
		// Reset new message counter if at bottom
		totalLines := m.getTotalLines()
		if m.cursorLine >= totalLines-1 {
			m.newMsgCount = 0
			m.firstNewMsgID = 0
		}

	case keybind.ActionMoveUp:
		// Move cursor up by one line
		m.cursorLine--
		m.adjustViewport()
		// Load more messages when reaching the top
		if m.cursorLine == 0 && !m.loadingMore {
			m.loadingMore = true
			return m, m.loadMoreMessages()
		}

	case keybind.ActionJumpTop:
		m.cursorLine = 0
		m.adjustViewport()
		// Load more messages when jumping to top
		if !m.loadingMore {
			m.loadingMore = true
			return m, m.loadMoreMessages()
		}

	case keybind.ActionJumpBottom:
		// Jump to last line
		totalLines := m.getTotalLines()
		if totalLines > 0 {
			m.cursorLine = totalLines - 1
			m.newMsgCount = 0 // Reset new message counter
			m.firstNewMsgID = 0
		}
		m.adjustViewport()

	case keybind.ActionHalfPageDown:
		pageSize := m.getViewportHeight() / 2
		if pageSize < 1 {
			pageSize = 1
		}
		m.cursorLine += pageSize
		m.adjustViewport()
		// Reset new message counter if at bottom
		totalLines := m.getTotalLines()
		if m.cursorLine >= totalLines-1 {
			m.newMsgCount = 0
			m.firstNewMsgID = 0
		}

	case keybind.ActionHalfPageUp:
		pageSize := m.getViewportHeight() / 2
		if pageSize < 1 {
			pageSize = 1
		}
		m.cursorLine -= pageSize
		m.adjustViewport()
		// Load more messages when reaching the top
		if m.cursorLine == 0 && !m.loadingMore {
			m.loadingMore = true
			return m, m.loadMoreMessages()
		}

	case keybind.ActionPageDown:
		pageSize := m.getViewportHeight()
		if pageSize < 1 {
			pageSize = 1
		}
		m.cursorLine += pageSize
		m.adjustViewport()
		// Reset new message counter if at bottom
		totalLines := m.getTotalLines()
		if m.cursorLine >= totalLines-1 {
			m.newMsgCount = 0
			m.firstNewMsgID = 0
		}

	case keybind.ActionPageUp:
		pageSize := m.getViewportHeight()
		if pageSize < 1 {
			pageSize = 1
		}
		m.cursorLine -= pageSize
		m.adjustViewport()
		// Load more messages when reaching the top
		if m.cursorLine == 0 && !m.loadingMore {
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
			m.firstNewMsgID = 0
		}
		// Don't call adjustViewport - we're moving within viewport

	case keybind.ActionEnterInsert:
		m.statusBar.SetMode(keybind.ModeInsert)
		return m, m.input.Focus()

	case keybind.ActionAppend:
		// Scroll to bottom and enter insert mode (vim 'A' behavior)
		totalLines := m.getTotalLines()
		m.cursorLine = totalLines - 1
		if m.cursorLine < 0 {
			m.cursorLine = 0
		}
		m.newMsgCount = 0
		m.firstNewMsgID = 0
		m.adjustViewport()
		m.statusBar.SetMode(keybind.ModeInsert)
		return m, m.input.Focus()

	case keybind.ActionReply:
		// Reply to message at cursor position
		msgIdx := m.getMessageAtCursor()
		if msgIdx >= 0 && msgIdx < len(m.messages) {
			m.replyingTo = m.messages[msgIdx]
			// Scroll viewport down by 1 to account for reply preview line
			m.viewportStart++
			totalLines := m.getTotalLines()
			viewportHeight := m.getViewportHeight()
			maxViewportStart := totalLines - viewportHeight
			if maxViewportStart < 0 {
				maxViewportStart = 0
			}
			if m.viewportStart > maxViewportStart {
				m.viewportStart = maxViewportStart
			}
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

	case keybind.ActionEditMessage:
		// Edit message at cursor (only own messages)
		if m.currentChat == nil {
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification("No chat selected", 2*time.Second)
			return m, cmd
		}
		msgIdx := m.getMessageAtCursor()
		if msgIdx < 0 || msgIdx >= len(m.messages) {
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification("No message at cursor", 2*time.Second)
			return m, cmd
		}
		msg := m.messages[msgIdx]
		if !msg.IsOwn {
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification("Can only edit own messages", 2*time.Second)
			return m, cmd
		}
		if msg.Text == "" {
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification("Cannot edit: no text", 2*time.Second)
			return m, cmd
		}
		if msg.ID == 0 {
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification("Cannot edit: invalid message ID", 2*time.Second)
			return m, cmd
		}
		// Enter edit mode
		m.editingMessage = msg
		m.input.SetValue(msg.Text)
		m.vim.SetMode(keybind.ModeInsert)
		m.statusBar.SetMode(keybind.ModeInsert)
		return m, m.input.Focus()

	case keybind.ActionDeleteMessage:
		// Delete message at cursor (only own messages)
		if m.currentChat == nil {
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification("No chat selected", 2*time.Second)
			return m, cmd
		}
		msgIdx := m.getMessageAtCursor()
		if msgIdx < 0 || msgIdx >= len(m.messages) {
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification("No message at cursor", 2*time.Second)
			return m, cmd
		}
		msg := m.messages[msgIdx]
		if !msg.IsOwn {
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification("Can only delete own messages", 2*time.Second)
			return m, cmd
		}
		// Delete immediately (Telegram allows deletion of own messages)
		return m, m.deleteMessage(m.currentChat.ID, msg.ID)

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

	case keybind.ActionShowChatID:
		m.showChatID = !m.showChatID
		m.updateStatusBarForChat()
		return m, nil

	case keybind.ActionMarkRead:
		// Mark messages as read up to cursor position
		msgIdx := m.getMessageAtCursor()
		if msgIdx >= 0 && msgIdx < len(m.messages) && m.currentChat != nil {
			return m, m.markMessagesAsRead(m.currentChat.ID, m.messages[msgIdx].ID)
		}

	case keybind.ActionMarkUnread:
		// Mark dialog as unread
		if m.currentChat != nil {
			return m, m.markDialogUnread(m.currentChat.ID)
		}

	case keybind.ActionJumpToOriginal:
		// Jump to original message if current message is a reply
		msgIdx := m.getMessageAtCursor()
		if msgIdx >= 0 && msgIdx < len(m.messages) {
			currentMsg := m.messages[msgIdx]
			if currentMsg.ReplyToID > 0 {
				// Find the original message
				originalIdx := m.findMessageIndex(currentMsg.ReplyToID)
				if originalIdx >= 0 {
					// Push current position to jump stack
					m.jumpStack = append(m.jumpStack, m.cursorLine)
					// Jump to original message
					targetLine := m.getLineForMessageIndex(originalIdx)
					if targetLine >= 0 {
						m.cursorLine = targetLine
						m.adjustViewport()
					}
				} else {
					// Original message not loaded
					var cmd tea.Cmd
					m.statusBar, cmd = m.statusBar.ShowNotification("Original message not loaded", 2*time.Second)
					return m, cmd
				}
			}
		}

	case keybind.ActionJumpBack:
		// Jump back to previous position
		if len(m.jumpStack) > 0 {
			// Pop from stack
			prevLine := m.jumpStack[len(m.jumpStack)-1]
			m.jumpStack = m.jumpStack[:len(m.jumpStack)-1]
			m.cursorLine = prevLine
			m.adjustViewport()
		}

	case keybind.ActionExitToNormal:
		m.input.Blur()
		m.input.Reset() // Clear input when exiting edit/reply mode
		m.replyingTo = nil
		m.editingMessage = nil
		m.statusBar.SetMode(keybind.ModeNormal)

	case keybind.ActionShowHelp:
		m.prevView = m.currentView
		m.currentView = ViewHelp
		return m, nil

	case keybind.ActionToggleMute:
		if m.currentChat != nil {
			chatID := m.currentChat.ID
			var msg string
			if m.mutedChats[chatID] {
				delete(m.mutedChats, chatID)
				msg = "Chat unmuted"
			} else {
				m.mutedChats[chatID] = true
				msg = "Chat muted"
			}
			m.statusBar.SetNotifyState(m.getNotifyState(chatID))
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification(msg, 2*time.Second)
			return m, cmd
		}

	case keybind.ActionToggleGlobalMute:
		m.globalMute = !m.globalMute
		var msg string
		if m.globalMute {
			msg = "All notifications muted"
		} else {
			msg = "Notifications enabled"
		}
		var cmd tea.Cmd
		m.statusBar, cmd = m.statusBar.ShowNotification(msg, 2*time.Second)
		return m, cmd

	case keybind.ActionToggleWatch:
		if m.currentChat != nil {
			chatID := m.currentChat.ID
			var msg string
			if m.watchedChats[chatID] {
				delete(m.watchedChats, chatID)
				msg = "Stopped watching"
			} else {
				m.watchedChats[chatID] = true
				msg = "Watching chat"
			}
			m.statusBar.SetNotifyState(m.getNotifyState(chatID))
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification(msg, 2*time.Second)
			return m, cmd
		}

	case keybind.ActionReact1, keybind.ActionReact2, keybind.ActionReact3,
		keybind.ActionReact4, keybind.ActionReact5, keybind.ActionReact6:
		// Send reaction to message at cursor (toggle behavior)
		if m.currentChat == nil {
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification("No chat selected", 2*time.Second)
			return m, cmd
		}
		msgIdx := m.getMessageAtCursor()
		if msgIdx < 0 || msgIdx >= len(m.messages) {
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification("No message at cursor", 2*time.Second)
			return m, cmd
		}
		msg := m.messages[msgIdx]
		// Map action to emoji index
		emojiIdx := int(result.Action - keybind.ActionReact1)
		if emojiIdx >= len(m.config.Reaction.Emojis) {
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification("No emoji configured for this key", 2*time.Second)
			return m, cmd
		}
		emoji := m.config.Reaction.Emojis[emojiIdx]
		// Always send the emoji - API layer handles toggle via MESSAGE_NOT_MODIFIED
		return m, m.sendReaction(m.currentChat.ID, msg.ID, emoji)

	case keybind.ActionRemoveReact:
		// Remove reaction from message at cursor
		if m.currentChat == nil {
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification("No chat selected", 2*time.Second)
			return m, cmd
		}
		msgIdx := m.getMessageAtCursor()
		if msgIdx < 0 || msgIdx >= len(m.messages) {
			var cmd tea.Cmd
			m.statusBar, cmd = m.statusBar.ShowNotification("No message at cursor", 2*time.Second)
			return m, cmd
		}
		msg := m.messages[msgIdx]
		return m, m.sendReaction(m.currentChat.ID, msg.ID, "") // Empty string removes reaction
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

	// Auto-expand textarea height based on content (min 3, max 6 lines)
	m.adjustTextareaHeight()

	return m, cmd
}

// adjustTextareaHeight adjusts textarea height based on content
func (m *Model) adjustTextareaHeight() {
	const minHeight = 3

	// Max height is half the terminal height
	maxHeight := m.height / 2
	if maxHeight < minHeight {
		maxHeight = minHeight
	}

	// Count lines in content
	lineCount := m.input.LineCount()

	// Clamp to min/max
	newHeight := lineCount
	if newHeight < minHeight {
		newHeight = minHeight
	}
	if newHeight > maxHeight {
		newHeight = maxHeight
	}

	// Get current height to calculate change
	oldHeight := m.input.Height()
	heightDiff := newHeight - oldHeight

	// Set the new height first so viewport calculations use correct size
	m.input.SetHeight(newHeight)

	// If textarea is growing, scroll viewport down to keep same message in view
	if heightDiff > 0 {
		m.viewportStart += heightDiff
		// Clamp viewportStart to valid range (using NEW viewport height)
		totalLines := m.getTotalLines()
		viewportHeight := m.getViewportHeight()
		maxViewportStart := totalLines - viewportHeight
		if maxViewportStart < 0 {
			maxViewportStart = 0
		}
		if m.viewportStart > maxViewportStart {
			m.viewportStart = maxViewportStart
		}
	}
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

	chatID := m.currentChat.ID

	// Handle edit mode
	if m.editingMessage != nil {
		messageID := m.editingMessage.ID
		m.input.Reset()
		m.editingMessage = nil
		return m, m.editMessage(chatID, messageID, text)
	}

	var replyToID int64
	if m.replyingTo != nil {
		replyToID = m.replyingTo.ID
	}

	// Clear input and reply state
	m.input.Reset()
	m.replyingTo = nil

	// Stay in insert mode to allow sending multiple messages

	// Send message via client
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

	case "ctrl+u":
		// Mark selected chat as unread from switcher
		filtered := m.getFilteredChats()
		if len(filtered) > 0 && m.switcherIdx < len(filtered) {
			return m, m.markDialogUnread(filtered[m.switcherIdx].ID)
		}
		return m, nil

	case "ctrl+r":
		// Mark selected chat as read from switcher
		filtered := m.getFilteredChats()
		if len(filtered) > 0 && m.switcherIdx < len(filtered) {
			chat := filtered[m.switcherIdx]
			// Find the latest message ID for this chat to mark all as read
			// We use 0 as a signal to mark all messages (will be handled in client)
			return m, m.markChatFullyRead(chat.ID)
		}
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
			m.newMsgCount = 0
			m.firstNewMsgID = 0

			// Find the index in the main chat list
			for i, c := range m.chats {
				if c.ID == m.currentChat.ID {
					m.chatIdx = i
					break
				}
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
	case ViewSetup:
		return m.viewSetup()
	case ViewAuth:
		return m.authView.View()
	case ViewChatList:
		return m.viewChatList()
	case ViewChat:
		return m.viewChat()
	case ViewSwitcher:
		return m.viewSwitcher()
	case ViewHelp:
		return m.viewHelp()
	}

	return "Unknown view"
}

func (m Model) viewSetup() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	highlightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	codeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("tlgram - Terminal Telegram Client"),
		"",
		highlightStyle.Render("Setup Required"),
		"",
		"No Telegram API credentials found.",
		"",
		infoStyle.Render("To use tlgram, you need API credentials from Telegram:"),
		"",
		"1. Go to "+highlightStyle.Render("https://my.telegram.org/apps"),
		"2. Log in with your phone number",
		"3. Create an application (any name works)",
		"4. Note your "+codeStyle.Render("api_id")+" and "+codeStyle.Render("api_hash"),
		"",
		infoStyle.Render("Then add them to your environment. Recommended approach:"),
		"",
		"Create "+codeStyle.Render("~/.secrets")+" file (don't commit to git):",
		codeStyle.Render("  export TLGRAM_API_ID=12345678"),
		codeStyle.Render("  export TLGRAM_API_HASH=\"your_api_hash\""),
		"",
		"Source it from your shell rc file:",
		codeStyle.Render("  # In ~/.bashrc or ~/.zshrc"),
		codeStyle.Render("  [ -f ~/.secrets ] && source ~/.secrets"),
		"",
		infoStyle.Render("Alternative: Add directly to ~/.config/tlgram/config.toml"),
		"",
		highlightStyle.Render("Press 'q' to quit"),
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
		} else if chat.MarkedUnread {
			unread = " •"
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
	// Height calculation: status bar (1) + input textarea (dynamic) + input border (2) + scroll indicator (1)
	inputHeight := m.input.Height() + 2         // textarea height + border (top + bottom)
	msgHeight := m.height - 1 - inputHeight - 1 // status bar, input area, scroll indicator
	if m.replyingTo != nil {
		msgHeight--
	}
	if msgHeight < 3 {
		msgHeight = 3
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
			} else if chat.MarkedUnread {
				unread = " •"
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
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Ctrl-n/p: navigate • Ctrl-r/u: read/unread • Enter: select • Esc: close"))

	box := boxStyle.Render(content.String())

	// Center the box in the screen
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) viewHelp() string {
	// Box width is about 70% of screen width, min 50, max 90
	boxWidth := m.width * 70 / 100
	if boxWidth < 50 {
		boxWidth = 50
	}
	if boxWidth > 90 {
		boxWidth = 90
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(boxWidth)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39"))

	categoryStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214"))

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("35"))

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	var content strings.Builder
	content.WriteString(titleStyle.Render("Keybindings"))
	content.WriteString("\n\n")

	// Navigation
	content.WriteString(categoryStyle.Render("NAVIGATION"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  j/k") + descStyle.Render("          Move down/up"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  gg/G") + descStyle.Render("         Jump to top/bottom"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  H/L") + descStyle.Render("          Viewport top/bottom"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  Ctrl+d/u") + descStyle.Render("     Half page down/up"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  Ctrl+f/b") + descStyle.Render("     Full page down/up"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  o") + descStyle.Render("            Jump to original message"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  Ctrl+o") + descStyle.Render("       Jump back"))
	content.WriteString("\n\n")

	// Mode changes
	content.WriteString(categoryStyle.Render("MODE"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  i") + descStyle.Render("            Enter insert mode"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  A") + descStyle.Render("            Scroll to bottom + insert"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  Esc") + descStyle.Render("          Exit to normal mode"))
	content.WriteString("\n\n")

	// Actions
	content.WriteString(categoryStyle.Render("ACTIONS"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  r") + descStyle.Render("            Reply to message"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  yy") + descStyle.Render("           Copy message"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  cc") + descStyle.Render("           Edit own message"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  D") + descStyle.Render("            Delete own message"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  d") + descStyle.Render("            Download media"))
	content.WriteString("\n\n")

	// Reading
	content.WriteString(categoryStyle.Render("READING"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  R") + descStyle.Render("            Mark messages as read"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  U") + descStyle.Render("            Mark dialog as unread"))
	content.WriteString("\n\n")

	// Display
	content.WriteString(categoryStyle.Render("DISPLAY"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  u") + descStyle.Render("            Toggle username display"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  I") + descStyle.Render("            Toggle chat ID in header"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  w") + descStyle.Render("            Watch chat (always notify)"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  m") + descStyle.Render("            Mute this chat"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  M") + descStyle.Render("            Mute ALL notifications"))
	content.WriteString("\n\n")

	// Reactions
	content.WriteString(categoryStyle.Render("REACTIONS"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  1-6") + descStyle.Render("          Add reaction (👍 👎 ❤️ 🔥 🎉 😁)"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  0") + descStyle.Render("            Remove reaction"))
	content.WriteString("\n\n")

	// Switcher
	content.WriteString(categoryStyle.Render("SWITCHER"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  Ctrl+p") + descStyle.Render("       Open chat switcher"))
	content.WriteString("\n\n")

	// General
	content.WriteString(categoryStyle.Render("GENERAL"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  q") + descStyle.Render("            Quit"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  ?") + descStyle.Render("            Show this help"))
	content.WriteString("\n\n")

	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Press ? or Esc to close"))

	box := boxStyle.Render(content.String())

	// Center the box in the screen
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "?", "esc", "escape", "q":
		m.currentView = m.prevView
		return m, nil
	}

	return m, nil
}

// messageLine represents a line and which message it belongs to
type messageLine struct {
	msgIdx    int
	lineType  int                 // 0=reply, 1=firstLine, 2=continuation, 3=reactions
	sender    string              // raw sender name (for first line)
	content   string              // message content or reply text
	timestamp string              // formatted timestamp (for first line)
	isOwn     bool                // whether this is own message
	isUnread  bool                // whether this message is unread (for visual indicator)
	reactions []telegram.Reaction // reactions (for lineType 3)
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
	case t.Year() == now.Year():
		// Same year - show date without year
		return t.Format("2 Jan 15:04")
	default:
		// Different year - show full date with year
		return t.Format("2 Jan 2006 15:04")
	}
}

func (m Model) renderMessages(maxHeight int) string {
	if len(m.messages) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Height(maxHeight)
		return emptyStyle.Render("No messages yet. Press 'i' to start typing.")
	}

	// Calculate max sender name length for alignment
	maxSenderLen := 0
	for _, msg := range m.messages {
		senderName := m.getSenderName(msg)
		if len(senderName) > maxSenderLen {
			maxSenderLen = len(senderName)
		}
	}

	// Calculate available width for message text (leave room for sender prefix)
	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	// Build all lines with raw data (styling applied at render time)
	var allLines []messageLine

	for msgIdx, msg := range m.messages {
		// Determine if message is unread (not own message and ID > last read)
		isUnread := false
		if m.currentChat != nil && !msg.IsOwn && msg.ID > m.currentChat.LastReadInboxID {
			isUnread = true
		}

		// Reply context (if any)
		if msg.ReplyToID > 0 {
			replyContent := "↳ Reply"
			// Look up the original message in loaded messages
			if originalMsg := m.findMessageByID(msg.ReplyToID); originalMsg != nil {
				replySender := originalMsg.Sender
				if m.showUsernames && originalMsg.SenderUsername != "" {
					replySender = originalMsg.SenderUsername
				}
				replyText := originalMsg.Text
				previewLen := m.config.Appearance.ReplyPreviewLength
				if previewLen <= 0 {
					previewLen = 30 // fallback default
				}
				if len(replyText) > previewLen {
					replyText = replyText[:previewLen] + "..."
				}
				replyContent = fmt.Sprintf("↳ %s: %s", replySender, replyText)
			}
			allLines = append(allLines, messageLine{
				msgIdx:   msgIdx,
				lineType: 0, // reply
				content:  replyContent,
				isUnread: isUnread,
			})
		}

		// Determine which name to show
		senderName := m.getSenderName(msg)

		// Format timestamp with relative date
		timestamp := formatRelativeTime(msg.Time)
		timestampWidth := displayWidth(timestamp)

		// Calculate widths: first line needs room for timestamp, continuation lines don't
		firstLineWidth := contentWidth - maxSenderLen - 4 - timestampWidth - 1
		if firstLineWidth < 20 {
			firstLineWidth = 20 // Minimum width for first line
		}
		contLineWidth := contentWidth - maxSenderLen - 4

		// Wrap long messages with different widths for first vs continuation lines
		wrappedLines := wrapTextFirstLine(msg.Text, firstLineWidth, contLineWidth)

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
					isUnread:  isUnread,
				})
			} else {
				// Continuation line
				allLines = append(allLines, messageLine{
					msgIdx:   msgIdx,
					lineType: 2, // continuation
					sender:   senderName,
					content:  wline,
					isUnread: isUnread,
				})
			}
		}

		// Reactions line (if any)
		if len(msg.Reactions) > 0 {
			allLines = append(allLines, messageLine{
				msgIdx:    msgIdx,
				lineType:  3, // reactions
				sender:    senderName,
				reactions: msg.Reactions,
				isUnread:  isUnread,
			})
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
		renderedLine := m.renderLine(line, contentWidth, maxSenderLen, isCursor)
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
func (m Model) renderLine(line messageLine, contentWidth int, maxSenderLen int, isCursor bool) string {
	// Base styles - cursor adds subtle background highlight
	cursorBg := lipgloss.Color("235")

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

	// Unread marker style (red bold bar for visibility)
	unreadMarkerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	unreadPrefix := "  " // Default: no marker (2 spaces for alignment)
	if line.isUnread {
		unreadPrefix = unreadMarkerStyle.Render("| ")
	}

	var result string

	switch line.lineType {
	case 0: // Reply line
		// Indent reply to align with message content
		replyIndent := strings.Repeat(" ", maxSenderLen)
		result = unreadPrefix + textStyle.Render(replyIndent) + replyStyle.Render(line.content)

	case 1: // First line with sender and timestamp
		// Right-align sender name by padding on the left
		senderPadding := strings.Repeat(" ", maxSenderLen-len(line.sender))
		var styledSender string
		if line.isOwn {
			styledSender = ownSenderStyle.Render(line.sender)
		} else {
			styledSender = senderStyle.Render(line.sender)
		}

		// Calculate padding for right-aligned timestamp (use displayWidth for emoji support)
		contentDisplayWidth := displayWidth(line.content)
		timestampDisplayWidth := displayWidth(line.timestamp)
		baseLen := maxSenderLen + 2 + contentDisplayWidth + 2 // +2 for unread marker
		padding := contentWidth - baseLen - timestampDisplayWidth
		if padding < 1 {
			padding = 1
		}

		result = unreadPrefix + textStyle.Render(senderPadding) + styledSender + textStyle.Render(": "+line.content+strings.Repeat(" ", padding)) + timeStyle.Render(line.timestamp)

	case 2: // Continuation line
		indent := strings.Repeat(" ", maxSenderLen)
		result = unreadPrefix + textStyle.Render(indent+line.content)

	case 3: // Reactions line
		indent := strings.Repeat(" ", maxSenderLen)
		var reactionParts []string
		reactionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		reactionChosenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
		if isCursor {
			reactionStyle = reactionStyle.Background(cursorBg)
			reactionChosenStyle = reactionChosenStyle.Background(cursorBg)
		}
		for _, r := range line.reactions {
			reactionText := fmt.Sprintf("%s %d", r.Emoji, r.Count)
			if r.Chosen {
				reactionParts = append(reactionParts, reactionChosenStyle.Render("["+reactionText+"]"))
			} else {
				reactionParts = append(reactionParts, reactionStyle.Render(reactionText))
			}
		}
		result = unreadPrefix + textStyle.Render(indent+"  ") + strings.Join(reactionParts, " ")
	}

	// Pad to full width for cursor line
	if isCursor {
		lineWidth := displayWidth(result)
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

// findMessageByID finds a message by its ID in the loaded messages
func (m Model) findMessageByID(id int64) *Message {
	for _, msg := range m.messages {
		if msg.ID == id {
			return msg
		}
	}
	return nil
}

// findMessageIndex finds the index of a message by its ID
func (m Model) findMessageIndex(id int64) int {
	for i, msg := range m.messages {
		if msg.ID == id {
			return i
		}
	}
	return -1
}

// getLineForMessageIndex returns the first cursor line for a message at the given index
func (m Model) getLineForMessageIndex(targetIdx int) int {
	if len(m.messages) == 0 || targetIdx < 0 || targetIdx >= len(m.messages) {
		return -1
	}

	// Calculate max sender name length (same logic as renderMessages)
	maxSenderLen := 0
	for _, msg := range m.messages {
		senderName := m.getSenderName(msg)
		if len(senderName) > maxSenderLen {
			maxSenderLen = len(senderName)
		}
	}

	// Calculate content width
	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	line := 0
	for msgIdx, msg := range m.messages {
		if msgIdx == targetIdx {
			return line
		}
		// Account for reply line
		if msg.ReplyToID > 0 {
			line++
		}
		// Account for message lines
		timestamp := formatRelativeTime(msg.Time)
		timestampWidth := displayWidth(timestamp)
		firstLineWidth := contentWidth - maxSenderLen - 4 - timestampWidth - 1
		if firstLineWidth < 20 {
			firstLineWidth = 20
		}
		contLineWidth := contentWidth - maxSenderLen - 4
		wrappedLines := wrapTextFirstLine(msg.Text, firstLineWidth, contLineWidth)
		line += len(wrappedLines)
	}
	return -1
}

// getMessageAtCursor returns the message index that contains the cursor line
func (m Model) getMessageAtCursor() int {
	if len(m.messages) == 0 {
		return -1
	}

	// Calculate max sender name length (same logic as renderMessages)
	maxSenderLen := 0
	for _, msg := range m.messages {
		senderName := m.getSenderName(msg)
		if len(senderName) > maxSenderLen {
			maxSenderLen = len(senderName)
		}
	}

	// Build line mapping (same logic as renderMessages)
	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	var lineToMsg []int // maps line index to message index
	for msgIdx, msg := range m.messages {
		// Account for reply line
		if msg.ReplyToID > 0 {
			lineToMsg = append(lineToMsg, msgIdx)
		}
		// Account for message lines (use same width calculation as renderMessages)
		timestamp := formatRelativeTime(msg.Time)
		timestampWidth := displayWidth(timestamp)
		firstLineWidth := contentWidth - maxSenderLen - 4 - timestampWidth - 1
		if firstLineWidth < 20 {
			firstLineWidth = 20
		}
		contLineWidth := contentWidth - maxSenderLen - 4
		wrappedLines := wrapTextFirstLine(msg.Text, firstLineWidth, contLineWidth)
		for range wrappedLines {
			lineToMsg = append(lineToMsg, msgIdx)
		}
		// Account for reaction line
		if len(msg.Reactions) > 0 {
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

	// Calculate max sender name length (same logic as renderMessages)
	maxSenderLen := 0
	for _, msg := range m.messages {
		senderName := m.getSenderName(msg)
		if len(senderName) > maxSenderLen {
			maxSenderLen = len(senderName)
		}
	}

	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	totalLines := 0
	for _, msg := range m.messages {
		if msg.ReplyToID > 0 {
			totalLines++
		}
		// Use same width calculation as renderMessages
		timestamp := formatRelativeTime(msg.Time)
		timestampWidth := displayWidth(timestamp)
		firstLineWidth := contentWidth - maxSenderLen - 4 - timestampWidth - 1
		if firstLineWidth < 20 {
			firstLineWidth = 20
		}
		contLineWidth := contentWidth - maxSenderLen - 4
		wrappedLines := wrapTextFirstLine(msg.Text, firstLineWidth, contLineWidth)
		totalLines += len(wrappedLines)
		// Account for reaction line
		if len(msg.Reactions) > 0 {
			totalLines++
		}
	}
	return totalLines
}

// getViewportHeight returns the number of visible lines in the message viewport
func (m Model) getViewportHeight() int {
	// Height minus status bar (1), input area (textarea + border), scroll indicator (1)
	inputHeight := m.input.Height() + 2      // textarea height + border (top + bottom)
	vh := m.height - 1 - inputHeight - 1 - 1 // status bar, input area, scroll indicator, buffer
	if m.replyingTo != nil {
		vh--
	}
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

// wrapTextFirstLine wraps text with different widths for first line vs continuation lines
// First line is narrower to leave room for timestamp
func wrapTextFirstLine(text string, firstLineWidth, contLineWidth int) []string {
	if firstLineWidth <= 0 {
		firstLineWidth = 40
	}
	if contLineWidth <= 0 {
		contLineWidth = 80
	}

	// First split by explicit newlines
	paragraphs := strings.Split(text, "\n")
	var lines []string
	isFirstLine := true

	for _, para := range paragraphs {
		// Handle empty lines (preserve blank lines in messages)
		if para == "" {
			lines = append(lines, "")
			isFirstLine = false
			continue
		}

		// Wrap this paragraph using display width
		for displayWidth(para) > 0 {
			width := contLineWidth
			if isFirstLine {
				width = firstLineWidth
			}

			paraWidth := displayWidth(para)
			if paraWidth <= width {
				lines = append(lines, para)
				isFirstLine = false
				break
			}

			// Find a good break point by iterating through runes
			runes := []rune(para)
			breakAt := 0
			currentWidth := 0
			lastSpace := -1
			i := 0

			for i < len(runes) {
				r := runes[i]
				var runeWidth int

				// Check for emoji + skin tone modifier sequence
				if isEmojiBase(r) && i+1 < len(runes) && isSkinToneModifier(runes[i+1]) {
					runeWidth = 4
					if currentWidth+runeWidth > width {
						break
					}
					currentWidth += runeWidth
					breakAt = i + 2
					i += 2
					continue
				}

				runeWidth = lipgloss.Width(string(r))
				if currentWidth+runeWidth > width {
					break
				}
				currentWidth += runeWidth
				breakAt = i + 1
				if r == ' ' {
					lastSpace = i
				}
				i++
			}

			// Prefer breaking at a space if we found one in the latter half
			if lastSpace > breakAt/2 {
				breakAt = lastSpace
			}

			if breakAt == 0 {
				breakAt = 1 // At minimum, take one rune
			}

			lines = append(lines, string(runes[:breakAt]))
			para = strings.TrimLeft(string(runes[breakAt:]), " ")
			isFirstLine = false
		}
	}

	// Ensure at least one line
	if len(lines) == 0 {
		lines = append(lines, "")
	}

	return lines
}

// wrapText wraps text to the specified width, handling explicit newlines
// Uses display width (not byte length) for proper emoji support
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

		// Wrap this paragraph using display width
		for displayWidth(para) > 0 {
			paraWidth := displayWidth(para)
			if paraWidth <= width {
				lines = append(lines, para)
				break
			}

			// Find a good break point by iterating through runes
			runes := []rune(para)
			breakAt := 0
			currentWidth := 0
			lastSpace := -1
			i := 0

			for i < len(runes) {
				r := runes[i]
				var runeWidth int

				// Check for emoji + skin tone modifier sequence
				if isEmojiBase(r) && i+1 < len(runes) && isSkinToneModifier(runes[i+1]) {
					runeWidth = 4
					if currentWidth+runeWidth > width {
						break
					}
					currentWidth += runeWidth
					breakAt = i + 2
					i += 2
					continue
				}

				runeWidth = lipgloss.Width(string(r))
				if currentWidth+runeWidth > width {
					break
				}
				currentWidth += runeWidth
				breakAt = i + 1
				if r == ' ' {
					lastSpace = i
				}
				i++
			}

			// Prefer breaking at a space if we found one in the latter half
			if lastSpace > breakAt/2 {
				breakAt = lastSpace
			}

			if breakAt == 0 {
				breakAt = 1 // At minimum, take one rune
			}

			lines = append(lines, string(runes[:breakAt]))
			para = strings.TrimLeft(string(runes[breakAt:]), " ")
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
		placeholder = "Press ? for help"
	}
	m.input.Placeholder = placeholder

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("241")).
		Width(m.width - 2)

	return inputStyle.Render(m.input.View())
}

// isSkinToneModifier returns true if the rune is an emoji skin tone modifier
// Skin tone modifiers are U+1F3FB through U+1F3FF
func isSkinToneModifier(r rune) bool {
	return r >= 0x1F3FB && r <= 0x1F3FF
}

// isEmojiBase returns true if the rune is a base emoji that can take a skin tone modifier
// This includes most human-related emoji (hands, people, body parts)
func isEmojiBase(r rune) bool {
	// Common emoji that accept skin tone modifiers
	// Hands and gestures: 👋👌👍👎👏🤝✌🤞🤟🤘🤙👆👇👈👉👊✊🤛🤜🙌👐🤲🙏💪🦵🦶🖐✋🖖👋🤚🖕☝️🫰🫱🫲🫳🫴🫵🫶
	// People emoji, body parts, etc.
	switch {
	case r >= 0x1F44A && r <= 0x1F44F: // 👊👋👌👍👎👏
		return true
	case r >= 0x1F466 && r <= 0x1F478: // Various people emoji
		return true
	case r >= 0x1F481 && r <= 0x1F487: // More people emoji
		return true
	case r >= 0x1F4AA && r <= 0x1F4AA: // 💪
		return true
	case r >= 0x1F574 && r <= 0x1F575: // 🕴🕵
		return true
	case r >= 0x1F590 && r <= 0x1F590: // 🖐
		return true
	case r >= 0x1F595 && r <= 0x1F596: // 🖕🖖
		return true
	case r >= 0x1F645 && r <= 0x1F64F: // Various gestures 🙅🙆🙇🙋🙌🙍🙎🙏
		return true
	case r >= 0x1F6A3 && r <= 0x1F6A3: // 🚣
		return true
	case r >= 0x1F6B4 && r <= 0x1F6B6: // 🚴🚵🚶
		return true
	case r >= 0x1F6C0 && r <= 0x1F6C0: // 🛀
		return true
	case r >= 0x1F6CC && r <= 0x1F6CC: // 🛌
		return true
	case r >= 0x1F90C && r <= 0x1F90F: // 🤌🤍🤎🤏
		return true
	case r >= 0x1F918 && r <= 0x1F91F: // Various hand gestures 🤘🤙🤚🤛🤜🤝🤞🤟
		return true
	case r >= 0x1F926 && r <= 0x1F926: // 🤦
		return true
	case r >= 0x1F930 && r <= 0x1F939: // Various people 🤰🤱🤲🤳🤴🤵🤶🤷🤸🤹
		return true
	case r >= 0x1F93D && r <= 0x1F93E: // 🤽🤾
		return true
	case r >= 0x1F977 && r <= 0x1F977: // 🥷
		return true
	case r >= 0x1F9B5 && r <= 0x1F9B6: // 🦵🦶
		return true
	case r >= 0x1F9B8 && r <= 0x1F9B9: // 🦸🦹
		return true
	case r >= 0x1F9BB && r <= 0x1F9BB: // 🦻
		return true
	case r >= 0x1F9CD && r <= 0x1F9CF: // 🧍🧎🧏
		return true
	case r >= 0x1F9D1 && r <= 0x1F9DD: // 🧑🧒🧓🧔🧕🧖🧗🧘🧙🧚🧛🧜🧝
		return true
	case r >= 0x1FAC3 && r <= 0x1FAC5: // 🫃🫄🫅
		return true
	case r >= 0x1FAF0 && r <= 0x1FAF6: // 🫰🫱🫲🫳🫴🫵🫶
		return true
	case r == 0x261D: // ☝
		return true
	case r == 0x270A: // ✊
		return true
	case r == 0x270B: // ✋
		return true
	case r == 0x270C: // ✌
		return true
	case r == 0x270D: // ✍
		return true
	}
	return false
}

// displayWidth calculates the display width of a string, accounting for emoji
// skin tone modifier sequences which render differently than lipgloss.Width calculates
func displayWidth(s string) int {
	runes := []rune(s)
	width := 0
	i := 0
	for i < len(runes) {
		r := runes[i]

		// Check if this is a base emoji followed by a skin tone modifier
		if isEmojiBase(r) && i+1 < len(runes) && isSkinToneModifier(runes[i+1]) {
			// Emoji + skin tone modifier renders as width 4 in terminal
			// (some terminals render as 2, but most modern terminals do 4)
			width += 4
			i += 2 // Skip both the base emoji and the modifier
			continue
		}

		// For other characters, use lipgloss.Width for individual rune
		width += lipgloss.Width(string(r))
		i++
	}
	return width
}
