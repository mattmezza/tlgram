package telegram

import "time"

// AuthState represents the authentication state
type AuthState int

const (
	AuthStateUnknown AuthState = iota
	AuthStateWaitPhoneNumber
	AuthStateWaitCode
	AuthStateWaitPassword
	AuthStateReady
	AuthStateLoggingOut
	AuthStateClosed
)

func (s AuthState) String() string {
	switch s {
	case AuthStateWaitPhoneNumber:
		return "WaitPhoneNumber"
	case AuthStateWaitCode:
		return "WaitCode"
	case AuthStateWaitPassword:
		return "WaitPassword"
	case AuthStateReady:
		return "Ready"
	case AuthStateLoggingOut:
		return "LoggingOut"
	case AuthStateClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}

// ConnectionState represents the connection state to Telegram
type ConnectionState int

const (
	ConnectionStateUnknown ConnectionState = iota
	ConnectionStateConnecting
	ConnectionStateConnectingToProxy
	ConnectionStateReady
	ConnectionStateUpdating
	ConnectionStateWaitingForNetwork
)

func (s ConnectionState) String() string {
	switch s {
	case ConnectionStateConnecting:
		return "Connecting"
	case ConnectionStateConnectingToProxy:
		return "ConnectingToProxy"
	case ConnectionStateReady:
		return "Connected"
	case ConnectionStateUpdating:
		return "Updating"
	case ConnectionStateWaitingForNetwork:
		return "WaitingForNetwork"
	default:
		return "Unknown"
	}
}

// Chat represents a Telegram chat
type Chat struct {
	ID            int64
	Type          ChatType
	Title         string
	Username      string
	UnreadCount   int
	LastMessage   *Message
	LastMessageAt time.Time
}

// ChatType represents the type of chat
type ChatType int

const (
	ChatTypePrivate ChatType = iota
	ChatTypeGroup
	ChatTypeSupergroup
	ChatTypeChannel
	ChatTypeSecret
)

func (t ChatType) String() string {
	switch t {
	case ChatTypePrivate:
		return "Private"
	case ChatTypeGroup:
		return "Group"
	case ChatTypeSupergroup:
		return "Supergroup"
	case ChatTypeChannel:
		return "Channel"
	case ChatTypeSecret:
		return "Secret"
	default:
		return "Unknown"
	}
}

// Message represents a Telegram message
type Message struct {
	ID            int64
	ChatID        int64
	SenderID      int64
	SenderName    string
	Text          string
	FormattedText *FormattedText
	Date          time.Time
	IsOutgoing    bool
	IsEdited      bool
	ReplyToID     int64
	ReplyTo       *Message
	Media         *MediaInfo
}

// FormattedText represents text with formatting entities
type FormattedText struct {
	Text     string
	Entities []TextEntity
}

// TextEntity represents a formatting entity in text
type TextEntity struct {
	Type   EntityType
	Offset int
	Length int
	URL    string // For EntityTypeTextURL
}

// EntityType represents the type of text formatting
type EntityType int

const (
	EntityTypeBold EntityType = iota
	EntityTypeItalic
	EntityTypeUnderline
	EntityTypeStrikethrough
	EntityTypeCode
	EntityTypePre
	EntityTypeTextURL
	EntityTypeMention
	EntityTypeHashtag
	EntityTypeCashtag
	EntityTypeBotCommand
	EntityTypeURL
	EntityTypeEmailAddress
	EntityTypePhoneNumber
)

// MediaInfo represents information about attached media
type MediaInfo struct {
	Type         MediaType
	FileName     string
	FileSize     int64
	MimeType     string
	Caption      string
	ThumbnailURL string
	FileID       string
	LocalPath    string
	IsDownloaded bool
}

// MediaType represents the type of media
type MediaType int

const (
	MediaTypePhoto MediaType = iota
	MediaTypeVideo
	MediaTypeAudio
	MediaTypeVoice
	MediaTypeVideoNote
	MediaTypeDocument
	MediaTypeSticker
	MediaTypeAnimation
)

func (t MediaType) String() string {
	switch t {
	case MediaTypePhoto:
		return "Photo"
	case MediaTypeVideo:
		return "Video"
	case MediaTypeAudio:
		return "Audio"
	case MediaTypeVoice:
		return "Voice"
	case MediaTypeVideoNote:
		return "VideoNote"
	case MediaTypeDocument:
		return "Document"
	case MediaTypeSticker:
		return "Sticker"
	case MediaTypeAnimation:
		return "Animation"
	default:
		return "Unknown"
	}
}

// User represents a Telegram user
type User struct {
	ID        int64
	FirstName string
	LastName  string
	Username  string
	Phone     string
}

// FullName returns the user's full name
func (u *User) FullName() string {
	if u.LastName == "" {
		return u.FirstName
	}
	return u.FirstName + " " + u.LastName
}

// DisplayName returns the best name to display for the user
func (u *User) DisplayName() string {
	if u.Username != "" {
		return "@" + u.Username
	}
	return u.FullName()
}
