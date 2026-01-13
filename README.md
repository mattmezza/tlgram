<p align="center">
  <img src=".github/assets/banner.png" alt="tlgram banner" />
</p>

# tlgram

A terminal-based Telegram client with vim keybindings, designed for tmux workflows.

## Features

- **Vim keybindings** - Navigate with hjkl, gg/G, Ctrl-d/u, H/L
- **Line-based cursor** - Move through messages line by line, handles wrapped text naturally
- **Multi-line messages** - Compose multi-line messages (Enter for newline, Alt-Enter to send)
- **CLI chat opening** - `tlgram --chat @username` or `tlgram --chat work`
- **Multiple instances** - Run different chats in different tmux panes
- **Fuzzy chat switcher** - Ctrl-p to search and switch chats (searches names and @usernames)
- **Chat aliases** - Define shortcuts like `work = "@john_doe"`
- **Smart header bar** - Shows chat type info (DM: name @username, Group: name (x members))
- **Unread indicator** - Visual badge for unread messages
- **Pure Go** - No CGO dependencies, uses gotd/td library

## Installation

### Quick Install (Linux/macOS)

```bash
# Download and install the latest release
curl -fsSL https://github.com/mattmezza/tlgram/releases/latest/download/tlgram_$(uname -s)_$(uname -m).tar.gz | tar -xz -C /usr/local/bin tlgram

# Or with wget
wget -qO- https://github.com/mattmezza/tlgram/releases/latest/download/tlgram_$(uname -s)_$(uname -m).tar.gz | tar -xz -C /usr/local/bin tlgram
```

> **Note:** You may need `sudo` for `/usr/local/bin`. Alternatively, install to `~/.local/bin`:
> ```bash
> mkdir -p ~/.local/bin
> curl -fsSL https://github.com/mattmezza/tlgram/releases/latest/download/tlgram_$(uname -s)_$(uname -m).tar.gz | tar -xz -C ~/.local/bin tlgram
> ```

### Manual Download

