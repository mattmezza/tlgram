package telegram

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
)

// ClientConfig holds configuration for the Telegram client
type ClientConfig struct {
	APIID       int
	APIHash     string
	SessionDir  string
	Phone       string
	DeviceModel string
	SystemVer   string
	AppVersion  string
	LangCode    string
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig(configDir string) ClientConfig {
	return ClientConfig{
		SessionDir:  filepath.Join(configDir, "session"),
		DeviceModel: "Desktop",
		SystemVer:   "Linux",
		AppVersion:  "0.1.0",
		LangCode:    "en",
	}
}

// Client wraps the Telegram client
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

	// gotd client
	client     *telegram.Client
	api        *tg.Client
	dispatcher *updates.Manager
	ctx        context.Context
	cancel     context.CancelFunc

	// Auth flow state
	phoneCode     chan string
	password      chan string
	authErr       chan error
	pendingPhone  string
	phoneCodeHash string

	// Connection ready signal (sends error or nil)
	ready     chan error
	readyOnce sync.Once
}

// NewClient creates a new Telegram client
func NewClient(cfg ClientConfig) (*Client, error) {
	// Ensure session directory exists
	if err := os.MkdirAll(cfg.SessionDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Client{
		config:    cfg,
		updates:   make(chan Update, 100),
		authState: AuthStateUnknown,
		connState: ConnectionStateUnknown,
		chats:     make(map[int64]*Chat),
		users:     make(map[int64]*User),
		ctx:       ctx,
		cancel:    cancel,
		phoneCode: make(chan string, 1),
		password:  make(chan string, 1),
		authErr:   make(chan error, 1),
		ready:     make(chan error, 1),
	}

	return c, nil
}

// Start initializes the client and begins connection
func (c *Client) Start() error {
	// Create session storage
	sessionPath := filepath.Join(c.config.SessionDir, "session.json")
	sessionStorage := &session.FileStorage{
		Path: sessionPath,
	}

	// Create update dispatcher
	c.dispatcher = updates.New(updates.Config{
		Handler: c,
	})

	// Create client options
	opts := telegram.Options{
		SessionStorage: sessionStorage,
		UpdateHandler:  c.dispatcher,
		Device: telegram.DeviceConfig{
			DeviceModel:    c.config.DeviceModel,
			SystemVersion:  c.config.SystemVer,
			AppVersion:     c.config.AppVersion,
			LangCode:       c.config.LangCode,
			SystemLangCode: c.config.LangCode,
		},
	}

	// Create the client
	c.client = telegram.NewClient(c.config.APIID, c.config.APIHash, opts)

	// Start the client in background
	go c.run()

	// Wait for connection to be ready (with timeout)
	select {
	case err := <-c.ready:
		return err
	case <-time.After(60 * time.Second):
		return fmt.Errorf("connection timeout - check network and API credentials")
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

func (c *Client) run() {
	c.setConnectionState(ConnectionStateConnecting)

	// Helper to signal ready only once
	signalReady := func(err error) {
		c.readyOnce.Do(func() {
			c.ready <- err
		})
	}

	err := c.client.Run(c.ctx, func(ctx context.Context) error {
		c.api = c.client.API()
		c.setConnectionState(ConnectionStateReady)

		// Signal that we're ready (no error)
		signalReady(nil)

		// Check if already authorized
		status, err := c.client.Auth().Status(ctx)
		if err != nil {
			return err
		}

		if !status.Authorized {
			c.setAuthState(AuthStateWaitPhoneNumber)
			// Wait for auth to complete
			if err := c.handleAuth(ctx); err != nil {
				return err
			}
		} else {
			c.setAuthState(AuthStateReady)
			// Load initial data
			if err := c.loadInitialData(ctx); err != nil {
				c.sendUpdate(ErrorUpdate{Message: err.Error()})
			}
		}

		// Keep running until context is canceled
		<-ctx.Done()
		return ctx.Err()
	})

	// If Run() failed before callback executed, signal the error
	if err != nil && err != context.Canceled {
		signalReady(err)
		c.sendUpdate(ErrorUpdate{Message: err.Error()})
	}
}

func (c *Client) handleAuth(ctx context.Context) error {
	// Don't use automatic auth flow - we handle it interactively
	// Just wait for auth to be completed via SendPhoneNumber/SendAuthCode/Send2FAPassword
	// The auth state machine is driven by the UI

	// Wait for auth to complete or context to be canceled
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if c.AuthState() == AuthStateReady {
				// Auth completed, load initial data
				if err := c.loadInitialData(ctx); err != nil {
					c.sendUpdate(ErrorUpdate{Message: err.Error()})
				}
				return nil
			}
		}
	}
}

// Handle implements updates.Handler
func (c *Client) Handle(ctx context.Context, u tg.UpdatesClass) error {
	switch upd := u.(type) {
	case *tg.Updates:
		for _, update := range upd.Updates {
			c.handleUpdate(ctx, update, upd.Users, upd.Chats)
		}
	case *tg.UpdatesCombined:
		for _, update := range upd.Updates {
			c.handleUpdate(ctx, update, upd.Users, upd.Chats)
		}
	case *tg.UpdateShort:
		c.handleUpdate(ctx, upd.Update, nil, nil)
	case *tg.UpdateShortMessage:
		c.handleShortMessage(upd)
	case *tg.UpdateShortChatMessage:
		c.handleShortChatMessage(upd)
	}
	return nil
}

func (c *Client) handleUpdate(_ context.Context, update tg.UpdateClass, users []tg.UserClass, chats []tg.ChatClass) {
	// Cache users and chats
	for _, u := range users {
		if user, ok := u.(*tg.User); ok {
			c.cacheUser(c.convertUser(user))
		}
	}
	for _, ch := range chats {
		c.cacheChat(c.convertChatClass(ch))
	}

	switch u := update.(type) {
	case *tg.UpdateNewMessage:
		if msg, ok := u.Message.(*tg.Message); ok {
			c.sendUpdate(NewMessageUpdate{Message: c.convertMessage(msg)})
		}

	case *tg.UpdateNewChannelMessage:
		if msg, ok := u.Message.(*tg.Message); ok {
			c.sendUpdate(NewMessageUpdate{Message: c.convertMessage(msg)})
		}

	case *tg.UpdateEditMessage:
		if msg, ok := u.Message.(*tg.Message); ok {
			c.sendUpdate(MessageEditedUpdate{
				ChatID:    c.getPeerID(msg.PeerID),
				MessageID: int64(msg.ID),
				NewText:   msg.Message,
			})
		}

	case *tg.UpdateDeleteMessages:
		ids := make([]int64, len(u.Messages))
		for i, id := range u.Messages {
			ids[i] = int64(id)
		}
		c.sendUpdate(MessageDeletedUpdate{MessageIDs: ids})

	case *tg.UpdateReadHistoryInbox:
		c.sendUpdate(ChatReadUpdate{
			ChatID:        c.getPeerID(u.Peer),
			LastReadInbox: int64(u.MaxID),
		})

	case *tg.UpdateUserTyping:
		c.sendUpdate(TypingUpdate{
			ChatID: u.UserID,
			UserID: u.UserID,
			Action: c.convertTypingAction(u.Action),
		})

	case *tg.UpdateChatUserTyping:
		if peerUser, ok := u.FromID.(*tg.PeerUser); ok {
			c.sendUpdate(TypingUpdate{
				ChatID: u.ChatID,
				UserID: peerUser.UserID,
				Action: c.convertTypingAction(u.Action),
			})
		}

	case *tg.UpdateDialogUnreadMark:
		if dialogPeer, ok := u.Peer.(*tg.DialogPeer); ok {
			chatID := c.getPeerID(dialogPeer.Peer)
			c.sendUpdate(DialogUnreadUpdate{
				ChatID:       chatID,
				MarkedUnread: u.Unread,
			})
		}
	}
}

func (c *Client) handleShortMessage(u *tg.UpdateShortMessage) {
	msg := &Message{
		ID:         int64(u.ID),
		ChatID:     u.UserID,
		SenderID:   u.UserID,
		Text:       u.Message,
		Date:       time.Unix(int64(u.Date), 0),
		IsOutgoing: u.Out,
	}

	// Get sender name from cache
	c.mu.RLock()
	if user, ok := c.users[u.UserID]; ok {
		msg.SenderName = user.FullName()
	}
	isMe := c.me != nil && u.UserID == c.me.ID
	c.mu.RUnlock()

	if msg.SenderName == "" {
		if msg.IsOutgoing || isMe {
			msg.SenderName = "You"
		} else {
			msg.SenderName = fmt.Sprintf("User %d", u.UserID)
		}
	}

	c.sendUpdate(NewMessageUpdate{Message: msg})
}

func (c *Client) handleShortChatMessage(u *tg.UpdateShortChatMessage) {
	msg := &Message{
		ID:         int64(u.ID),
		ChatID:     u.ChatID,
		SenderID:   u.FromID,
		Text:       u.Message,
		Date:       time.Unix(int64(u.Date), 0),
		IsOutgoing: u.Out,
	}

	// Get sender name from cache
	c.mu.RLock()
	if user, ok := c.users[u.FromID]; ok {
		msg.SenderName = user.FullName()
	}
	isMe := c.me != nil && u.FromID == c.me.ID
	c.mu.RUnlock()

	if msg.SenderName == "" {
		if msg.IsOutgoing || isMe {
			msg.SenderName = "You"
		} else {
			msg.SenderName = fmt.Sprintf("User %d", u.FromID)
		}
	}

	c.sendUpdate(NewMessageUpdate{Message: msg})
}

func (c *Client) loadInitialData(ctx context.Context) error {
	// Get current user
	me, err := c.api.UsersGetUsers(ctx, []tg.InputUserClass{&tg.InputUserSelf{}})
	if err != nil {
		return err
	}
	if len(me) > 0 {
		if user, ok := me[0].(*tg.User); ok {
			c.setMe(c.convertUser(user))
		}
	}

	// Get dialogs (chats)
	dialogs, err := c.api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100,
	})
	if err != nil {
		return err
	}

	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		c.cacheDialogs(d.Users, d.Chats, d.Dialogs)
	case *tg.MessagesDialogsSlice:
		c.cacheDialogs(d.Users, d.Chats, d.Dialogs)
	}

	return nil
}

