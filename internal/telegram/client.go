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
		LogVerbosity:    1,
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
	me        *User
	mu        sync.RWMutex
	closed    bool

	// TDLib client (platform-specific implementation)
	impl clientImpl
}

// clientImpl is the interface for the platform-specific TDLib implementation
type clientImpl interface {
	start() error
	close() error
	sendPhoneNumber(phone string) error
	sendAuthCode(code string) error
	send2FAPassword(password string) error
	getChats(limit int) ([]*Chat, error)
	getChat(chatID int64) (*Chat, error)
	searchPublicChat(username string) (*Chat, error)
	getChatHistory(chatID int64, fromMessageID int64, limit int) ([]*Message, error)
	sendMessage(chatID int64, text string, replyToID int64) (*Message, error)
	markAsRead(chatID int64, messageIDs []int64) error
	downloadFile(fileID string, priority int) error
	cancelDownload(fileID string) error
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

	// Create platform-specific implementation
	impl, err := newClientImpl(c)
	if err != nil {
		return nil, err
	}
	c.impl = impl

	return c, nil
}

// Start initializes the TDLib client and starts processing updates
func (c *Client) Start() error {
	return c.impl.start()
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

// Me returns the current user
func (c *Client) Me() *User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.me
}

// SendPhoneNumber sends the phone number for authentication
func (c *Client) SendPhoneNumber(phone string) error {
	return c.impl.sendPhoneNumber(phone)
}

// SendAuthCode sends the authentication code
func (c *Client) SendAuthCode(code string) error {
	return c.impl.sendAuthCode(code)
}

// Send2FAPassword sends the two-factor authentication password
func (c *Client) Send2FAPassword(password string) error {
	return c.impl.send2FAPassword(password)
}

// GetChats returns the list of chats
func (c *Client) GetChats(limit int) ([]*Chat, error) {
	return c.impl.getChats(limit)
}

// GetChat returns a chat by ID
func (c *Client) GetChat(chatID int64) (*Chat, error) {
	c.mu.RLock()
	chat, ok := c.chats[chatID]
	c.mu.RUnlock()

	if ok {
		return chat, nil
	}

	return c.impl.getChat(chatID)
}

// SearchPublicChat searches for a public chat by username
func (c *Client) SearchPublicChat(username string) (*Chat, error) {
	return c.impl.searchPublicChat(username)
}

// GetChatHistory returns messages from a chat
func (c *Client) GetChatHistory(chatID int64, fromMessageID int64, limit int) ([]*Message, error) {
	return c.impl.getChatHistory(chatID, fromMessageID, limit)
}

// SendMessage sends a text message to a chat
func (c *Client) SendMessage(chatID int64, text string, replyToID int64) (*Message, error) {
	return c.impl.sendMessage(chatID, text, replyToID)
}

// MarkAsRead marks messages as read
func (c *Client) MarkAsRead(chatID int64, messageIDs []int64) error {
	return c.impl.markAsRead(chatID, messageIDs)
}

// DownloadFile downloads a file
func (c *Client) DownloadFile(fileID string, priority int) error {
	return c.impl.downloadFile(fileID, priority)
}

// CancelDownload cancels a file download
func (c *Client) CancelDownload(fileID string) error {
	return c.impl.cancelDownload(fileID)
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

	if c.impl != nil {
		_ = c.impl.close()
	}

	close(c.updates)
	return nil
}

// Internal methods used by implementations

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

func (c *Client) setMe(user *User) {
	c.mu.Lock()
	c.me = user
	c.mu.Unlock()
}
