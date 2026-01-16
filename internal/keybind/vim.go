package keybind

import (
	"strings"
	"time"
)

// Mode represents the current vim mode
type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
)

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	default:
		return "UNKNOWN"
	}
}

// Action represents the result of processing a key
type Action int

const (
	ActionNone    Action = iota
	ActionPending        // Waiting for more input (e.g., first 'g' in 'gg')
	ActionUnknown        // Unknown key sequence

	// Navigation
	ActionMoveUp
	ActionMoveDown
	ActionMoveLeft
	ActionMoveRight
	ActionJumpTop
	ActionJumpBottom
	ActionHalfPageUp
	ActionHalfPageDown
	ActionPageUp
	ActionPageDown
	ActionViewportTop    // H - move to top of visible viewport
	ActionViewportBottom // L - move to bottom of visible viewport

	// Mode changes
	ActionEnterInsert
	ActionExitToNormal

	// Actions
	ActionReply
	ActionCopy
	ActionDownload
	ActionOpenSwitcher
	ActionSelect
	ActionCancel
	ActionQuit
	ActionToggleUsername
	ActionMarkRead         // R - mark messages as read up to cursor
	ActionMarkUnread       // U - mark dialog as unread
	ActionJumpToOriginal   // o - jump to original message (for replies)
	ActionJumpBack         // ctrl+o - jump back after jumping to original
	ActionShowChatID       // I - toggle chat ID display in header
	ActionAppend           // A - scroll to bottom and enter insert mode
	ActionEditMessage      // cc - edit message at cursor
	ActionDeleteMessage    // D - delete message at cursor
	ActionShowHelp         // ? - show help screen
	ActionToggleMute       // m - toggle mute for current chat
	ActionToggleGlobalMute // M - toggle global mute for all notifications
	ActionToggleWatch      // w - toggle watch for current chat (always notify)

	// Reactions
	ActionReact1      // 1 - first reaction emoji
	ActionReact2      // 2 - second reaction emoji
	ActionReact3      // 3 - third reaction emoji
	ActionReact4      // 4 - fourth reaction emoji
	ActionReact5      // 5 - fifth reaction emoji
	ActionReact6      // 6 - sixth reaction emoji
	ActionRemoveReact // 0 - remove own reaction
)

// Result holds the action
type Result struct {
	Action Action
}

// VimState manages vim keybinding state
type VimState struct {
	mode      Mode
	keyBuffer []string
	lastKeyAt time.Time
}

const keyTimeout = 500 * time.Millisecond

// NewVimState creates a new vim state machine
func NewVimState() *VimState {
	return &VimState{
		mode:      ModeNormal,
		keyBuffer: make([]string, 0, 4),
	}
}

// Mode returns the current mode
func (v *VimState) Mode() Mode {
	return v.mode
}

// SetMode sets the current mode
func (v *VimState) SetMode(mode Mode) {
	v.mode = mode
	v.keyBuffer = v.keyBuffer[:0]
}

// ProcessKey processes a key and returns the resulting action
func (v *VimState) ProcessKey(key string) Result {
	now := time.Now()

	// Reset buffer if timeout expired
	if now.Sub(v.lastKeyAt) > keyTimeout {
		v.keyBuffer = v.keyBuffer[:0]
	}
	v.lastKeyAt = now

	// Handle based on current mode
	switch v.mode {
	case ModeNormal:
		return v.processNormalMode(key)
	case ModeInsert:
		return v.processInsertMode(key)
	}

	return Result{Action: ActionUnknown}
}