func (c *Client) cacheDialogs(users []tg.UserClass, chats []tg.ChatClass, _ []tg.DialogClass) {
	for _, u := range users {
		if user, ok := u.(*tg.User); ok {
			c.cacheUser(c.convertUser(user))
		}
	}
	for _, ch := range chats {
		c.cacheChat(c.convertChatClass(ch))
	}
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

// SendPhoneNumber initiates authentication with the given phone number
func (c *Client) SendPhoneNumber(phone string) error {
	c.pendingPhone = phone

	// Trigger auth flow
	go func() {
		ctx, cancel := context.WithTimeout(c.ctx, 60*time.Second)
		defer cancel()

		sentCodeClass, err := c.api.AuthSendCode(ctx, &tg.AuthSendCodeRequest{
			PhoneNumber: phone,
			APIID:       c.config.APIID,
			APIHash:     c.config.APIHash,
			Settings:    tg.CodeSettings{},
		})
		if err != nil {
			c.sendUpdate(ErrorUpdate{Message: fmt.Sprintf("SendCode error: %v", err)})
			return
		}
		// Store the phone code hash for later use in SignIn
		switch v := sentCodeClass.(type) {
		case *tg.AuthSentCode:
			c.phoneCodeHash = v.PhoneCodeHash
			c.setAuthState(AuthStateWaitCode)
		case *tg.AuthSentCodeSuccess:
			// Already authorized via another session
			if auth, ok := v.Authorization.(*tg.AuthAuthorization); ok {
				if user, ok := auth.User.(*tg.User); ok {
					c.setMe(c.convertUser(user))
				}
			}
			c.setAuthState(AuthStateReady)
		default:
			c.sendUpdate(ErrorUpdate{Message: fmt.Sprintf("Unexpected response type: %T", sentCodeClass)})
		}
	}()

	return nil
}

// SendAuthCode sends the authentication code
func (c *Client) SendAuthCode(code string) error {
	go func() {
		ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
		defer cancel()

		result, err := c.api.AuthSignIn(ctx, &tg.AuthSignInRequest{
			PhoneNumber:   c.pendingPhone,
			PhoneCodeHash: c.phoneCodeHash,
			PhoneCode:     code,
		})
		if err != nil {
			// Check if 2FA is required
			if errors.Is(err, auth.ErrPasswordAuthNeeded) {
				c.setAuthState(AuthStateWaitPassword)
				return
			}
			c.sendUpdate(ErrorUpdate{Message: err.Error()})
			return
		}

		if authorization, ok := result.(*tg.AuthAuthorization); ok {
			if user, ok := authorization.User.(*tg.User); ok {
				c.setMe(c.convertUser(user))
			}
			c.setAuthState(AuthStateReady)
		}
	}()
	return nil
}

// Send2FAPassword sends the two-factor authentication password
func (c *Client) Send2FAPassword(pwd string) error {
	go func() {
		ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
		defer cancel()

		// Get password info to compute SRP
		password, err := c.client.Auth().Password(ctx, pwd)
		if err != nil {
			c.sendUpdate(ErrorUpdate{Message: err.Error()})
			return
		}

		if user, ok := password.User.(*tg.User); ok {
			c.setMe(c.convertUser(user))
		}
		c.setAuthState(AuthStateReady)
	}()
	return nil
}

// GetChats returns the list of chats
func (c *Client) GetChats(limit int) ([]*Chat, error) {
	if c.api == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	dialogs, err := c.api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	var chats []*Chat
	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		chats = c.extractChatsFromDialogs(d.Users, d.Chats, d.Dialogs)
	case *tg.MessagesDialogsSlice:
		chats = c.extractChatsFromDialogs(d.Users, d.Chats, d.Dialogs)
	}

	return chats, nil
}

