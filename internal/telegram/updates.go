package telegram

// Update represents an update from Telegram
type Update interface {
	isUpdate()
}

// AuthStateUpdate is sent when the authentication state changes
type AuthStateUpdate struct {
	State AuthState
}

func (AuthStateUpdate) isUpdate() {}

// ConnectionStateUpdate is sent when the connection state changes
type ConnectionStateUpdate struct {
	State ConnectionState
}

func (ConnectionStateUpdate) isUpdate() {}

// NewMessageUpdate is sent when a new message is received
type NewMessageUpdate struct {
	Message *Message
}

func (NewMessageUpdate) isUpdate() {}

// MessageEditedUpdate is sent when a message is edited
type MessageEditedUpdate struct {
	ChatID    int64
	MessageID int64
	NewText   string
	EditDate  int64
}

func (MessageEditedUpdate) isUpdate() {}

// MessageDeletedUpdate is sent when messages are deleted
type MessageDeletedUpdate struct {
	ChatID     int64
	MessageIDs []int64
}

func (MessageDeletedUpdate) isUpdate() {}

// ChatUpdate is sent when chat information changes
type ChatUpdate struct {
	Chat *Chat
}

func (ChatUpdate) isUpdate() {}

// ChatReadUpdate is sent when messages in a chat are marked as read
type ChatReadUpdate struct {
	ChatID        int64
	LastReadInbox int64
}

func (ChatReadUpdate) isUpdate() {}

// DialogUnreadUpdate is sent when a dialog's unread flag changes
type DialogUnreadUpdate struct {
	ChatID       int64
	MarkedUnread bool
}

func (DialogUnreadUpdate) isUpdate() {}

// TypingUpdate is sent when a user starts/stops typing
type TypingUpdate struct {
	ChatID   int64
	UserID   int64
	UserName string
	Action   TypingAction
}

func (TypingUpdate) isUpdate() {}

// TypingAction represents what the user is doing
type TypingAction int

const (
	TypingActionTyping TypingAction = iota
	TypingActionRecordingVideo
	TypingActionRecordingVoice
	TypingActionUploadingDocument
	TypingActionUploadingPhoto
	TypingActionUploadingVideo
	TypingActionCancel
)

// DownloadProgressUpdate is sent during file download
type DownloadProgressUpdate struct {
	FileID          string
	DownloadedBytes int64
	TotalBytes      int64
}

func (DownloadProgressUpdate) isUpdate() {}

// DownloadCompleteUpdate is sent when a file download completes
type DownloadCompleteUpdate struct {
	FileID    string
	LocalPath string
}

func (DownloadCompleteUpdate) isUpdate() {}

// ErrorUpdate is sent when an error occurs
type ErrorUpdate struct {
	Code    int
	Message string
}

func (ErrorUpdate) isUpdate() {}