Download the appropriate binary from [Releases](https://github.com/mattmezza/tlgram/releases):

| Platform | Architecture | Download |
|----------|--------------|----------|
| Linux | x86_64 | `tlgram_VERSION_linux_amd64.tar.gz` |
| Linux | ARM64 | `tlgram_VERSION_linux_arm64.tar.gz` |
| macOS | Intel | `tlgram_VERSION_darwin_amd64.tar.gz` |
| macOS | Apple Silicon | `tlgram_VERSION_darwin_arm64.tar.gz` |
| Windows | x86_64 | `tlgram_VERSION_windows_amd64.zip` |

### Windows Installation

1. Download `tlgram_VERSION_windows_amd64.zip` from [Releases](https://github.com/mattmezza/tlgram/releases)
2. Extract the zip file
3. Move `tlgram.exe` to a directory in your PATH (e.g., `C:\Users\YourName\bin`)
4. Or run it directly from the extracted folder

### Build from Source

Requires Go 1.22+.

```bash
# Clone and build
git clone https://github.com/mattmezza/tlgram.git
cd tlgram
go build -o tlgram ./cmd/tlgram

# Install to your PATH
sudo mv tlgram /usr/local/bin/
```

### Platform Compatibility

- **Linux** - Fully tested and supported
- **macOS** - Builds available but untested
- **Windows** - Builds available but untested (contributions welcome!)

## Quick Start

1. **Get Telegram API credentials**
   - Go to https://my.telegram.org/apps
   - Log in with your phone number
   - Create an application (any name works)
   - Note your `api_id` and `api_hash`

2. **Configure tlgram**

   **Option A: ~/.secrets file** (recommended - keeps secrets out of dotfiles)

   Create `~/.secrets` (this file should NOT be committed to git):
   ```bash
   # ~/.secrets
   export TLGRAM_API_ID=12345678
   export TLGRAM_API_HASH="your_api_hash_here"
   ```

   Source it from your shell rc file:
   ```bash
   # Add to ~/.bashrc or ~/.zshrc
   [ -f ~/.secrets ] && source ~/.secrets
   ```

   **Option B: Config file**
   ```bash
   vim ~/.config/tlgram/config.toml
   ```
   ```toml
   [telegram]
   api_id = 12345678
   api_hash = "your_api_hash_here"
   ```

   Environment variables take precedence over config file values.

3. **Run tlgram**
   ```bash
   # Open chat switcher
   tlgram

   # Open specific chat
   tlgram --chat @username
   ```

4. **Authenticate**
   - Enter your phone number
   - Enter the SMS code from Telegram
   - Enter 2FA password if enabled

## Usage

### Command Line

```bash
tlgram                     # Open chat switcher
tlgram --chat @john_doe    # Open DM with @john_doe
tlgram --chat 1234567890   # Open chat by ID (use I key to find ID)
tlgram --chat work         # Open chat aliased as "work"
tlgram --help              # Show help
tlgram --version           # Show version
tlgram --changelog         # Show changelog
tlgram --default-config    # Print default config (useful for resetting)
```

### Keybindings

#### Navigation (NORMAL mode)

| Key | Action |
|-----|--------|
| `j` / `k` | Move cursor down/up by line |
| `gg` | Jump to first line (loads older messages) |
| `G` | Jump to last line |
| `H` | Move cursor to top of visible viewport |
| `L` | Move cursor to bottom of visible viewport |
| `Ctrl-d` | Scroll half page down |
| `Ctrl-u` | Scroll half page up |
| `Ctrl-f` | Scroll full page down |
| `Ctrl-b` | Scroll full page up |

#### Modes

| Key | Action |
|-----|--------|
| `i` | Enter INSERT mode (compose message) |
| `A` | Jump to bottom and enter INSERT mode |
| `Escape` | Return to NORMAL mode |

#### Actions (NORMAL mode)

| Key | Action |
|-----|--------|
| `Enter` | Select / Open chat |
| `Ctrl-p` | Open chat switcher |
| `r` | Reply to message at cursor |
| `R` | Mark messages as read up to cursor |
| `o` | Jump to original message (for replies) |
| `Ctrl-o` | Jump back after jumping to original |
| `yy` | Copy message at cursor to clipboard |
| `cc` | Edit own message at cursor |
| `D` | Delete own message at cursor |
| `u` | Toggle between full names and @usernames |
| `U` | Mark dialog as unread |
| `I` | Toggle chat ID display in header |
| `d` | Download media |
| `q` | Quit |

#### Chat Switcher

| Key | Action |
|-----|--------|
| `Ctrl-p` | Open switcher |
| `Ctrl-n` | Navigate down |
| `Ctrl-p` | Navigate up |
| `Enter` | Select chat |
| `Ctrl-r` | Mark selected chat as read |
| `Ctrl-u` | Mark selected chat as unread |
| `Escape` | Close switcher |
| Type | Filter chats by name or @username |

#### Insert Mode

| Key | Action |
|-----|--------|
| `Enter` | Insert newline (for multi-line messages) |
| `Alt-Enter` | Send message (stays in INSERT mode) |
| `Escape` | Exit to NORMAL mode |

## Header Bar

The header bar displays contextual information based on chat type:

- **DMs**: `NORMAL Name Surname @username` + connection status
- **Groups**: `NORMAL Group Name (x members)` + connection status
- **Channels**: `NORMAL Channel Name (x subscribers)` + connection status

When there are unread messages, a red badge shows the count on the right side.

## Scroll Indicator

The bottom of the message view shows a scroll indicator:

```
↑ ─ line 50/133 ↓ (3 new)
```

- `↑` / `↓` arrows indicate more content above/below
- Line position shows current cursor location
- `(N new)` appears when new messages arrive while scrolled up

## Configuration

Configuration file: `~/.config/tlgram/config.toml`

To reset your config or see all options: `tlgram --default-config > ~/.config/tlgram/config.toml`

```toml
[telegram]
api_id = 12345678
api_hash = "your_api_hash"

[general]
# Send key: "ctrl-enter" (default, allows multi-line with Enter) or "enter"
send_key = "ctrl-enter"
download_dir = "~/Downloads/tlgram"
# Auto-mark messages as read: -1=disabled (default), 0=instant, >0=delay in seconds
auto_mark_read = -1

[appearance]
# Author display: "fullname" or "username" (toggle with 'u' key)
author_display = "fullname"
reply_preview_length = 30
# Show chat ID in header (toggle with 'I' key)
show_chat_id = false

[chat_aliases]
work = "@john_doe"
team = "-1001234567890"
```

## tmux Integration

tlgram is designed to work seamlessly with tmux. Example session setup:

```bash
# Create a new tmux window with tlgram
tmux new-window -n telegram "tlgram --chat work"

# Split and open another chat
tmux split-window -h "tlgram --chat team"
```

### With hop

If you use [hop](https://github.com/mattmezza/hop) for tmux session templates, add a `.tmux-session` file to your project:

```bash
#!/usr/bin/env bash
# .tmux-session - hop will source this when starting the session

tmux rename-window -t "$SESSION_NAME:1" "code"
tmux send-keys -t "$SESSION_NAME:1" "nvim ." C-m

tmux new-window -t "$SESSION_NAME" -n "telegram"
tmux send-keys -t "$SESSION_NAME:telegram" "tlgram -c work" C-m

tmux select-window -t "$SESSION_NAME:1"
```

## Building

### Requirements

- Go 1.22+

### Build tlgram

```bash
go build -o tlgram ./cmd/tlgram   # Development build
go test ./...                      # Run tests
```

## Contributing

Contributions are welcome! Please read the contributing guidelines first.

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feat/amazing-feature`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [gotd/td](https://github.com/gotd/td) - Pure Go Telegram client library
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling for TUIs
