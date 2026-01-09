//go:build tdlib

package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/zelenin/go-tdlib/client"
)

// tdlibImpl is the TDLib-based implementation
type tdlibImpl struct {
	client   *Client
	tdClient *client.Client
	ctx      context.Context
	cancel   context.CancelFunc
}

func newClientImpl(c *Client) (clientImpl, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &tdlibImpl{
		client: c,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (t *tdlibImpl) start() error {
	// Set log verbosity
	_, err := client.SetLogVerbosityLevel(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: int32(t.client.config.LogVerbosity),
	})
	if err != nil {
		return fmt.Errorf("failed to set log verbosity: %w", err)
	}

	// Create authorization state handler
	authorizer := client.ClientAuthorizer()
	go t.handleAuthorizationState(authorizer)

	// Set API credentials
	authorizer.TdlibParameters <- &client.SetTdlibParametersRequest{
		UseTestDc:           t.client.config.UseTestDC,
		DatabaseDirectory:   t.client.config.DatabaseDir,
		FilesDirectory:      t.client.config.FilesDir,
		UseFileDatabase:     t.client.config.UseFileDatabase,
		UseChatInfoDatabase: t.client.config.UseChatInfo,
		UseMessageDatabase:  t.client.config.UseMessageDB,
		UseSecretChats:      t.client.config.UseSecretChats,
		ApiId:               t.client.config.APIID,
		ApiHash:             t.client.config.APIHash,
		SystemLanguageCode:  "en",
		DeviceModel:         "Desktop",
		SystemVersion:       "Linux",
		ApplicationVersion:  "0.1.0",
	}

	// Create TDLib client
	tdClient, err := client.NewClient(authorizer)
	if err != nil {
		return fmt.Errorf("failed to create TDLib client: %w", err)
	}
	t.tdClient = tdClient

	// Start update listener
	go t.listenForUpdates()

	return nil
}

func (t *tdlibImpl) close() error {
	t.cancel()
	if t.tdClient != nil {
		_, _ = t.tdClient.Close()
	}
	return nil
}

func (t *tdlibImpl) handleAuthorizationState(authorizer *client.ClientAuthorizer) {
	for {
		select {
		case <-t.ctx.Done():
			return
		case state, ok := <-authorizer.State:
			if !ok {
				return
			}
			t.processAuthState(state)
		}
	}
}

func (t *tdlibImpl) processAuthState(state client.AuthorizationState) {
	switch state.AuthorizationStateType() {
	case client.TypeAuthorizationStateWaitPhoneNumber:
		t.client.setAuthState(AuthStateWaitPhoneNumber)

	case client.TypeAuthorizationStateWaitCode:
		t.client.setAuthState(AuthStateWaitCode)

	case client.TypeAuthorizationStateWaitPassword:
		t.client.setAuthState(AuthStateWaitPassword)

	case client.TypeAuthorizationStateReady:
		t.client.setAuthState(AuthStateReady)
		t.client.setConnectionState(ConnectionStateReady)
		go t.loadInitialData()

	case client.TypeAuthorizationStateLoggingOut:
		t.client.setAuthState(AuthStateLoggingOut)

	case client.TypeAuthorizationStateClosed:
		t.client.setAuthState(AuthStateClosed)
	}
}

func (t *tdlibImpl) listenForUpdates() {
	listener := t.tdClient.GetListener()
	defer listener.Close()

	for {
		select {
		case <-t.ctx.Done():
			return
		case update, ok := <-listener.Updates:
			if !ok {
				return
			}
			t.processUpdate(update)
		}
	}
}

func (t *tdlibImpl) processUpdate(update client.Type) {
	switch u := update.(type) {
	case *client.UpdateConnectionState:
		t.handleConnectionState(u)

	case *client.UpdateNewMessage:
		t.handleNewMessage(u)

	case *client.UpdateMessageContent:
		t.handleMessageEdit(u)

	case *client.UpdateDeleteMessages:
		t.handleDeleteMessages(u)

	case *client.UpdateNewChat:
		t.handleNewChat(u)

	case *client.UpdateChatTitle:
		t.handleChatTitle(u)

	case *client.UpdateChatReadInbox:
		t.handleChatRead(u)

	case *client.UpdateChatLastMessage:
		t.handleChatLastMessage(u)

	case *client.UpdateUser:
		t.handleUserUpdate(u)

	case *client.UpdateChatAction:
		t.handleTyping(u)

	case *client.UpdateFile:
		t.handleFileUpdate(u)
	}
}

func (t *tdlibImpl) handleConnectionState(u *client.UpdateConnectionState) {
	switch u.State.ConnectionStateType() {
	case client.TypeConnectionStateConnecting:
		t.client.setConnectionState(ConnectionStateConnecting)
	case client.TypeConnectionStateConnectingToProxy:
		t.client.setConnectionState(ConnectionStateConnectingToProxy)
	case client.TypeConnectionStateReady:
		t.client.setConnectionState(ConnectionStateReady)
	case client.TypeConnectionStateUpdating:
		t.client.setConnectionState(ConnectionStateUpdating)
	case client.TypeConnectionStateWaitingForNetwork:
		t.client.setConnectionState(ConnectionStateWaitingForNetwork)
	}
}

func (t *tdlibImpl) handleNewMessage(u *client.UpdateNewMessage) {
	msg := t.convertMessage(u.Message)
	if msg != nil {
		t.client.sendUpdate(NewMessageUpdate{Message: msg})
	}
}

func (t *tdlibImpl) handleMessageEdit(u *client.UpdateMessageContent) {
	var newText string
	if textContent, ok := u.NewContent.(*client.MessageText); ok {
		newText = textContent.Text.Text
	}

	t.client.sendUpdate(MessageEditedUpdate{
		ChatID:    u.ChatId,
		MessageID: u.MessageId,
		NewText:   newText,
		EditDate:  0,
	})
}

func (t *tdlibImpl) handleDeleteMessages(u *client.UpdateDeleteMessages) {
	if !u.IsPermanent {
		return
	}
	t.client.sendUpdate(MessageDeletedUpdate{
		ChatID:     u.ChatId,
		MessageIDs: u.MessageIds,
	})
}

func (t *tdlibImpl) handleNewChat(u *client.UpdateNewChat) {
	chat := t.convertChat(u.Chat)
	t.client.cacheChat(chat)
	t.client.sendUpdate(ChatUpdate{Chat: chat})
}

func (t *tdlibImpl) handleChatTitle(u *client.UpdateChatTitle) {
	t.client.mu.Lock()
	if chat, ok := t.client.chats[u.ChatId]; ok {
		chat.Title = u.Title
	}
	t.client.mu.Unlock()
}

func (t *tdlibImpl) handleChatRead(u *client.UpdateChatReadInbox) {
	t.client.mu.Lock()
	if chat, ok := t.client.chats[u.ChatId]; ok {
		chat.UnreadCount = int(u.UnreadCount)
	}
	t.client.mu.Unlock()

	t.client.sendUpdate(ChatReadUpdate{
		ChatID:        u.ChatId,
		LastReadInbox: u.LastReadInboxMessageId,
	})
}

func (t *tdlibImpl) handleChatLastMessage(u *client.UpdateChatLastMessage) {
	if u.LastMessage == nil {
		return
	}

	msg := t.convertMessage(u.LastMessage)
	t.client.mu.Lock()
	if chat, ok := t.client.chats[u.ChatId]; ok {
		chat.LastMessage = msg
		chat.LastMessageAt = msg.Date
	}
	t.client.mu.Unlock()
}

func (t *tdlibImpl) handleUserUpdate(u *client.UpdateUser) {
	user := t.convertUser(u.User)
	t.client.cacheUser(user)
}

func (t *tdlibImpl) handleTyping(u *client.UpdateChatAction) {
	var action TypingAction
	switch u.Action.ChatActionType() {
	case client.TypeChatActionTyping:
		action = TypingActionTyping
	case client.TypeChatActionRecordingVideo:
		action = TypingActionRecordingVideo
	case client.TypeChatActionRecordingVoiceNote:
		action = TypingActionRecordingVoice
	case client.TypeChatActionUploadingDocument:
		action = TypingActionUploadingDocument
	case client.TypeChatActionUploadingPhoto:
		action = TypingActionUploadingPhoto
	case client.TypeChatActionUploadingVideo:
		action = TypingActionUploadingVideo
	case client.TypeChatActionCancel:
		action = TypingActionCancel
	default:
		return
	}

	var userName string
	if sender, ok := u.SenderId.(*client.MessageSenderUser); ok {
		t.client.mu.RLock()
		if user, exists := t.client.users[sender.UserId]; exists {
			userName = user.DisplayName()
		}
		t.client.mu.RUnlock()
	}

	t.client.sendUpdate(TypingUpdate{
		ChatID:   u.ChatId,
		UserID:   0,
		UserName: userName,
		Action:   action,
	})
}

func (t *tdlibImpl) handleFileUpdate(u *client.UpdateFile) {
	file := u.File
	if file.Local.IsDownloadingCompleted {
		t.client.sendUpdate(DownloadCompleteUpdate{
			FileID:    fmt.Sprintf("%d", file.Id),
			LocalPath: file.Local.Path,
		})
	} else if file.Local.IsDownloadingActive {
		t.client.sendUpdate(DownloadProgressUpdate{
			FileID:          fmt.Sprintf("%d", file.Id),
			DownloadedBytes: int64(file.Local.DownloadedSize),
			TotalBytes:      int64(file.ExpectedSize),
		})
	}
}

func (t *tdlibImpl) loadInitialData() {
	// Get current user
	me, err := t.tdClient.GetMe()
	if err == nil {
		t.client.setMe(t.convertUser(me))
	}

	// Load chat list
	_, _ = t.tdClient.LoadChats(&client.LoadChatsRequest{
		ChatList: &client.ChatListMain{},
		Limit:    100,
	})
}

func (t *tdlibImpl) sendPhoneNumber(phone string) error {
	_, err := t.tdClient.SetAuthenticationPhoneNumber(&client.SetAuthenticationPhoneNumberRequest{
		PhoneNumber: phone,
	})
	return err
}

func (t *tdlibImpl) sendAuthCode(code string) error {
	_, err := t.tdClient.CheckAuthenticationCode(&client.CheckAuthenticationCodeRequest{
		Code: code,
	})
	return err
}

func (t *tdlibImpl) send2FAPassword(password string) error {
	_, err := t.tdClient.CheckAuthenticationPassword(&client.CheckAuthenticationPasswordRequest{
		Password: password,
	})
	return err
}

func (t *tdlibImpl) getChats(limit int) ([]*Chat, error) {
	chatIDs, err := t.tdClient.GetChats(&client.GetChatsRequest{
		ChatList: &client.ChatListMain{},
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, err
	}

	chats := make([]*Chat, 0, len(chatIDs.ChatIds))
	for _, chatID := range chatIDs.ChatIds {
		tdChat, err := t.tdClient.GetChat(&client.GetChatRequest{
			ChatId: chatID,
		})
		if err != nil {
			continue
		}
		chat := t.convertChat(tdChat)
		t.client.cacheChat(chat)
		chats = append(chats, chat)
	}

	return chats, nil
}

func (t *tdlibImpl) getChat(chatID int64) (*Chat, error) {
	tdChat, err := t.tdClient.GetChat(&client.GetChatRequest{
		ChatId: chatID,
	})
	if err != nil {
		return nil, err
	}

	chat := t.convertChat(tdChat)
	t.client.cacheChat(chat)
	return chat, nil
}

func (t *tdlibImpl) searchPublicChat(username string) (*Chat, error) {
	tdChat, err := t.tdClient.SearchPublicChat(&client.SearchPublicChatRequest{
		Username: username,
	})
	if err != nil {
		return nil, err
	}

	chat := t.convertChat(tdChat)
	t.client.cacheChat(chat)
	return chat, nil
}

func (t *tdlibImpl) getChatHistory(chatID int64, fromMessageID int64, limit int) ([]*Message, error) {
	history, err := t.tdClient.GetChatHistory(&client.GetChatHistoryRequest{
		ChatId:        chatID,
		FromMessageId: fromMessageID,
		Offset:        0,
		Limit:         int32(limit),
		OnlyLocal:     false,
	})
	if err != nil {
		return nil, err
	}

	messages := make([]*Message, 0, len(history.Messages))
	for i := len(history.Messages) - 1; i >= 0; i-- {
		msg := t.convertMessage(history.Messages[i])
		if msg != nil {
			messages = append(messages, msg)
		}
	}

	return messages, nil
}

func (t *tdlibImpl) sendMessage(chatID int64, text string, replyToID int64) (*Message, error) {
	var replyTo client.InputMessageReplyTo
	if replyToID > 0 {
		replyTo = &client.InputMessageReplyToMessage{
			MessageId: replyToID,
		}
	}

	sent, err := t.tdClient.SendMessage(&client.SendMessageRequest{
		ChatId:  chatID,
		ReplyTo: replyTo,
		InputMessageContent: &client.InputMessageText{
			Text: &client.FormattedText{
				Text: text,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return t.convertMessage(sent), nil
}

func (t *tdlibImpl) markAsRead(chatID int64, messageIDs []int64) error {
	if len(messageIDs) == 0 {
		return nil
	}

	_, err := t.tdClient.ViewMessages(&client.ViewMessagesRequest{
		ChatId:     chatID,
		MessageIds: messageIDs,
		ForceRead:  true,
	})
	return err
}

func (t *tdlibImpl) downloadFile(fileID string, priority int) error {
	var id int32
	_, err := fmt.Sscanf(fileID, "%d", &id)
	if err != nil {
		return fmt.Errorf("invalid file ID: %w", err)
	}

	_, err = t.tdClient.DownloadFile(&client.DownloadFileRequest{
		FileId:      id,
		Priority:    int32(priority),
		Offset:      0,
		Limit:       0,
		Synchronous: false,
	})
	return err
}

func (t *tdlibImpl) cancelDownload(fileID string) error {
	var id int32
	_, err := fmt.Sscanf(fileID, "%d", &id)
	if err != nil {
		return fmt.Errorf("invalid file ID: %w", err)
	}

	_, err = t.tdClient.CancelDownloadFile(&client.CancelDownloadFileRequest{
		FileId:        id,
		OnlyIfPending: false,
	})
	return err
}

// Conversion methods

func (t *tdlibImpl) convertChat(tdChat *client.Chat) *Chat {
	chat := &Chat{
		ID:          tdChat.Id,
		Title:       tdChat.Title,
		UnreadCount: int(tdChat.UnreadCount),
	}

	switch tdChat.Type.ChatTypeType() {
	case client.TypeChatTypePrivate:
		chat.Type = ChatTypePrivate
		if private, ok := tdChat.Type.(*client.ChatTypePrivate); ok {
			t.client.mu.RLock()
			if user, exists := t.client.users[private.UserId]; exists {
				chat.Username = user.Username
			}
			t.client.mu.RUnlock()
		}
	case client.TypeChatTypeBasicGroup:
		chat.Type = ChatTypeGroup
	case client.TypeChatTypeSupergroup:
		if sg, ok := tdChat.Type.(*client.ChatTypeSupergroup); ok {
			if sg.IsChannel {
				chat.Type = ChatTypeChannel
			} else {
				chat.Type = ChatTypeSupergroup
			}
		}
	case client.TypeChatTypeSecret:
		chat.Type = ChatTypeSecret
	}

	if tdChat.LastMessage != nil {
		chat.LastMessage = t.convertMessage(tdChat.LastMessage)
		chat.LastMessageAt = time.Unix(int64(tdChat.LastMessage.Date), 0)
	}

	return chat
}

func (t *tdlibImpl) convertMessage(tdMsg *client.Message) *Message {
	if tdMsg == nil {
		return nil
	}

	msg := &Message{
		ID:         tdMsg.Id,
		ChatID:     tdMsg.ChatId,
		Date:       time.Unix(int64(tdMsg.Date), 0),
		IsOutgoing: tdMsg.IsOutgoing,
		IsEdited:   tdMsg.EditDate > 0,
	}

	if sender, ok := tdMsg.SenderId.(*client.MessageSenderUser); ok {
		msg.SenderID = sender.UserId
		t.client.mu.RLock()
		if user, exists := t.client.users[sender.UserId]; exists {
			msg.SenderName = user.FullName()
		}
		t.client.mu.RUnlock()
		if msg.SenderName == "" {
			msg.SenderName = fmt.Sprintf("User %d", sender.UserId)
		}
	} else if sender, ok := tdMsg.SenderId.(*client.MessageSenderChat); ok {
		msg.SenderID = sender.ChatId
		t.client.mu.RLock()
		if chat, exists := t.client.chats[sender.ChatId]; exists {
			msg.SenderName = chat.Title
		}
		t.client.mu.RUnlock()
	}

	switch content := tdMsg.Content.(type) {
	case *client.MessageText:
		msg.Text = content.Text.Text
		msg.FormattedText = t.convertFormattedText(content.Text)

	case *client.MessagePhoto:
		msg.Text = "[Photo]"
		if content.Caption != nil && content.Caption.Text != "" {
			msg.Text += " " + content.Caption.Text
		}
		msg.Media = &MediaInfo{Type: MediaTypePhoto, Caption: content.Caption.Text}

	case *client.MessageVideo:
		msg.Text = "[Video]"
		if content.Caption != nil && content.Caption.Text != "" {
			msg.Text += " " + content.Caption.Text
		}
		msg.Media = &MediaInfo{
			Type:     MediaTypeVideo,
			FileName: content.Video.FileName,
			FileSize: int64(content.Video.Video.Size),
			MimeType: content.Video.MimeType,
			Caption:  content.Caption.Text,
		}

	case *client.MessageDocument:
		msg.Text = fmt.Sprintf("[Document: %s]", content.Document.FileName)
		if content.Caption != nil && content.Caption.Text != "" {
			msg.Text += " " + content.Caption.Text
		}
		msg.Media = &MediaInfo{
			Type:     MediaTypeDocument,
			FileName: content.Document.FileName,
			FileSize: int64(content.Document.Document.Size),
			MimeType: content.Document.MimeType,
			Caption:  content.Caption.Text,
		}

	case *client.MessageSticker:
		msg.Text = fmt.Sprintf("[Sticker: %s]", content.Sticker.Emoji)
		msg.Media = &MediaInfo{Type: MediaTypeSticker}

	case *client.MessageAnimation:
		msg.Text = "[GIF]"
		if content.Caption != nil && content.Caption.Text != "" {
			msg.Text += " " + content.Caption.Text
		}
		msg.Media = &MediaInfo{Type: MediaTypeAnimation, Caption: content.Caption.Text}

	case *client.MessageVoiceNote:
		msg.Text = "[Voice Message]"
		if content.Caption != nil && content.Caption.Text != "" {
			msg.Text += " " + content.Caption.Text
		}
		msg.Media = &MediaInfo{Type: MediaTypeVoice, Caption: content.Caption.Text}

	case *client.MessageAudio:
		title := content.Audio.Title
		if title == "" {
			title = content.Audio.FileName
		}
		msg.Text = fmt.Sprintf("[Audio: %s]", title)
		msg.Media = &MediaInfo{Type: MediaTypeAudio, FileName: content.Audio.FileName}

	case *client.MessageVideoNote:
		msg.Text = "[Video Message]"
		msg.Media = &MediaInfo{Type: MediaTypeVideoNote}

	default:
		msg.Text = "[Unsupported message type]"
	}

	if tdMsg.ReplyTo != nil {
		if reply, ok := tdMsg.ReplyTo.(*client.MessageReplyToMessage); ok {
			msg.ReplyToID = reply.MessageId
		}
	}

	return msg
}

func (t *tdlibImpl) convertFormattedText(ft *client.FormattedText) *FormattedText {
	if ft == nil {
		return nil
	}

	formatted := &FormattedText{
		Text:     ft.Text,
		Entities: make([]TextEntity, 0, len(ft.Entities)),
	}

	for _, entity := range ft.Entities {
		te := TextEntity{
			Offset: int(entity.Offset),
			Length: int(entity.Length),
		}

		switch e := entity.Type.(type) {
		case *client.TextEntityTypeBold:
			te.Type = EntityTypeBold
		case *client.TextEntityTypeItalic:
			te.Type = EntityTypeItalic
		case *client.TextEntityTypeUnderline:
			te.Type = EntityTypeUnderline
		case *client.TextEntityTypeStrikethrough:
			te.Type = EntityTypeStrikethrough
		case *client.TextEntityTypeCode:
			te.Type = EntityTypeCode
		case *client.TextEntityTypePre:
			te.Type = EntityTypePre
		case *client.TextEntityTypeTextUrl:
			te.Type = EntityTypeTextURL
			te.URL = e.Url
		case *client.TextEntityTypeMention:
			te.Type = EntityTypeMention
		case *client.TextEntityTypeHashtag:
			te.Type = EntityTypeHashtag
		case *client.TextEntityTypeCashtag:
			te.Type = EntityTypeCashtag
		case *client.TextEntityTypeBotCommand:
			te.Type = EntityTypeBotCommand
		case *client.TextEntityTypeUrl:
			te.Type = EntityTypeURL
		case *client.TextEntityTypeEmailAddress:
			te.Type = EntityTypeEmailAddress
		case *client.TextEntityTypePhoneNumber:
			te.Type = EntityTypePhoneNumber
		default:
			continue
		}

		formatted.Entities = append(formatted.Entities, te)
	}

	return formatted
}

func (t *tdlibImpl) convertUser(tdUser *client.User) *User {
	username := ""
	if tdUser.Usernames != nil && len(tdUser.Usernames.ActiveUsernames) > 0 {
		username = tdUser.Usernames.ActiveUsernames[0]
	}

	return &User{
		ID:        tdUser.Id,
		FirstName: tdUser.FirstName,
		LastName:  tdUser.LastName,
		Username:  username,
		Phone:     tdUser.PhoneNumber,
	}
}
