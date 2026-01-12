package chatview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mattmezza/tlgram/internal/keybind"
	"github.com/mattmezza/tlgram/internal/telegram"
)

// Styles
var (
	senderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	ownSenderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("237"))

	timeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	replyContextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Italic(true)

	unreadSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true)

	inputBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("241"))

	replyIndicatorStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("252")).
				Padding(0, 1)
)

// Model represents the chat view
type Model struct {
	width  int
	height int

	chatID   int64
	chatName string

	messages    []*telegram.Message
	selectedIdx int
	unreadSepAt int // Index where unread separator should appear, -1 if none

	viewport viewport.Model
	input    textinput.Model

	vim        *keybind.VimState
	replyingTo *telegram.Message

	// Configuration
	sendKey string // "enter" or "ctrl-enter"

	// Callbacks
	onSendMessage func(chatID int64, text string, replyTo int64) tea.Cmd
}

// New creates a new chat view model
func New(vim *keybind.VimState) Model {
	vp := viewport.New(80, 20)

	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.CharLimit = 4096
	ti.Width = 76

	return Model{
		vim:         vim,
		viewport:    vp,
		input:       ti,
		messages:    make([]*telegram.Message, 0),
		unreadSepAt: -1,
		sendKey:     "ctrl-enter",
	}
}

// SetSize sets the dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height - 4 // Room for input and reply indicator
	m.input.Width = width - 4
}

// SetChat sets the current chat
func (m *Model) SetChat(chatID int64, chatName string) {
	m.chatID = chatID
	m.chatName = chatName
	m.messages = make([]*telegram.Message, 0)
	m.selectedIdx = 0
	m.replyingTo = nil
	m.unreadSepAt = -1
}

// SetMessages sets the messages
func (m *Model) SetMessages(messages []*telegram.Message) {
	m.messages = messages
	if len(messages) > 0 {
		m.selectedIdx = len(messages) - 1
	}
}

// AppendMessage adds a new message
func (m *Model) AppendMessage(msg *telegram.Message) {
	m.messages = append(m.messages, msg)
	// Auto-scroll to bottom if already at bottom
	if m.selectedIdx >= len(m.messages)-2 {
		m.selectedIdx = len(m.messages) - 1
	}
}

// SetUnreadSeparator sets where the unread separator should appear
func (m *Model) SetUnreadSeparator(index int) {
	m.unreadSepAt = index
}

// SetSendKey sets the send keybinding configuration
func (m *Model) SetSendKey(key string) {
	m.sendKey = key
}

// SetOnSendMessage sets the send callback
func (m *Model) SetOnSendMessage(callback func(int64, string, int64) tea.Cmd) {
	m.onSendMessage = callback
}

// ChatID returns the current chat ID
func (m Model) ChatID() int64 {
	return m.chatID
}

// ChatName returns the current chat name
func (m Model) ChatName() string {
	return m.chatName
}

// FocusInput focuses the text input
func (m *Model) FocusInput() tea.Cmd {
	return m.input.Focus()
}

// BlurInput blurs the text input
func (m *Model) BlurInput() {
	m.input.Blur()
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	// Handle based on mode
	if m.vim.Mode() == keybind.ModeInsert {
		return m.handleInsertMode(msg, key)
	}

	return m.handleNormalMode(msg, key)
}

func (m Model) handleNormalMode(_ tea.KeyMsg, key string) (Model, tea.Cmd) {
	result := m.vim.ProcessKey(key)

	switch result.Action {
	case keybind.ActionMoveDown:
		for i := 0; i < result.Count; i++ {
			if m.selectedIdx < len(m.messages)-1 {
				m.selectedIdx++
			}
		}
		m.updateViewport()

	case keybind.ActionMoveUp:
		for i := 0; i < result.Count; i++ {
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		}
		m.updateViewport()

	case keybind.ActionJumpTop:
		m.selectedIdx = 0
		m.updateViewport()

	case keybind.ActionJumpBottom:
		if len(m.messages) > 0 {
			m.selectedIdx = len(m.messages) - 1
		}
		m.updateViewport()

	case keybind.ActionHalfPageDown:
		pageSize := m.viewport.Height / 2
		for i := 0; i < pageSize*result.Count; i++ {
			if m.selectedIdx < len(m.messages)-1 {
				m.selectedIdx++
			}
		}
		m.updateViewport()

	case keybind.ActionHalfPageUp:
		pageSize := m.viewport.Height / 2
		for i := 0; i < pageSize*result.Count; i++ {
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		}
		m.updateViewport()

	case keybind.ActionEnterInsert:
		m.vim.SetMode(keybind.ModeInsert)
		return m, m.input.Focus()

	case keybind.ActionReply:
		if len(m.messages) > 0 && m.selectedIdx < len(m.messages) {
			m.replyingTo = m.messages[m.selectedIdx]
			m.vim.SetMode(keybind.ModeInsert)
			return m, m.input.Focus()
		}
	}

	return m, nil
}

