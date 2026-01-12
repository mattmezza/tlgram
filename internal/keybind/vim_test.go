package keybind

import (
	"testing"
	"time"
)

func TestNewVimState(t *testing.T) {
	v := NewVimState()
	if v.Mode() != ModeNormal {
		t.Errorf("expected ModeNormal, got %v", v.Mode())
	}
}

func TestVimState_BasicNavigation(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected Action
		count    int
	}{
		{"move down j", "j", ActionMoveDown, 1},
		{"move down arrow", "down", ActionMoveDown, 1},
		{"move up k", "k", ActionMoveUp, 1},
		{"move up arrow", "up", ActionMoveUp, 1},
		{"move left h", "h", ActionMoveLeft, 1},
		{"move right l", "l", ActionMoveRight, 1},
		{"jump bottom G", "G", ActionJumpBottom, 0},
		{"half page down", "ctrl+d", ActionHalfPageDown, 1},
		{"half page up", "ctrl+u", ActionHalfPageUp, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewVimState()
			result := v.ProcessKey(tt.key)
			if result.Action != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result.Action)
			}
			if tt.count > 0 && result.Count != tt.count {
				t.Errorf("expected count %d, got %d", tt.count, result.Count)
			}
		})
	}
}

func TestVimState_GGSequence(t *testing.T) {
	v := NewVimState()

	// First 'g' should return pending
	result := v.ProcessKey("g")
	if result.Action != ActionPending {
		t.Errorf("first g: expected ActionPending, got %v", result.Action)
	}

	// Second 'g' should return jump to top
	result = v.ProcessKey("g")
	if result.Action != ActionJumpTop {
		t.Errorf("second g: expected ActionJumpTop, got %v", result.Action)
	}
}

func TestVimState_KeyTimeout(t *testing.T) {
	v := NewVimState()

	// First 'g'
	v.ProcessKey("g")

	// Simulate timeout
	v.lastKeyAt = time.Now().Add(-600 * time.Millisecond)

	// Next 'g' should start fresh (pending)
	result := v.ProcessKey("g")
	if result.Action != ActionPending {
		t.Errorf("expected ActionPending after timeout, got %v", result.Action)
	}
}

func TestVimState_CountPrefix(t *testing.T) {
	v := NewVimState()

	v.ProcessKey("5")
	result := v.ProcessKey("j")

	if result.Action != ActionMoveDown {
		t.Errorf("expected ActionMoveDown, got %v", result.Action)
	}
	if result.Count != 5 {
		t.Errorf("expected count 5, got %d", result.Count)
	}
}

func TestVimState_MultiDigitCount(t *testing.T) {
	v := NewVimState()

	v.ProcessKey("1")
	v.ProcessKey("5")
	result := v.ProcessKey("k")

	if result.Action != ActionMoveUp {
		t.Errorf("expected ActionMoveUp, got %v", result.Action)
	}
	if result.Count != 15 {
		t.Errorf("expected count 15, got %d", result.Count)
	}
}

func TestVimState_ModeTransitions(t *testing.T) {
	v := NewVimState()

	// Enter insert mode
	result := v.ProcessKey("i")
	if result.Action != ActionEnterInsert {
		t.Errorf("expected ActionEnterInsert, got %v", result.Action)
	}
	if v.Mode() != ModeInsert {
		t.Errorf("expected ModeInsert, got %v", v.Mode())
	}

	// Exit with escape
	result = v.ProcessKey("esc")
	if result.Action != ActionExitToNormal {
		t.Errorf("expected ActionExitToNormal, got %v", result.Action)
	}
	if v.Mode() != ModeNormal {
		t.Errorf("expected ModeNormal, got %v", v.Mode())
	}
}

// Note: Search and Command modes are deferred (not implemented in current version)
// See FR-009 and FR-010 requirements

func TestVimState_SetMode(t *testing.T) {
	v := NewVimState()

	v.SetMode(ModeInsert)
	if v.Mode() != ModeInsert {
		t.Errorf("expected ModeInsert, got %v", v.Mode())
	}

	// Buffer should be cleared
	v.keyBuffer = []string{"g"}
	v.count = 5
	v.SetMode(ModeNormal)

	if len(v.keyBuffer) != 0 {
		t.Errorf("expected empty buffer, got %v", v.keyBuffer)
	}
	if v.count != 0 {
		t.Errorf("expected count 0, got %d", v.count)
	}
}

func TestVimState_Actions(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected Action
	}{
		{"reply", "r", ActionReply},
		{"copy", "yy", ActionCopy},
		{"download", "d", ActionDownload},
		{"quit", "q", ActionQuit},
		{"switcher", "ctrl+p", ActionOpenSwitcher},
		{"select", "enter", ActionSelect},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewVimState()
			result := v.ProcessKey(tt.key)
			if result.Action != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result.Action)
			}
		})
	}
}

func TestModeString(t *testing.T) {
	tests := []struct {
		mode     Mode
		expected string
	}{
		{ModeNormal, "NORMAL"},
		{ModeInsert, "INSERT"},
		{Mode(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.mode.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.mode.String())
			}
		})
	}
}
