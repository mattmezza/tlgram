# tlgram

A terminal-based Telegram client with vim keybindings, designed for tmux workflows.

## Features

- **Vim keybindings** - Navigate with hjkl, gg/G, Ctrl-d/u
- **CLI chat opening** - `tlgram --chat @username` or `tlgram --chat work`
- **Multiple instances** - Run different chats in different tmux panes
- **Fuzzy chat switcher** - Ctrl-p to search and switch chats
- **Chat aliases** - Define shortcuts like `work = "@john_doe"`
- **Snappy performance** - <50ms UI updates

## Installation

### Pre-built Binaries

Download from [Releases](https://github.com/mattmezza/tlgram/releases).

### Build from Source

Requires Go 1.22+ and TDLib.

```bash
# Install TDLib (see docs/tdlib.md for details)
./scripts/build-tdlib.sh

# Build tlgram
make build
```

## Quick Start

1. **Get Telegram API credentials**
   - Go to https://my.telegram.org/apps
   - Create an application
   - Note your `api_id` and `api_hash`

2. **Configure tlgram**
   ```bash
   # Edit config file
   vim ~/.config/tlgram/config.toml
   ```

   Add your credentials:
   ```toml
   [telegram]
   api_id = 12345678
   api_hash = "your_api_hash_here"
   ```

3. **Run tlgram**
   ```bash
   # Open chat list
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
tlgram                     # Open chat list
tlgram --chat @john_doe    # Open DM with @john_doe
tlgram --chat -100123456   # Open group by ID
tlgram --chat work         # Open chat aliased as "work"
tlgram --help              # Show help
tlgram --version           # Show version
```

### Keybindings

#### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `gg` | Jump to top |
| `G` | Jump to bottom |
| `Ctrl-d` | Scroll half page down |
| `Ctrl-u` | Scroll half page up |

#### Modes

| Key | Action |
|-----|--------|
| `i` | Enter INSERT mode (compose message) |
| `/` | Enter SEARCH mode |
| `:` | Enter COMMAND mode |
| `Escape` | Return to NORMAL mode |

#### Actions

| Key | Action |
|-----|--------|
| `Enter` | Select / Open chat |
| `Ctrl-p` | Open chat switcher |
| `r` | Reply to message |
| `y` | Copy message to clipboard |
| `d` | Download media |
| `:q` | Quit |

#### Insert Mode

| Key | Action |
|-----|--------|
| `Ctrl-Enter` | Send message |
| `Enter` | New line |
| `Escape` | Exit to NORMAL mode |

## Configuration

Configuration file: `~/.config/tlgram/config.toml`

```toml
[telegram]
api_id = 12345678
api_hash = "your_api_hash"

[general]
send_key = "ctrl-enter"  # or "enter"
download_dir = "~/Downloads/tlgram"
auto_mark_read = true
initial_message_count = 50

[chat_aliases]
work = "@john_doe"
team = "-1001234567890"

[keybindings]
chat_switcher = "ctrl+p"
search = "/"
reply = "r"
copy = "y"
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
- TDLib 1.8.31+
- CMake, gperf, OpenSSL, zlib (for building TDLib)

### Build TDLib

```bash
# Automatic build
./scripts/build-tdlib.sh

# Or use Docker for reproducible builds
make docker-tdlib
```

### Build tlgram

```bash
make build           # Development build
make build-static    # Static binary for distribution
make test            # Run tests
make lint            # Run linter
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

- [TDLib](https://github.com/tdlib/td) - Telegram Database Library
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling for TUIs