func (m Model) handleInsertMode(msg tea.KeyMsg, key string) (Model, tea.Cmd) {
	// Check for send key
	shouldSend := false
	if m.sendKey == "enter" && key == "enter" {
		shouldSend = true
	} else if m.sendKey == "ctrl-enter" && key == "ctrl+enter" {
		shouldSend = true
	}

	if shouldSend {
		return m.sendMessage()
	}

	// Check for escape
	if key == "esc" || key == "escape" {
		m.vim.SetMode(keybind.ModeNormal)
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
	m.replyingTo = nil

	// Call the send callback
	if m.onSendMessage != nil {
		return m, m.onSendMessage(m.chatID, text, replyToID)
	}

	return m, nil
}

func (m *Model) updateViewport() {
	// Ensure selected message is visible
	// This would involve scrolling the viewport
	// For now, we'll handle this in the View
}

// View renders the chat view
func (m Model) View() string {
	var parts []string

	// Messages area
	msgView := m.renderMessages()
	parts = append(parts, msgView)

	// Reply indicator
	if m.replyingTo != nil {
		replyText := truncate(m.replyingTo.Text, 50)
		indicator := fmt.Sprintf("Replying to %s: %s", m.replyingTo.SenderName, replyText)
		parts = append(parts, replyIndicatorStyle.Width(m.width).Render(indicator))
	}

	// Input area
	parts = append(parts, m.renderInput())

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) renderMessages() string {
	if len(m.messages) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Height(m.viewport.Height).
			Render("No messages yet. Press 'i' to start typing.")
	}

	var lines []string

	for i, msg := range m.messages {
		// Unread separator
		if i == m.unreadSepAt && m.unreadSepAt >= 0 {
			sep := strings.Repeat("─", m.width/2-5) + " NEW " + strings.Repeat("─", m.width/2-5)
			lines = append(lines, unreadSeparatorStyle.Render(sep))
		}

		line := m.renderMessage(msg, i == m.selectedIdx)
		lines = append(lines, line)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	// Ensure content fits in viewport
	return lipgloss.NewStyle().
		Height(m.viewport.Height).
		MaxHeight(m.viewport.Height).
		Render(content)
}

func (m Model) renderMessage(msg *telegram.Message, selected bool) string {
	var parts []string

	// Reply context if present
	if msg.ReplyTo != nil {
		replyText := truncate(msg.ReplyTo.Text, 40)
		context := fmt.Sprintf("↳ %s: %s", msg.ReplyTo.SenderName, replyText)
		parts = append(parts, replyContextStyle.Render(context))
	}

	// Sender
	var sender string
	if msg.IsOutgoing {
		sender = ownSenderStyle.Render("You")
	} else {
		sender = senderStyle.Render(msg.SenderName)
	}

	// Main message line
	msgLine := sender + ": " + msg.Text

	// Add time if selected
	if selected {
		timeStr := msg.Date.Format("15:04:05")
		msgLine += " " + timeStyle.Render(timeStr)
	}

	if selected {
		msgLine = selectedStyle.Render(msgLine)
	}

	parts = append(parts, msgLine)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) renderInput() string {
	placeholder := "Type a message..."
	if m.vim.Mode() == keybind.ModeNormal {
		placeholder = "Press 'i' to type, 'r' to reply"
	}
	m.input.Placeholder = placeholder

	return inputBorderStyle.Width(m.width - 2).Render(m.input.View())
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Message types for chat view updates

// NewMessageMsg is sent when a new message arrives
type NewMessageMsg struct {
	Message *telegram.Message
}

// MessageSentMsg is sent when a message was sent successfully
type MessageSentMsg struct {
	Message *telegram.Message
}

// MessageSendFailedMsg is sent when sending a message fails
type MessageSendFailedMsg struct {
	Error error
}

// MessagesLoadedMsg is sent when message history is loaded
type MessagesLoadedMsg struct {
	Messages []*telegram.Message
}
