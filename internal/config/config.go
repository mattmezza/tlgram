package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Environment variable names for API credentials
const (
	EnvAPIID   = "TLGRAM_API_ID"
	EnvAPIHash = "TLGRAM_API_HASH"
)

// Config represents the application configuration
type Config struct {
	ConfigDir string `toml:"-"` // Not in TOML, set programmatically

	General     GeneralConfig     `toml:"general"`
	Appearance  AppearanceConfig  `toml:"appearance"`
	ChatAliases map[string]string `toml:"chat_aliases"`
	Keybindings KeybindingsConfig `toml:"keybindings"`
	Network     NetworkConfig     `toml:"network"`
	Logging     LoggingConfig     `toml:"logging"`
	Telegram    TelegramConfig    `toml:"telegram"`
}

// GeneralConfig contains general settings
type GeneralConfig struct {
	SendKey             string `toml:"send_key"`
	DownloadDir         string `toml:"download_dir"`
	AutoMarkRead        bool   `toml:"auto_mark_read"`
	InitialMessageCount int    `toml:"initial_message_count"`
}

// AppearanceConfig contains appearance settings
type AppearanceConfig struct {
	AlwaysShowTimestamps bool   `toml:"always_show_timestamps"`
	TimestampFormat      string `toml:"timestamp_format"`
	Theme                string `toml:"theme"`
	AuthorDisplay        string `toml:"author_display"`       // "fullname" or "username"
	ReplyPreviewLength   int    `toml:"reply_preview_length"` // max chars for reply preview
	ShowChatID           bool   `toml:"show_chat_id"`         // show chat ID in header
}

// KeybindingsConfig contains keybinding settings
type KeybindingsConfig struct {
	ChatSwitcher string `toml:"chat_switcher"`
	Reply        string `toml:"reply"`
	Copy         string `toml:"copy"`
	Download     string `toml:"download"`
	InsertMode   string `toml:"insert_mode"`
}