func (c *Client) extractChatsFromDialogs(users []tg.UserClass, chats []tg.ChatClass, dialogs []tg.DialogClass) []*Chat {
	// Build lookup maps
	userMap := make(map[int64]*tg.User)
	for _, u := range users {
		if user, ok := u.(*tg.User); ok {
			userMap[user.ID] = user
			c.cacheUser(c.convertUser(user))
		}
	}

	chatMap := make(map[int64]tg.ChatClass)
	for _, ch := range chats {
		switch chat := ch.(type) {
		case *tg.Chat:
			chatMap[chat.ID] = ch
		case *tg.Channel:
			chatMap[chat.ID] = ch
		}
	}

	var result []*Chat
	for _, d := range dialogs {
		dialog, ok := d.(*tg.Dialog)
		if !ok {
			continue
		}

		var chat *Chat
		switch peer := dialog.Peer.(type) {
		case *tg.PeerUser:
			if user, ok := userMap[peer.UserID]; ok {
				chat = c.convertUserToChat(user)
				chat.UnreadCount = dialog.UnreadCount
				chat.LastReadInboxID = int64(dialog.ReadInboxMaxID)
				chat.MarkedUnread = dialog.UnreadMark
			}
		case *tg.PeerChat:
			if ch, ok := chatMap[peer.ChatID]; ok {
				chat = c.convertChatClass(ch)
				chat.UnreadCount = dialog.UnreadCount
				chat.LastReadInboxID = int64(dialog.ReadInboxMaxID)
				chat.MarkedUnread = dialog.UnreadMark
			}
		case *tg.PeerChannel:
			if ch, ok := chatMap[peer.ChannelID]; ok {
				chat = c.convertChatClass(ch)
				chat.UnreadCount = dialog.UnreadCount
				chat.LastReadInboxID = int64(dialog.ReadInboxMaxID)
				chat.MarkedUnread = dialog.UnreadMark
			}
		}

		if chat != nil {
			c.cacheChat(chat)
			result = append(result, chat)
		}
	}

	return result
}

