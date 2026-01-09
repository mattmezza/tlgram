package app

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattmezza/tlgram/internal/config"
	"github.com/mattmezza/tlgram/internal/keybind"
)

// View represents the current view
type View int

const (
	ViewAuth View = iota
	ViewChatList
	ViewChat
	ViewSwitcher
)

// ConnectionState represents the connection state
type ConnectionState int

const (
	ConnectionConnecting ConnectionState = iota
	ConnectionConnected
	ConnectionReconnecting
	ConnectionOffline
)

func (c ConnectionState) String() string {
	switch c {
	case ConnectionConnecting:
		return "Connecting..."
	case ConnectionConnected:
		return "Connected"
	case ConnectionReconnecting:
		return "Reconnecting..."
	case ConnectionOffline:
		return "Offline"
	default:
		return "Unknown"
	}
}

// Model is the main application model
type Model struct {
	// Dimensions
	width  int
	height int

	// State
	vim         *keybind.VimState
	currentView View
	prevView    View
	connState   ConnectionState
	quitting    bool

	// Target chat from CLI
	targetChat string

	// Configuration
	config *config.Config

	// Sub-components
	viewport viewport.Model
	input    textinput.Model

	// Chat state (temporary - will be replaced with proper types)
	messages    []Message
	selectedIdx int
	chatName    string

	// Status bar
	statusMsg     string
	statusTimeout int
}

// Message represents a chat message (temporary simple version)
type Message struct {
	ID       int64
	Sender   string
	Text     string
	IsOwn    bool
	Time     string
	ReplyTo  *Message
}

// New creates a new application model
func New(cfg *config.Config, targetChat string) Model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.CharLimit = 4096

	vp := viewport.New(80, 20)

	return Model{
		config:      cfg,
		targetChat:  targetChat,
		vim:         keybind.NewVimState(),
		currentView: ViewAuth, // Start with auth
		connState:   ConnectionConnecting,
		input:       ti,
		viewport:    vp,
		messages:    []Message{},
		chatName:    "tlgram",
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("tlgram"),
		// TODO: Initialize TDLib client
		// TODO: Start auth flow or load existing session
	)
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 4 // Leave room for input and status
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global quit with Ctrl+C
	if key == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}

	// Process through vim state machine
	result := m.vim.ProcessKey(key)

	switch result.Action {
	case keybind.ActionQuit:
		m.quitting = true
		return m, tea.Quit

	case keybind.ActionMoveUp:
		for i := 0; i < result.Count; i++ {
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		}
		return m, nil

	case keybind.ActionMoveDown:
		for i := 0; i < result.Count; i++ {
			if m.selectedIdx < len(m.messages)-1 {
				m.selectedIdx++
			}
		}
		return m, nil

	case keybind.ActionJumpTop:
		m.selectedIdx = 0
		return m, nil

	case keybind.ActionJumpBottom:
		if len(m.messages) > 0 {
			m.selectedIdx = len(m.messages) - 1
		}
		return m, nil

	case keybind.ActionHalfPageDown:
		pageSize := m.viewport.Height / 2
		for i := 0; i < pageSize*result.Count; i++ {
			if m.selectedIdx < len(m.messages)-1 {
				m.selectedIdx++
			}
		}
		return m, nil

	case keybind.ActionHalfPageUp:
		pageSize := m.viewport.Height / 2
		for i := 0; i < pageSize*result.Count; i++ {
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		}
		return m, nil

	case keybind.ActionEnterInsert:
		return m, m.input.Focus()

	case keybind.ActionExitToNormal:
		m.input.Blur()
		return m, nil

	case keybind.ActionOpenSwitcher:
		m.prevView = m.currentView
		m.currentView = ViewSwitcher
		return m, nil

	case keybind.ActionCancel:
		if m.currentView == ViewSwitcher {
			m.currentView = m.prevView
		}
		return m, nil
	}

	// Handle input in insert mode
	if m.vim.Mode() == keybind.ModeInsert {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
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
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("tlgram - Terminal Telegram Client"),
		"",
		"Telegram API credentials required.",
		"",
		infoStyle.Render("1. Go to https://my.telegram.org/apps"),
		infoStyle.Render("2. Create an application"),
		infoStyle.Render("3. Add api_id and api_hash to ~/.config/tlgram/config.toml"),
		"",
		"Press 'q' to quit.",
	)
}

func (m Model) viewChatList() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39"))

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("Chats"),
		"",
		"(Chat list will be loaded here)",
		"",
		m.viewStatusBar(),
	)
}

func (m Model) viewChat() string {
	var content string

	if len(m.messages) == 0 {
		content = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("No messages yet")
	} else {
		content = m.renderMessages()
	}

	// Compose the view
	return lipgloss.JoinVertical(lipgloss.Left,
		m.viewStatusBar(),
		content,
		m.viewInput(),
	)
}

func (m Model) viewSwitcher() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2)

	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		"Chat Switcher",
		"",
		"(Type to search...)",
		"",
		"Press Escape to close",
	))
}

func (m Model) viewStatusBar() string {
	modeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	chatStyle := lipgloss.NewStyle().
		Bold(true)

	connStyle := lipgloss.NewStyle()
	switch m.connState {
	case ConnectionConnected:
		connStyle = connStyle.Foreground(lipgloss.Color("42"))
	case ConnectionReconnecting:
		connStyle = connStyle.Foreground(lipgloss.Color("214"))
	case ConnectionOffline:
		connStyle = connStyle.Foreground(lipgloss.Color("196"))
	default:
		connStyle = connStyle.Foreground(lipgloss.Color("241"))
	}

	mode := modeStyle.Render(m.vim.Mode().String())
	chat := chatStyle.Render(m.chatName)
	conn := connStyle.Render(m.connState.String())

	// Use remaining width for spacing
	leftPart := mode + " " + chat
	rightPart := conn

	gap := m.width - lipgloss.Width(leftPart) - lipgloss.Width(rightPart)
	if gap < 1 {
		gap = 1
	}

	barStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Width(m.width)

	return barStyle.Render(leftPart + lipgloss.NewStyle().Width(gap).Render("") + rightPart)
}

func (m Model) viewInput() string {
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("241")).
		Width(m.width - 2)

	return inputStyle.Render(m.input.View())
}

func (m Model) renderMessages() string {
	var lines []string

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("237"))

	senderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39"))

	ownSenderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("42"))

	for i, msg := range m.messages {
		var sender string
		if msg.IsOwn {
			sender = ownSenderStyle.Render("You")
		} else {
			sender = senderStyle.Render(msg.Sender)
		}

		line := sender + ": " + msg.Text

		if i == m.selectedIdx {
			line = selectedStyle.Render(line)
		}

		lines = append(lines, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
