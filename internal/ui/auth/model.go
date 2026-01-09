package auth

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattmezza/tlgram/internal/telegram"
)

// Messages returned by auth view for parent to handle
type PhoneSubmitMsg struct{ Phone string }
type CodeSubmitMsg struct{ Code string }
type PasswordSubmitMsg struct{ Password string }

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginBottom(1)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	inputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("241")).
			Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// Model represents the authentication UI
type Model struct {
	width  int
	height int

	state    telegram.AuthState
	input    textinput.Model
	errorMsg string
	loading  bool
}

// New creates a new auth model
func New() Model {
	ti := textinput.New()
	ti.Placeholder = "+1234567890"
	ti.Focus()
	ti.CharLimit = 20
	ti.Width = 30

	return Model{
		state: telegram.AuthStateUnknown, // Start in loading state until client connects
		input: ti,
	}
}

// SetState sets the current auth state
func (m *Model) SetState(state telegram.AuthState) {
	m.state = state
	m.errorMsg = ""
	m.loading = false

	// Update input for the new state
	m.input.Reset()
	switch state {
	case telegram.AuthStateWaitPhoneNumber:
		m.input.Placeholder = "+1234567890"
		m.input.EchoMode = textinput.EchoNormal
		m.input.CharLimit = 20
	case telegram.AuthStateWaitCode:
		m.input.Placeholder = "12345"
		m.input.EchoMode = textinput.EchoNormal
		m.input.CharLimit = 10
	case telegram.AuthStateWaitPassword:
		m.input.Placeholder = "Password"
		m.input.EchoMode = textinput.EchoPassword
		m.input.CharLimit = 100
	}
	m.input.Focus()
}

// SetError sets an error message
func (m *Model) SetError(err string) {
	m.errorMsg = err
	m.loading = false
}

// SetLoading sets the loading state
func (m *Model) SetLoading(loading bool) {
	m.loading = loading
}

// SetSize sets the dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.loading {
				return m, nil
			}
			return m.handleSubmit()
		case "esc":
			m.input.Reset()
			return m, nil
		}
	}

	// Forward to text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) handleSubmit() (Model, tea.Cmd) {
	// Don't submit if we're in loading/unknown state (client not ready)
	if m.state == telegram.AuthStateUnknown || m.state == telegram.AuthStateReady {
		return m, nil
	}

	value := strings.TrimSpace(m.input.Value())
	if value == "" {
		return m, nil
	}

	m.loading = true

	switch m.state {
	case telegram.AuthStateWaitPhoneNumber:
		return m, func() tea.Msg { return PhoneSubmitMsg{Phone: value} }
	case telegram.AuthStateWaitCode:
		return m, func() tea.Msg { return CodeSubmitMsg{Code: value} }
	case telegram.AuthStateWaitPassword:
		return m, func() tea.Msg { return PasswordSubmitMsg{Password: value} }
	}

	return m, nil
}

// View implements tea.Model
func (m Model) View() string {
	switch m.state {
	case telegram.AuthStateWaitPhoneNumber:
		return m.viewPhoneInput()
	case telegram.AuthStateWaitCode:
		return m.viewCodeInput()
	case telegram.AuthStateWaitPassword:
		return m.viewPasswordInput()
	case telegram.AuthStateReady:
		return m.viewReady()
	default:
		return m.viewLoading()
	}
}

func (m Model) viewPhoneInput() string {
	var content strings.Builder

	content.WriteString(titleStyle.Render("tlgram - Terminal Telegram Client"))
	content.WriteString("\n\n")
	content.WriteString(promptStyle.Render("Enter your phone number:"))
	content.WriteString("\n")
	content.WriteString(inputStyle.Render(m.input.View()))
	content.WriteString("\n")

	if m.errorMsg != "" {
		content.WriteString("\n")
		content.WriteString(errorStyle.Render(m.errorMsg))
		content.WriteString("\n")
	}

	if m.loading {
		content.WriteString("\n")
		content.WriteString("Sending code...")
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(helpStyle.Render("Format: +1234567890 (with country code)"))
	content.WriteString("\n")
	content.WriteString(helpStyle.Render("Press Enter to submit, Esc to clear"))

	return content.String()
}

func (m Model) viewCodeInput() string {
	var content strings.Builder

	content.WriteString(titleStyle.Render("Verification Code"))
	content.WriteString("\n\n")
	content.WriteString(promptStyle.Render("Enter the code from Telegram:"))
	content.WriteString("\n")
	content.WriteString(inputStyle.Render(m.input.View()))
	content.WriteString("\n")

	if m.errorMsg != "" {
		content.WriteString("\n")
		content.WriteString(errorStyle.Render(m.errorMsg))
		content.WriteString("\n")
	}

	if m.loading {
		content.WriteString("\n")
		content.WriteString("Verifying...")
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(helpStyle.Render("Check your Telegram app for the code"))
	content.WriteString("\n")
	content.WriteString(helpStyle.Render("Press Enter to submit"))

	return content.String()
}

func (m Model) viewPasswordInput() string {
	var content strings.Builder

	content.WriteString(titleStyle.Render("Two-Factor Authentication"))
	content.WriteString("\n\n")
	content.WriteString(promptStyle.Render("Enter your 2FA password:"))
	content.WriteString("\n")
	content.WriteString(inputStyle.Render(m.input.View()))
	content.WriteString("\n")

	if m.errorMsg != "" {
		content.WriteString("\n")
		content.WriteString(errorStyle.Render(m.errorMsg))
		content.WriteString("\n")
	}

	if m.loading {
		content.WriteString("\n")
		content.WriteString("Authenticating...")
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(helpStyle.Render("Press Enter to submit"))

	return content.String()
}

func (m Model) viewReady() string {
	return titleStyle.Render("Authenticated successfully!")
}

func (m Model) viewLoading() string {
	var content strings.Builder
	content.WriteString(titleStyle.Render("tlgram - Terminal Telegram Client"))
	content.WriteString("\n\n")
	content.WriteString("Connecting to Telegram...")
	content.WriteString("\n")

	if m.errorMsg != "" {
		content.WriteString("\n")
		content.WriteString(errorStyle.Render(m.errorMsg))
		content.WriteString("\n")
	}

	return content.String()
}
