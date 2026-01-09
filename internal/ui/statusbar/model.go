package statusbar

import (
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

	modeSearchStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("136")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1).
			Bold(true)

	modeCommandStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("241")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1).
			Bold(true)

	chatNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252"))

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
)

// Model represents the status bar
type Model struct {
	width  int
	height int

	mode       keybind.Mode
	chatName   string
	connState  telegram.ConnectionState
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

// SetConnectionState sets the connection state
func (m *Model) SetConnectionState(state telegram.ConnectionState) {
	m.connState = state
}

// ShowNotification displays a temporary notification
func (m *Model) ShowNotification(text string, duration time.Duration) tea.Cmd {
	m.notification = text
	m.notifyUntil = time.Now().Add(duration)
	return tea.Tick(duration, func(t time.Time) tea.Msg {
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
	case keybind.ModeSearch:
		modeView = modeSearchStyle.Render("SEARCH")
	case keybind.ModeCommand:
		modeView = modeCommandStyle.Render("COMMAND")
	default:
		modeView = modeNormalStyle.Render("NORMAL")
	}

	// Chat name
	chatView := chatNameStyle.Render(m.chatName)

	// Connection state or notification
	var rightView string
	if m.notification != "" && time.Now().Before(m.notifyUntil) {
		rightView = notificationStyle.Render(m.notification)
	} else {
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