func (v *VimState) processNormalMode(key string) Result {
	// Add key to buffer
	v.keyBuffer = append(v.keyBuffer, key)
	sequence := strings.Join(v.keyBuffer, "")

	// Check for complete sequences
	switch sequence {
	case "j", "down":
		v.reset()
		return Result{Action: ActionMoveDown}
	case "k", "up":
		v.reset()
		return Result{Action: ActionMoveUp}
	case "h", "left":
		v.reset()
		return Result{Action: ActionMoveLeft}
	case "l", "right":
		v.reset()
		return Result{Action: ActionMoveRight}
	case "gg":
		v.reset()
		return Result{Action: ActionJumpTop}
	case "g":
		// Waiting for second 'g'
		return Result{Action: ActionPending}
	case "G":
		v.reset()
		return Result{Action: ActionJumpBottom}
	case "H":
		v.reset()
		return Result{Action: ActionViewportTop}
	case "L":
		v.reset()
		return Result{Action: ActionViewportBottom}
	case "ctrl+d":
		v.reset()
		return Result{Action: ActionHalfPageDown}
	case "ctrl+u":
		v.reset()
		return Result{Action: ActionHalfPageUp}
	case "ctrl+f":
		v.reset()
		return Result{Action: ActionPageDown}
	case "ctrl+b":
		v.reset()
		return Result{Action: ActionPageUp}
	case "i":
		v.reset()
		v.mode = ModeInsert
		return Result{Action: ActionEnterInsert}
	case "r":
		v.reset()
		return Result{Action: ActionReply}
	case "yy":
		v.reset()
		return Result{Action: ActionCopy}
	case "cc":
		v.reset()
		return Result{Action: ActionEditMessage}
	case "D":
		v.reset()
		return Result{Action: ActionDeleteMessage}
	case "u":
		v.reset()
		return Result{Action: ActionToggleUsername}
	case "R":
		v.reset()
		return Result{Action: ActionMarkRead}
	case "U":
		v.reset()
		return Result{Action: ActionMarkUnread}
	case "d":
		v.reset()
		return Result{Action: ActionDownload}
	case "o":
		v.reset()
		return Result{Action: ActionJumpToOriginal}
	case "ctrl+o":
		v.reset()
		return Result{Action: ActionJumpBack}
	case "I":
		v.reset()
		return Result{Action: ActionShowChatID}
	case "A":
		v.reset()
		v.mode = ModeInsert
		return Result{Action: ActionAppend}
	case "ctrl+p":
		v.reset()
		return Result{Action: ActionOpenSwitcher}
	case "enter":
		v.reset()
		return Result{Action: ActionSelect}
	case "q":
		v.reset()
		return Result{Action: ActionQuit}
	case "?":
		v.reset()
		return Result{Action: ActionShowHelp}
	case "m":
		v.reset()
		return Result{Action: ActionToggleMute}
	case "M":
		v.reset()
		return Result{Action: ActionToggleGlobalMute}
	case "w":
		v.reset()
		return Result{Action: ActionToggleWatch}
	// Reactions
	case "1":
		v.reset()
		return Result{Action: ActionReact1}
	case "2":
		v.reset()
		return Result{Action: ActionReact2}
	case "3":
		v.reset()
		return Result{Action: ActionReact3}
	case "4":
		v.reset()
		return Result{Action: ActionReact4}
	case "5":
		v.reset()
		return Result{Action: ActionReact5}
	case "6":
		v.reset()
		return Result{Action: ActionReact6}
	case "0":
		v.reset()
		return Result{Action: ActionRemoveReact}
	}

	// Unknown sequence - reset if not a prefix
	if !v.isPrefixOfKnownSequence(sequence) {
		v.reset()
		return Result{Action: ActionUnknown}
	}

	return Result{Action: ActionPending}
}

func (v *VimState) processInsertMode(key string) Result {
	switch key {
	case "esc", "escape":
		v.reset()
		v.mode = ModeNormal
		return Result{Action: ActionExitToNormal}
	}
	// In insert mode, pass through to text input
	return Result{Action: ActionNone}
}

func (v *VimState) reset() {
	v.keyBuffer = v.keyBuffer[:0]
}

func (v *VimState) isPrefixOfKnownSequence(prefix string) bool {
	knownSequences := []string{"gg", "yy", "cc"}
	for _, seq := range knownSequences {
		if strings.HasPrefix(seq, prefix) && seq != prefix {
			return true
		}
	}
	return false
}