// GetChat returns a chat by ID
func (c *Client) GetChat(chatID int64) (*Chat, error) {
	c.mu.RLock()
	chat, ok := c.chats[chatID]
	c.mu.RUnlock()

	if ok {
		return chat, nil
	}

	return nil, fmt.Errorf("chat not found: %d", chatID)
}

// SearchPublicChat searches for a public chat by username
func (c *Client) SearchPublicChat(username string) (*Chat, error) {
	if c.api == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	resolved, err := c.api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		return nil, err
	}

	// Cache users and chats
	for _, u := range resolved.Users {
		if user, ok := u.(*tg.User); ok {
			c.cacheUser(c.convertUser(user))
		}
	}
	for _, ch := range resolved.Chats {
		c.cacheChat(c.convertChatClass(ch))
	}

	// Return the resolved peer
	switch peer := resolved.Peer.(type) {
	case *tg.PeerUser:
		return c.GetChat(peer.UserID)
	case *tg.PeerChat:
		return c.GetChat(peer.ChatID)
	case *tg.PeerChannel:
		return c.GetChat(peer.ChannelID)
	}

	return nil, fmt.Errorf("could not resolve username: %s", username)
}

// GetChatHistory returns messages from a chat
func (c *Client) GetChatHistory(chatID int64, fromMessageID int64, limit int) ([]*Message, error) {
	if c.api == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	peer := c.getInputPeer(chatID)
	if peer == nil {
		return nil, fmt.Errorf("unknown chat: %d", chatID)
	}

	history, err := c.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:      peer,
		OffsetID:  int(fromMessageID),
		Limit:     limit,
		AddOffset: 0,
	})
	if err != nil {
		return nil, err
	}

	var messages []*Message
	switch h := history.(type) {
	case *tg.MessagesMessages:
		messages = c.extractMessages(h.Messages, h.Users)
	case *tg.MessagesMessagesSlice:
		messages = c.extractMessages(h.Messages, h.Users)
	case *tg.MessagesChannelMessages:
		messages = c.extractMessages(h.Messages, h.Users)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (c *Client) extractMessages(msgs []tg.MessageClass, users []tg.UserClass) []*Message {
	// Cache users
	for _, u := range users {
		if user, ok := u.(*tg.User); ok {
			c.cacheUser(c.convertUser(user))
		}
	}

	var result []*Message
	for _, m := range msgs {
		if msg, ok := m.(*tg.Message); ok {
			result = append(result, c.convertMessage(msg))
		}
	}
	return result
}

// SendMessage sends a text message to a chat
func (c *Client) SendMessage(chatID int64, text string, replyToID int64) (*Message, error) {
	if c.api == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	peer := c.getInputPeer(chatID)
	if peer == nil {
		return nil, fmt.Errorf("unknown chat: %d", chatID)
	}

	req := &tg.MessagesSendMessageRequest{
		Peer:     peer,
		Message:  text,
		RandomID: time.Now().UnixNano(),
	}

	if replyToID > 0 {
		req.ReplyTo = &tg.InputReplyToMessage{
			ReplyToMsgID: int(replyToID),
		}
	}

	result, err := c.api.MessagesSendMessage(ctx, req)
	if err != nil {
		return nil, err
	}

	// Extract sent message info
	switch r := result.(type) {
	case *tg.UpdateShortSentMessage:
		return &Message{
			ID:         int64(r.ID),
			ChatID:     chatID,
			Text:       text,
			Date:       time.Unix(int64(r.Date), 0),
			IsOutgoing: true,
			SenderName: "You",
		}, nil
	case *tg.Updates:
		// For groups/channels, find the message in the updates
		for _, update := range r.Updates {
			if msgUpdate, ok := update.(*tg.UpdateNewMessage); ok {
				if msg, ok := msgUpdate.Message.(*tg.Message); ok {
					return &Message{
						ID:         int64(msg.ID),
						ChatID:     chatID,
						Text:       text,
						Date:       time.Unix(int64(msg.Date), 0),
						IsOutgoing: true,
						SenderName: "You",
					}, nil
				}
			}
			if msgUpdate, ok := update.(*tg.UpdateNewChannelMessage); ok {
				if msg, ok := msgUpdate.Message.(*tg.Message); ok {
					return &Message{
						ID:         int64(msg.ID),
						ChatID:     chatID,
						Text:       text,
						Date:       time.Unix(int64(msg.Date), 0),
						IsOutgoing: true,
						SenderName: "You",
					}, nil
				}
			}
		}
	}

	// Fallback - this shouldn't happen but log for debugging
	return &Message{
		ID:         0, // Invalid ID - won't be editable
		ChatID:     chatID,
		Text:       text,
		Date:       time.Now(),
		IsOutgoing: true,
		SenderName: "You",
	}, nil
}

// EditMessage edits a message in a chat
func (c *Client) EditMessage(chatID int64, messageID int64, newText string) error {
	if c.api == nil {
		return fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	peer := c.getInputPeer(chatID)
	if peer == nil {
		return fmt.Errorf("unknown chat: %d", chatID)
	}

	req := &tg.MessagesEditMessageRequest{
		Peer: peer,
		ID:   int(messageID),
	}
	req.SetMessage(newText)

	_, err := c.api.MessagesEditMessage(ctx, req)
	return err
}

// DeleteMessages deletes messages from a chat
func (c *Client) DeleteMessages(chatID int64, messageIDs []int64, revoke bool) error {
	if c.api == nil || len(messageIDs) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	// Convert int64 to int
	ids := make([]int, len(messageIDs))
	for i, id := range messageIDs {
		ids[i] = int(id)
	}

	// Check chat type to determine which API to use
	c.mu.RLock()
	chat, ok := c.chats[chatID]
	c.mu.RUnlock()

	if ok && (chat.Type == ChatTypeSupergroup || chat.Type == ChatTypeChannel) {
		// Use channels.deleteMessages for channels/supergroups
		_, err := c.api.ChannelsDeleteMessages(ctx, &tg.ChannelsDeleteMessagesRequest{
			Channel: &tg.InputChannel{
				ChannelID:  chatID,
				AccessHash: chat.AccessHash,
			},
			ID: ids,
		})
		return err
	}

	// Use messages.deleteMessages for private chats and basic groups
	_, err := c.api.MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{
		Revoke: revoke, // Delete for everyone if true
		ID:     ids,
	})
	return err
}

// MarkAsRead marks messages as read up to maxMsgID
func (c *Client) MarkAsRead(chatID int64, maxMsgID int64) error {
	if c.api == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	// Check chat type to determine which API to use
	c.mu.RLock()
	chat, ok := c.chats[chatID]
	c.mu.RUnlock()

	if ok && (chat.Type == ChatTypeSupergroup || chat.Type == ChatTypeChannel) {
		// Use channels.readHistory for channels/supergroups
		_, err := c.api.ChannelsReadHistory(ctx, &tg.ChannelsReadHistoryRequest{
			Channel: &tg.InputChannel{
				ChannelID:  chatID,
				AccessHash: chat.AccessHash,
			},
			MaxID: int(maxMsgID),
		})
		return err
	}

	// Use messages.readHistory for private chats and basic groups
	peer := c.getInputPeer(chatID)
	if peer == nil {
		return nil
	}
	_, err := c.api.MessagesReadHistory(ctx, &tg.MessagesReadHistoryRequest{
		Peer:  peer,
		MaxID: int(maxMsgID),
	})
	return err
}

// MarkDialogUnread marks a dialog as unread (or clears the unread mark)
func (c *Client) MarkDialogUnread(chatID int64, unread bool) error {
	if c.api == nil {
		return fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	peer := c.getInputPeer(chatID)
	if peer == nil {
		return fmt.Errorf("unknown chat: %d", chatID)
	}

	// Convert InputPeer to InputDialogPeer
	dialogPeer := &tg.InputDialogPeer{Peer: peer}

	_, err := c.api.MessagesMarkDialogUnread(ctx, &tg.MessagesMarkDialogUnreadRequest{
		Peer:   dialogPeer,
		Unread: unread,
	})
	return err
}

// DownloadFile downloads a file (not implemented yet)
func (c *Client) DownloadFile(fileID string, priority int) error {
	return fmt.Errorf("file download not implemented")
}

// CancelDownload cancels a file download
func (c *Client) CancelDownload(fileID string) error {
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

	c.cancel()
	close(c.updates)
	return nil
}

// Helper methods

func (c *Client) getInputPeer(chatID int64) tg.InputPeerClass {
	c.mu.RLock()
	chat, ok := c.chats[chatID]
	c.mu.RUnlock()

	if !ok {
		// Try as user
		return &tg.InputPeerUser{UserID: chatID}
	}

	switch chat.Type {
	case ChatTypePrivate:
		return &tg.InputPeerUser{UserID: chatID, AccessHash: chat.AccessHash}
	case ChatTypeGroup:
		return &tg.InputPeerChat{ChatID: chatID}
	case ChatTypeSupergroup, ChatTypeChannel:
		return &tg.InputPeerChannel{ChannelID: chatID, AccessHash: chat.AccessHash}
	default:
		return &tg.InputPeerUser{UserID: chatID}
	}
}

func (c *Client) getPeerID(peer tg.PeerClass) int64 {
	switch p := peer.(type) {
	case *tg.PeerUser:
		return p.UserID
	case *tg.PeerChat:
		return p.ChatID
	case *tg.PeerChannel:
		return p.ChannelID
	}
	return 0
}

func (c *Client) convertMessage(msg *tg.Message) *Message {
	m := &Message{
		ID:         int64(msg.ID),
		ChatID:     c.getPeerID(msg.PeerID),
		Text:       msg.Message,
		Date:       time.Unix(int64(msg.Date), 0),
		IsOutgoing: msg.Out,
		IsEdited:   msg.EditDate > 0,
	}

	// Get sender info
	if msg.FromID != nil {
		switch from := msg.FromID.(type) {
		case *tg.PeerUser:
			m.SenderID = from.UserID
			c.mu.RLock()
			if user, ok := c.users[from.UserID]; ok {
				m.SenderName = user.FullName()
				if user.Username != "" {
					m.SenderUsername = "@" + user.Username
				}
			}
			c.mu.RUnlock()
		case *tg.PeerChat:
			m.SenderID = from.ChatID
		case *tg.PeerChannel:
			m.SenderID = from.ChannelID
		}
	} else {
		// FromID is nil - for private chats, sender is the peer user (if not outgoing)
		if !msg.Out {
			if peerUser, ok := msg.PeerID.(*tg.PeerUser); ok {
				m.SenderID = peerUser.UserID
				c.mu.RLock()
				if user, ok := c.users[peerUser.UserID]; ok {
					m.SenderName = user.FullName()
					if user.Username != "" {
						m.SenderUsername = "@" + user.Username
					}
				}
				c.mu.RUnlock()
			}
		}
	}

	if m.SenderName == "" {
		// Check if it's our own message (either outgoing or sender is current user)
		c.mu.RLock()
		isMe := c.me != nil && m.SenderID == c.me.ID
		// Also try to get the sender name from cache if we have a sender ID
		if !isMe && m.SenderID != 0 {
			if user, ok := c.users[m.SenderID]; ok {
				m.SenderName = user.FullName()
				if user.Username != "" {
					m.SenderUsername = "@" + user.Username
				}
			}
		}
		// For private chats, try to get name from chat cache as fallback
		if m.SenderName == "" && !isMe && m.ChatID != 0 {
			if chat, ok := c.chats[m.ChatID]; ok && chat.Type == ChatTypePrivate {
				m.SenderName = chat.Title
				if chat.Username != "" {
					m.SenderUsername = "@" + chat.Username
				}
			}
		}
		c.mu.RUnlock()

		if m.SenderName == "" {
			if m.IsOutgoing || isMe {
				m.SenderName = "You"
			} else if m.SenderID != 0 {
				m.SenderName = fmt.Sprintf("User %d", m.SenderID)
			} else {
				m.SenderName = "Unknown"
			}
		}
	}

	// Handle reply
	if reply, ok := msg.ReplyTo.(*tg.MessageReplyHeader); ok {
		m.ReplyToID = int64(reply.ReplyToMsgID)
	}

	// Handle media
	if msg.Media != nil {
		m.Media = c.convertMedia(msg.Media)
		if m.Text == "" && m.Media != nil {
			m.Text = fmt.Sprintf("[%s]", m.Media.Type)
		}
	}

	return m
}

func (c *Client) convertMedia(media tg.MessageMediaClass) *MediaInfo {
	switch m := media.(type) {
	case *tg.MessageMediaPhoto:
		return &MediaInfo{Type: MediaTypePhoto}
	case *tg.MessageMediaDocument:
		if doc, ok := m.Document.(*tg.Document); ok {
			info := &MediaInfo{
				Type:     MediaTypeDocument,
				FileName: getDocumentFilename(doc),
				FileSize: doc.Size,
				MimeType: doc.MimeType,
			}
			// Determine specific type from attributes
			for _, attr := range doc.Attributes {
				switch attr.(type) {
				case *tg.DocumentAttributeVideo:
					info.Type = MediaTypeVideo
				case *tg.DocumentAttributeAudio:
					info.Type = MediaTypeAudio
				case *tg.DocumentAttributeSticker:
					info.Type = MediaTypeSticker
				case *tg.DocumentAttributeAnimated:
					info.Type = MediaTypeAnimation
				}
			}
			return info
		}
	case *tg.MessageMediaGeo:
		return &MediaInfo{Type: MediaTypeDocument}
	case *tg.MessageMediaContact:
		return &MediaInfo{Type: MediaTypeDocument}
	}
	return nil
}

func getDocumentFilename(doc *tg.Document) string {
	for _, attr := range doc.Attributes {
		if filename, ok := attr.(*tg.DocumentAttributeFilename); ok {
			return filename.FileName
		}
	}
	return ""
}

func (c *Client) convertUser(user *tg.User) *User {
	u := &User{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Phone:     user.Phone,
	}
	if user.Username != "" {
		u.Username = user.Username
	} else if len(user.Usernames) > 0 {
		u.Username = user.Usernames[0].Username
	}
	return u
}

func (c *Client) convertUserToChat(user *tg.User) *Chat {
	name := user.FirstName
	if user.LastName != "" {
		name += " " + user.LastName
	}
	return &Chat{
		ID:         user.ID,
		AccessHash: user.AccessHash,
		Type:       ChatTypePrivate,
		Title:      name,
		Username:   user.Username,
	}
}

func (c *Client) convertChatClass(ch tg.ChatClass) *Chat {
	switch chat := ch.(type) {
	case *tg.Chat:
		return &Chat{
			ID:    chat.ID,
			Type:  ChatTypeGroup,
			Title: chat.Title,
		}
	case *tg.Channel:
		chatType := ChatTypeSupergroup
		if chat.Broadcast {
			chatType = ChatTypeChannel
		}
		return &Chat{
			ID:         chat.ID,
			AccessHash: chat.AccessHash,
			Type:       chatType,
			Title:      chat.Title,
			Username:   chat.Username,
		}
	case *tg.ChatForbidden:
		return &Chat{
			ID:    chat.ID,
			Type:  ChatTypeGroup,
			Title: chat.Title,
		}
	case *tg.ChannelForbidden:
		return &Chat{
			ID:         chat.ID,
			AccessHash: chat.AccessHash,
			Type:       ChatTypeChannel,
			Title:      chat.Title,
		}
	}
	return nil
}

func (c *Client) convertTypingAction(action tg.SendMessageActionClass) TypingAction {
	switch action.(type) {
	case *tg.SendMessageTypingAction:
		return TypingActionTyping
	case *tg.SendMessageRecordVideoAction:
		return TypingActionRecordingVideo
	case *tg.SendMessageRecordAudioAction:
		return TypingActionRecordingVoice
	case *tg.SendMessageUploadDocumentAction:
		return TypingActionUploadingDocument
	case *tg.SendMessageUploadPhotoAction:
		return TypingActionUploadingPhoto
	case *tg.SendMessageUploadVideoAction:
		return TypingActionUploadingVideo
	case *tg.SendMessageCancelAction:
		return TypingActionCancel
	}
	return TypingActionTyping
}

// Internal state management

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
	if chat == nil {
		return
	}
	c.mu.Lock()
	c.chats[chat.ID] = chat
	c.mu.Unlock()
}

func (c *Client) cacheUser(user *User) {
	if user == nil {
		return
	}
	c.mu.Lock()
	c.users[user.ID] = user
	c.mu.Unlock()
}

func (c *Client) setMe(user *User) {
	c.mu.Lock()
	c.me = user
	c.mu.Unlock()
}
