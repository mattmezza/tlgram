package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.General.SendKey != "ctrl-enter" {
		t.Errorf("expected send_key 'ctrl-enter', got %q", cfg.General.SendKey)
	}

	if cfg.Appearance.AuthorDisplay != "fullname" {
		t.Errorf("expected author_display 'fullname', got %q", cfg.Appearance.AuthorDisplay)
	}

	if cfg.Appearance.ReplyPreviewLength != 30 {
		t.Errorf("expected reply_preview_length 30, got %d", cfg.Appearance.ReplyPreviewLength)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "valid default config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "valid send_key enter",
			modify: func(c *Config) {
				c.General.SendKey = "enter"
			},
			wantErr: false,
		},
		{
			name: "invalid send_key",
			modify: func(c *Config) {
				c.General.SendKey = "invalid"
			},
			wantErr: true,
		},
		{
			name: "empty alias name",
			modify: func(c *Config) {
				c.ChatAliases[""] = "@user"
			},
			wantErr: true,
		},
		{
			name: "empty alias target",
			modify: func(c *Config) {
				c.ChatAliases["work"] = ""
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modify(cfg)
			err := cfg.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_ResolveAlias(t *testing.T) {
	cfg := Default()
	cfg.ChatAliases["work"] = "@john_doe"
	cfg.ChatAliases["Project"] = "-1001234567890"

	tests := []struct {
		input    string
		expected string
	}{
		{"work", "@john_doe"},
		{"WORK", "@john_doe"}, // Case insensitive
		{"Project", "-1001234567890"},
		{"project", "-1001234567890"}, // Case insensitive
		{"@jane_doe", "@jane_doe"},    // Not an alias, return as-is
		{"unknown", "unknown"},        // Not an alias, return as-is
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := cfg.ResolveAlias(tt.input)
			if result != tt.expected {
				t.Errorf("ResolveAlias(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	homeDir, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/Downloads", filepath.Join(homeDir, "Downloads")},
		{"~/", homeDir},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := expandPath(tt.input)
			if result != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLoad_CreatesDefaultConfig(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "tlgram-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override home directory for test
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check that config was loaded with defaults
	if cfg.General.SendKey != "ctrl-enter" {
		t.Errorf("expected default send_key, got %q", cfg.General.SendKey)
	}

	// Check that config file was created
	configPath := filepath.Join(tmpDir, ".config", "tlgram", "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}
}
