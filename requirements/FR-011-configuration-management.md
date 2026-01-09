# FR-011: Configuration Management

## Description
The application must support configuration via a TOML file to customize behavior, appearance, and keybindings.

## Priority
**MEDIUM** - Important for customization and hop template integration

## User Story
As a user, I want to configure the application through a simple TOML file, so that I can customize behavior, set up chat aliases, and integrate it into my workflow.

## Acceptance Criteria

### AC-011.1: Configuration File Location
- Configuration SHALL be stored at `~/.config/tlgram/config.toml`
- The application SHALL create the config directory if it doesn't exist on first run
- IF no config file exists, the application SHALL:
  - Create a default config file with commented examples
  - Use sensible defaults for all settings

### AC-011.2: Configuration File Format
- Configuration SHALL use TOML format
- The file SHALL be human-readable and well-commented
- The application SHALL validate the config and report parsing errors clearly

### AC-011.3: General Settings Section
```toml
[general]
# Send keybinding: "enter" or "ctrl-enter"
send_key = "ctrl-enter"  # default: ctrl-enter

# Download directory for media files
download_dir = "~/Downloads/tlgram"  # default: ~/Downloads/tlgram

# Auto-mark messages as read when viewing chat
auto_mark_read = true  # default: true

# Number of messages to load initially
initial_message_count = 50  # default: 50
```

### AC-011.4: Appearance Settings Section
```toml
[appearance]
# Show timestamps on all messages (vs. only on focused message)
always_show_timestamps = false  # default: false

# Timestamp format (Go time.Time format)
timestamp_format = "15:04:05"  # default: HH:MM:SS

# Show sender avatars (if terminal supports it - future)
show_avatars = false  # default: false

# Color theme (future: support multiple themes)
theme = "default"
```

### AC-011.5: Chat Aliases Section (See FR-012)
```toml
[chat_aliases]
work = "@john_doe"
project = "-1001234567890"
team = "@project_team_group"
```

### AC-011.6: Keybindings Section
```toml
[keybindings]
# Chat switcher
chat_switcher = "Ctrl-p"

# Search in chat
search = "/"

# Reply to message
reply = "r"

# Copy message
copy = "y"

# Download media
download = "d"

# Enter insert mode
insert_mode = "i"

# Quit application
quit = ":q"
```

### AC-011.7: Network Settings Section
```toml
[network]
# Reconnection settings
auto_reconnect = true  # default: true
reconnect_delay = 2  # seconds, default: 2

# Proxy settings (optional, for future)
# proxy_type = "socks5"
# proxy_host = "localhost"
# proxy_port = 9050
```

### AC-011.8: Logging Settings Section
```toml
[logging]
# Log level: "debug", "info", "warn", "error"
level = "info"  # default: info

# Log file location
log_file = "~/.config/tlgram/logs/app.log"

# Max log file size (MB) before rotation
max_size = 10

# Number of old log files to keep
max_backups = 3
```

### AC-011.9: Configuration Reload
- Changes to config SHALL require application restart
- No live reload needed for v1 (complexity vs. value)

### AC-011.10: Configuration Validation
- On startup, the application SHALL:
  - Parse the config file
  - Validate all settings
  - Report errors with line numbers if config is invalid
  - Fall back to defaults for invalid individual settings
  - Exit if config is completely unparseable

### AC-011.11: Configuration Errors
- IF config parsing fails, the application SHALL:
  - Print error message to stderr
  - Show which setting is invalid
  - Suggest correct format
  - Exit with code 1

## Default Config File Template
The application SHALL generate this on first run:
```toml
# tlgram configuration file
# See https://github.com/username/tlgram for full documentation

[general]
send_key = "ctrl-enter"  # "enter" or "ctrl-enter"
download_dir = "~/Downloads/tlgram"
auto_mark_read = true
initial_message_count = 50

[appearance]
always_show_timestamps = false
timestamp_format = "15:04:05"
theme = "default"

[chat_aliases]
# Define shortcuts for your frequent chats
# Example:
# work = "@john_doe"
# project = "-1001234567890"

[keybindings]
chat_switcher = "Ctrl-p"
search = "/"
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
```

## Technical Notes
- Use Go's standard TOML parsing library (e.g., `github.com/BurntSushi/toml`)
- Expand `~` to home directory in file paths
- Validate download_dir exists or can be created
- Use Go's `time.Time` format strings for timestamps

## Dependencies
- None

## Related Requirements
- FR-012: Chat Aliases (uses config [chat_aliases] section)
- FR-004: Send Text Messages (uses send_key setting)
- FR-007: Media File Handling (uses download_dir)
- NFR-002: Reliability (proper error handling)