// NetworkConfig contains network settings
type NetworkConfig struct {
	AutoReconnect  bool `toml:"auto_reconnect"`
	ReconnectDelay int  `toml:"reconnect_delay"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level      string `toml:"level"`
	LogFile    string `toml:"log_file"`
	MaxSize    int    `toml:"max_size"`
	MaxBackups int    `toml:"max_backups"`
}

// TelegramConfig contains Telegram API settings
type TelegramConfig struct {
	APIID   int    `toml:"api_id"`
	APIHash string `toml:"api_hash"`
}

// Default returns the default configuration
func Default() *Config {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "tlgram")

	return &Config{
		ConfigDir: configDir,
		General: GeneralConfig{
			SendKey:             "ctrl-enter",
			DownloadDir:         filepath.Join(homeDir, "Downloads", "tlgram"),
			AutoMarkRead:        true,
			InitialMessageCount: 50,
		},
		Appearance: AppearanceConfig{
			AlwaysShowTimestamps: false,
			TimestampFormat:      "15:04:05",
			Theme:                "default",
			AuthorDisplay:        "fullname",
			ReplyPreviewLength:   30,
			ShowChatID:           false,
		},
		ChatAliases: make(map[string]string),
		Keybindings: KeybindingsConfig{
			ChatSwitcher: "ctrl+p",
			Reply:        "r",
			Copy:         "y",
			Download:     "d",
			InsertMode:   "i",
		},
		Network: NetworkConfig{
			AutoReconnect:  true,
			ReconnectDelay: 2,
		},
		Logging: LoggingConfig{
			Level:      "info",
			LogFile:    filepath.Join(configDir, "logs", "app.log"),
			MaxSize:    10,
			MaxBackups: 3,
		},
		Telegram: TelegramConfig{
			// Users must provide their own API credentials
			// Get them from https://my.telegram.org/apps
			APIID:   0,
			APIHash: "",
		},
	}
}

// Load loads configuration from file, falling back to defaults
func Load() (*Config, error) {
	cfg := Default()

	configPath := filepath.Join(cfg.ConfigDir, "config.toml")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(cfg.ConfigDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file
		if err := createDefaultConfig(configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return cfg, nil
	}

	// Load config from file
	if _, err := toml.DecodeFile(configPath, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Expand paths
	cfg.General.DownloadDir = expandPath(cfg.General.DownloadDir)
	cfg.Logging.LogFile = expandPath(cfg.Logging.LogFile)

	// Load API credentials from environment variables (overrides config file)
	cfg.loadEnvCredentials()

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadEnvCredentials loads API credentials from environment variables
// Environment variables take precedence over config file values
func (c *Config) loadEnvCredentials() {
	if apiID := os.Getenv(EnvAPIID); apiID != "" {
		if id, err := strconv.Atoi(apiID); err == nil {
			c.Telegram.APIID = id
		}
	}
	if apiHash := os.Getenv(EnvAPIHash); apiHash != "" {
		c.Telegram.APIHash = apiHash
	}
}

// validate checks if the configuration is valid
func (c *Config) validate() error {
	// Validate send_key
	if c.General.SendKey != "enter" && c.General.SendKey != "ctrl-enter" {
		return fmt.Errorf("invalid send_key: %q (must be 'enter' or 'ctrl-enter')", c.General.SendKey)
	}

	// Validate log level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %q (must be debug, info, warn, or error)", c.Logging.Level)
	}

	// Validate author_display
	if c.Appearance.AuthorDisplay != "" && c.Appearance.AuthorDisplay != "fullname" && c.Appearance.AuthorDisplay != "username" {
		return fmt.Errorf("invalid author_display: %q (must be 'fullname' or 'username')", c.Appearance.AuthorDisplay)
	}

	// Validate chat aliases
	for alias, target := range c.ChatAliases {
		if alias == "" || target == "" {
			return fmt.Errorf("invalid chat alias: empty alias or target")
		}
	}

	return nil
}

// ResolveAlias resolves a chat alias to its target, or returns the input if not an alias
func (c *Config) ResolveAlias(input string) string {
	// Check for exact alias match (case-insensitive)
	for alias, target := range c.ChatAliases {
		if strings.EqualFold(alias, input) {
			return target
		}
	}
	return input
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// createDefaultConfig creates a default configuration file
func createDefaultConfig(path string) error {
	content := `# tlgram configuration file
# See https://github.com/mattmezza/tlgram for full documentation

[telegram]
# Get your API credentials from https://my.telegram.org/apps
#
# You can set credentials here OR via environment variables:
#   export TLGRAM_API_ID=12345678
#   export TLGRAM_API_HASH="your_api_hash"
#
# Environment variables take precedence over config file values.
# This allows you to commit this config to your dotfiles without exposing secrets.
api_id = 0
api_hash = ""

[general]
# Send keybinding: "enter" or "ctrl-enter"
send_key = "ctrl-enter"

# Download directory for media files
download_dir = "~/Downloads/tlgram"

# Auto-mark messages as read when viewing chat
auto_mark_read = true

# Number of messages to load initially
initial_message_count = 50

[appearance]
# Show timestamps on all messages (vs. only on focused message)
always_show_timestamps = false

# Timestamp format (Go time.Time format)
timestamp_format = "15:04:05"

# Color theme
theme = "default"

# Author display format: "fullname" or "username"
# Toggle with 'u' key in chat view
author_display = "fullname"

# Maximum characters to show in reply preview
reply_preview_length = 30

# Show chat ID in header bar (useful for creating aliases)
# Toggle with 'I' key in chat view
show_chat_id = false

[chat_aliases]
# Define shortcuts for your frequent chats
# Example:
# work = "@john_doe"
# project = "-1001234567890"
# team = "@project_team_group"

[keybindings]
chat_switcher = "ctrl+p"
reply = "r"
copy = "y"
download = "d"
insert_mode = "i"

[network]
auto_reconnect = true
reconnect_delay = 2

[logging]
level = "info"
log_file = "~/.config/tlgram/logs/app.log"
max_size = 10
max_backups = 3
`
	return os.WriteFile(path, []byte(content), 0600)
}
