package statusbar

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mattmezza/tlgram/internal/keybind"
	"github.com/mattmezza/tlgram/internal/telegram"
)

// Styles
var (
	barStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236"))

	modeNormalStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1).
			Bold(true)

	modeInsertStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("28")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1).
			Bold(true)

	chatNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252"))

	chatInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	connectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	reconnectingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))

	offlineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	connectingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	notificationStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))

	unreadStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("196")).
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Padding(0, 1)

	mutedIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Italic(true)

	watchedIndicatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Italic(true)
)

// NotifyState represents the notification state of a chat
type NotifyState int

const (
	NotifyStateNone    NotifyState = iota // Default - notifications enabled
	NotifyStateMuted                      // Chat is muted
	NotifyStateWatched                    // Chat is watched (always notify)
)

// Model represents the status bar
type Model struct {
	width  int
	height int

	mode         keybind.Mode
	chatName     string
	chatUsername string
	chatType     telegram.ChatType
	chatID       int64
	memberCount  int
	unreadCount  int
	showChatID   bool
	notifyState  NotifyState
	connState    telegram.ConnectionState
	notification string
	notifyUntil  time.Time
}

// New creates a new status bar model
func New() Model {
	return Model{
		mode:      keybind.ModeNormal,
		chatName:  "tlgram",
		connState: telegram.ConnectionStateUnknown,
	}
}

// SetSize sets the dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetMode sets the current vim mode
func (m *Model) SetMode(mode keybind.Mode) {
	m.mode = mode
}

// SetChatName sets the current chat name
func (m *Model) SetChatName(name string) {
	m.chatName = name
}

// SetChatInfo sets the full chat info for display
func (m *Model) SetChatInfo(name, username string, chatType telegram.ChatType, memberCount, unreadCount int) {
	m.chatName = name
	m.chatUsername = username
	m.chatType = chatType
	m.memberCount = memberCount
	m.unreadCount = unreadCount
}

// SetUnreadCount updates just the unread count
func (m *Model) SetUnreadCount(count int) {
	m.unreadCount = count
}

// SetChatID sets the chat ID for display
func (m *Model) SetChatID(id int64) {
	m.chatID = id
}

// SetShowChatID sets whether to display the chat ID
func (m *Model) SetShowChatID(show bool) {
	m.showChatID = show
}

// SetNotifyState sets the notification state indicator
func (m *Model) SetNotifyState(state NotifyState) {
	m.notifyState = state
}

// SetConnectionState sets the connection state
func (m *Model) SetConnectionState(state telegram.ConnectionState) {
	m.connState = state
}

// ShowNotification displays a temporary notification and returns the updated model and command
func (m Model) ShowNotification(text string, duration time.Duration) (Model, tea.Cmd) {
	m.notification = text
	m.notifyUntil = time.Now().Add(duration)
	return m, tea.Tick(duration, func(t time.Time) tea.Msg {
		return ClearNotificationMsg{}
	})
}

// ClearNotificationMsg is sent to clear the notification
type ClearNotificationMsg struct{}

// ClearNotification clears the notification if expired
func (m *Model) ClearNotification() {
	if time.Now().After(m.notifyUntil) {
		m.notification = ""
	}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg.(type) {
	case ClearNotificationMsg:
		m.ClearNotification()
	}
	return m, nil
}

// View renders the status bar
func (m Model) View() string {
	// Mode indicator
	var modeView string
	switch m.mode {
	case keybind.ModeNormal:
		modeView = modeNormalStyle.Render("NORMAL")
	case keybind.ModeInsert:
		modeView = modeInsertStyle.Render("INSERT")
	default:
		modeView = modeNormalStyle.Render("NORMAL")
	}

	// Build chat info based on type
	var chatView string
	switch m.chatType {
	case telegram.ChatTypePrivate:
		// DM: Name Surname @username
		chatView = chatNameStyle.Render(m.chatName)
		if m.chatUsername != "" {
			chatView += " " + chatInfoStyle.Render("@"+m.chatUsername)
		}
	case telegram.ChatTypeGroup, telegram.ChatTypeSupergroup:
		// Group: Group name (x members)
		chatView = chatNameStyle.Render(m.chatName)
		if m.memberCount > 0 {
			chatView += " " + chatInfoStyle.Render(fmt.Sprintf("(%d members)", m.memberCount))
		}
	case telegram.ChatTypeChannel:
		// Channel: Channel name (x subscribers)
		chatView = chatNameStyle.Render(m.chatName)
		if m.memberCount > 0 {
			chatView += " " + chatInfoStyle.Render(fmt.Sprintf("(%d subscribers)", m.memberCount))
		}
	default:
		chatView = chatNameStyle.Render(m.chatName)
	}

	// Add chat ID if enabled
	if m.showChatID && m.chatID != 0 {
		chatView += " " + chatInfoStyle.Render(fmt.Sprintf("ID: %d", m.chatID))
	}

	// Add notify state indicator
	switch m.notifyState {
	case NotifyStateMuted:
		chatView += " " + mutedIndicatorStyle.Render("(muted)")
	case NotifyStateWatched:
		chatView += " " + watchedIndicatorStyle.Render("(watched)")
	}

	// Build right side: unread indicator, notification, or connection state
	var rightView string

	// Check for notification first
	if m.notification != "" && time.Now().Before(m.notifyUntil) {
		rightView = notificationStyle.Render(m.notification)
	} else {
		// Show connection state
		switch m.connState {
		case telegram.ConnectionStateReady:
			rightView = connectedStyle.Render("Connected")
		case telegram.ConnectionStateConnecting, telegram.ConnectionStateConnectingToProxy:
			rightView = connectingStyle.Render("Connecting...")
		case telegram.ConnectionStateUpdating:
			rightView = reconnectingStyle.Render("Updating...")
		case telegram.ConnectionStateWaitingForNetwork:
			rightView = offlineStyle.Render("Offline")
		default:
			rightView = connectingStyle.Render("...")
		}
	}

	// Add unread indicator if there are unread messages
	if m.unreadCount > 0 {
		unreadBadge := unreadStyle.Render(fmt.Sprintf("%d new", m.unreadCount))
		rightView = unreadBadge + " " + rightView
	}

	// Calculate spacing
	leftPart := modeView + " " + chatView
	leftWidth := lipgloss.Width(leftPart)
	rightWidth := lipgloss.Width(rightView)

	gap := m.width - leftWidth - rightWidth
	if gap < 1 {
		gap = 1
	}

	spacer := lipgloss.NewStyle().Width(gap).Render("")

	return barStyle.Width(m.width).Render(leftPart + spacer + rightView)
}
