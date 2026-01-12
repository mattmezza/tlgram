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

### Pre-built Binaries

Download from [Releases](https://github.com/mattmezza/tlgram/releases).

### Build from Source

Requires Go 1.22+.

```bash
# Clone the repository
git clone https://github.com/mattmezza/tlgram.git
cd tlgram

# Build tlgram
go build -o tlgram ./cmd/tlgram
```

## Quick Start

1. **Get Telegram API credentials**
   - Go to https://my.telegram.org/apps
   - Create an application
   - Note your `api_id` and `api_hash`

2. **Configure tlgram**

   **Option A: Environment variables** (recommended for dotfiles)
   ```bash
   export TLGRAM_API_ID=12345678
   export TLGRAM_API_HASH="your_api_hash_here"
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
tlgram --chat -100123456   # Open group by ID
tlgram --chat work         # Open chat aliased as "work"
tlgram --help              # Show help
tlgram --version           # Show version
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
| `Escape` | Return to NORMAL mode |

#### Actions (NORMAL mode)

| Key | Action |
|-----|--------|
| `Enter` | Select / Open chat |
| `Ctrl-p` | Open chat switcher |
| `r` | Reply to message at cursor |
| `yy` | Copy message at cursor to clipboard |
| `u` | Toggle between full names and @usernames |
| `d` | Download media |
| `q` | Quit |

#### Chat Switcher

| Key | Action |
|-----|--------|
| `Ctrl-p` | Open switcher |
| `Ctrl-n` | Navigate down |
| `Ctrl-p` | Navigate up |
| `Enter` | Select chat |
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

```toml
[telegram]
api_id = 12345678
api_hash = "your_api_hash"

[general]
# Send key: "ctrl-enter" (default, allows multi-line with Enter) or "enter"
send_key = "ctrl-enter"
download_dir = "~/Downloads/tlgram"
auto_mark_read = true
initial_message_count = 50

[appearance]
# Author display: "fullname" or "username" (toggle with 'u' key)
author_display = "fullname"

[chat_aliases]
work = "@john_doe"
team = "-1001234567890"

[keybindings]
chat_switcher = "ctrl+p"
reply = "r"
copy = "yy"
download = "d"

[network]
auto_reconnect = true
reconnect_delay = 2

[logging]
level = "info"
log_file = "~/.config/tlgram/logs/app.log"
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

If you use [hop](https://github.com/mattmezza/hop) for tmux templates:

```yaml
# ~/.config/hop/templates/project.yml
windows:
  - name: code
    panes:
      - nvim
  - name: telegram
    panes:
      - tlgram --chat work
      - tlgram --chat team
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
