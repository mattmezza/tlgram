package keybind

import (
	"strings"
	"time"
	"unicode"
)

// Mode represents the current vim mode
type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeSearch
	ModeCommand
)

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	case ModeSearch:
		return "SEARCH"
	case ModeCommand:
		return "COMMAND"
	default:
		return "UNKNOWN"
	}
}

// Action represents the result of processing a key
type Action int

const (
	ActionNone Action = iota
	ActionPending      // Waiting for more input (e.g., first 'g' in 'gg')
	ActionUnknown      // Unknown key sequence

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
	ActionEnterSearch
	ActionEnterCommand
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

	// Search
	ActionSearchNext
	ActionSearchPrev
)

// Result holds the action and any associated count
type Result struct {
	Action Action
	Count  int
}

// VimState manages vim keybinding state
type VimState struct {
	mode      Mode
	keyBuffer []string
	lastKeyAt time.Time
	count     int
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
	v.count = 0
}

// ProcessKey processes a key and returns the resulting action
func (v *VimState) ProcessKey(key string) Result {
	now := time.Now()

	// Reset buffer if timeout expired
	if now.Sub(v.lastKeyAt) > keyTimeout {
		v.keyBuffer = v.keyBuffer[:0]
		v.count = 0
	}
	v.lastKeyAt = now

	// Handle based on current mode
	switch v.mode {
	case ModeNormal:
		return v.processNormalMode(key)
	case ModeInsert:
		return v.processInsertMode(key)
	case ModeSearch:
		return v.processSearchMode(key)
	case ModeCommand:
		return v.processCommandMode(key)
	}

	return Result{Action: ActionUnknown}
}

func (v *VimState) processNormalMode(key string) Result {
	// Handle digit for count accumulation
	if len(key) == 1 && unicode.IsDigit(rune(key[0])) && (v.count > 0 || key != "0") {
		v.count = v.count*10 + int(key[0]-'0')
		return Result{Action: ActionNone}
	}

	// Add key to buffer
	v.keyBuffer = append(v.keyBuffer, key)
	sequence := strings.Join(v.keyBuffer, "")

	// Get effective count (minimum 1)
	count := v.count
	if count == 0 {
		count = 1
	}

	// Check for complete sequences
	switch sequence {
	case "j", "down":
		v.reset()
		return Result{Action: ActionMoveDown, Count: count}
	case "k", "up":
		v.reset()
		return Result{Action: ActionMoveUp, Count: count}
	case "h", "left":
		v.reset()
		return Result{Action: ActionMoveLeft, Count: count}
	case "l", "right":
		v.reset()
		return Result{Action: ActionMoveRight, Count: count}
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
		return Result{Action: ActionHalfPageDown, Count: count}
	case "ctrl+u":
		v.reset()
		return Result{Action: ActionHalfPageUp, Count: count}
	case "ctrl+f":
		v.reset()
		return Result{Action: ActionPageDown, Count: count}
	case "ctrl+b":
		v.reset()
		return Result{Action: ActionPageUp, Count: count}
	case "i":
		v.reset()
		v.mode = ModeInsert
		return Result{Action: ActionEnterInsert}
	case "/":
		v.reset()
		v.mode = ModeSearch
		return Result{Action: ActionEnterSearch}
	case ":":
		v.reset()
		v.mode = ModeCommand
		return Result{Action: ActionEnterCommand}
	case "r":
		v.reset()
		return Result{Action: ActionReply}
	case "yy":
		v.reset()
		return Result{Action: ActionCopy}
	case "u":
		v.reset()
		return Result{Action: ActionToggleUsername}
	case "d":
		v.reset()
		return Result{Action: ActionDownload}
	case "ctrl+p":
		v.reset()
		return Result{Action: ActionOpenSwitcher}
	case "enter":
		v.reset()
		return Result{Action: ActionSelect}
	case "q":
		v.reset()
		return Result{Action: ActionQuit}
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

func (v *VimState) processSearchMode(key string) Result {
	switch key {
	case "esc", "escape":
		v.reset()
		v.mode = ModeNormal
		return Result{Action: ActionExitToNormal}
	case "enter":
		v.reset()
		return Result{Action: ActionSelect}
	case "n":
		return Result{Action: ActionSearchNext}
	case "N":
		return Result{Action: ActionSearchPrev}
	}
	return Result{Action: ActionNone}
}

func (v *VimState) processCommandMode(key string) Result {
	switch key {
	case "esc", "escape":
		v.reset()
		v.mode = ModeNormal
		return Result{Action: ActionExitToNormal}
	case "enter":
		v.reset()
		return Result{Action: ActionSelect}
	}
	return Result{Action: ActionNone}
}

func (v *VimState) reset() {
	v.keyBuffer = v.keyBuffer[:0]
	v.count = 0
}

func (v *VimState) isPrefixOfKnownSequence(prefix string) bool {
	knownSequences := []string{"gg", "yy"}
	for _, seq := range knownSequences {
		if strings.HasPrefix(seq, prefix) && seq != prefix {
			return true
		}
	}
	return false
}
