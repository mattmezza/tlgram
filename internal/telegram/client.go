package telegram

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ClientConfig holds configuration for the Telegram client
type ClientConfig struct {
	APIID           int32
	APIHash         string
	SessionDir      string
	FilesDir        string
	DatabaseDir     string
	LogFile         string
	LogVerbosity    int
	UseTestDC       bool
	UseFileDatabase bool
	UseChatInfo     bool
	UseMessageDB    bool
	UseSecretChats  bool
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig(configDir string) ClientConfig {
	return ClientConfig{
		SessionDir:      filepath.Join(configDir, "session"),
		FilesDir:        filepath.Join(configDir, "files"),
		DatabaseDir:     filepath.Join(configDir, "database"),
		LogFile:         filepath.Join(configDir, "logs", "tdlib.log"),
		LogVerbosity:    2,
		UseTestDC:       false,
		UseFileDatabase: true,
		UseChatInfo:     true,
		UseMessageDB:    true,
		UseSecretChats:  false,
	}
}

// Client wraps the TDLib client
type Client struct {
	config    ClientConfig
	updates   chan Update
	authState AuthState
	connState ConnectionState
	chats     map[int64]*Chat
	users     map[int64]*User
	mu        sync.RWMutex
	closed    bool

	// Will hold actual TDLib client when integrated
	// tdlib *client.Client
}

// NewClient creates a new Telegram client
func NewClient(cfg ClientConfig) (*Client, error) {
	// Ensure directories exist with proper permissions
	dirs := []string{cfg.SessionDir, cfg.FilesDir, cfg.DatabaseDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create log directory
	if cfg.LogFile != "" {
		logDir := filepath.Dir(cfg.LogFile)
		if err := os.MkdirAll(logDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
	}

	c := &Client{
		config:    cfg,
		updates:   make(chan Update, 100),
		authState: AuthStateUnknown,
		connState: ConnectionStateUnknown,
		chats:     make(map[int64]*Chat),
		users:     make(map[int64]*User),
	}

	return c, nil
}

// Start initializes the TDLib client and starts processing updates
func (c *Client) Start() error {
	// TODO: Initialize TDLib client
	// This will be implemented when TDLib is integrated

	// For now, transition to waiting for phone number
	c.setAuthState(AuthStateWaitPhoneNumber)
	return nil
}

// Updates returns the channel for receiving updates
func (c *Client) Updates() <-chan Update {
	return c.updates
}

// AuthState returns the current authentication state
func (c *Client) AuthState() AuthState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.authState
}

// ConnectionState returns the current connection state
func (c *Client) ConnectionState() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connState
}

// SendPhoneNumber sends the phone number for authentication
func (c *Client) SendPhoneNumber(phone string) error {
	// TODO: Implement with TDLib
	// For now, simulate transition to code waiting
	c.setAuthState(AuthStateWaitCode)
	return nil
}

// SendAuthCode sends the authentication code
func (c *Client) SendAuthCode(code string) error {
	// TODO: Implement with TDLib
	// For now, simulate transition to ready (or 2FA if needed)
	c.setAuthState(AuthStateReady)
	c.setConnectionState(ConnectionStateReady)
	return nil
}

// Send2FAPassword sends the two-factor authentication password
func (c *Client) Send2FAPassword(password string) error {
	// TODO: Implement with TDLib
	c.setAuthState(AuthStateReady)
	c.setConnectionState(ConnectionStateReady)
	return nil
}

// GetChats returns the list of chats
func (c *Client) GetChats(limit int) ([]*Chat, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// TODO: Implement with TDLib
	// For now, return cached chats
	chats := make([]*Chat, 0, len(c.chats))
	for _, chat := range c.chats {
		chats = append(chats, chat)
		if len(chats) >= limit {
			break
		}
	}
	return chats, nil
}

// GetChat returns a chat by ID
func (c *Client) GetChat(chatID int64) (*Chat, error) {
	c.mu.RLock()
	chat, ok := c.chats[chatID]
	c.mu.RUnlock()

	if ok {
		return chat, nil
	}

	// TODO: Fetch from TDLib
	return nil, fmt.Errorf("chat not found: %d", chatID)
}

// SearchPublicChat searches for a public chat by username
func (c *Client) SearchPublicChat(username string) (*Chat, error) {
	// TODO: Implement with TDLib
	return nil, fmt.Errorf("not implemented")
}

// GetChatHistory returns messages from a chat
func (c *Client) GetChatHistory(chatID int64, fromMessageID int64, limit int) ([]*Message, error) {
	// TODO: Implement with TDLib
	return nil, nil
}

// SendMessage sends a text message to a chat
func (c *Client) SendMessage(chatID int64, text string, replyToID int64) (*Message, error) {
	// TODO: Implement with TDLib
	return nil, fmt.Errorf("not implemented")
}

// MarkAsRead marks messages as read
func (c *Client) MarkAsRead(chatID int64, messageIDs []int64) error {
	// TODO: Implement with TDLib
	return nil
}

// DownloadFile downloads a file
func (c *Client) DownloadFile(fileID string, priority int) error {
	// TODO: Implement with TDLib
	return fmt.Errorf("not implemented")
}

// CancelDownload cancels a file download
func (c *Client) CancelDownload(fileID string) error {
	// TODO: Implement with TDLib
	return nil
}

// Close closes the client
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	// TODO: Close TDLib client
	close(c.updates)
	return nil
}

// Internal methods

func (c *Client) setAuthState(state AuthState) {
	c.mu.Lock()
	c.authState = state
	c.mu.Unlock()

	c.sendUpdate(AuthStateUpdate{State: state})
}

func (c *Client) setConnectionState(state ConnectionState) {
	c.mu.Lock()
	c.connState = state
	c.mu.Unlock()

	c.sendUpdate(ConnectionStateUpdate{State: state})
}

func (c *Client) sendUpdate(update Update) {
	c.mu.RLock()
	closed := c.closed
	c.mu.RUnlock()

	if !closed {
		select {
		case c.updates <- update:
		default:
			// Channel full, drop update
		}
	}
}

func (c *Client) cacheChat(chat *Chat) {
	c.mu.Lock()
	c.chats[chat.ID] = chat
	c.mu.Unlock()
}

func (c *Client) cacheUser(user *User) {
	c.mu.Lock()
	c.users[user.ID] = user
	c.mu.Unlock()
}
