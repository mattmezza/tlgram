# NFR-006: Compatibility

## Description
The application must work across different operating systems, terminal emulators, and environments, particularly with tmux and SSH sessions.

## Priority
**MEDIUM** - Important for broad usability

## Compatibility Requirements

### AC-006.1: Operating Systems
The application SHALL run on:
- **Linux** (primary target, user's environment)
  - Ubuntu, Debian, Fedora, Arch, etc.
  - x86_64, ARM64 architectures
- **macOS**
  - Intel and Apple Silicon
- **Windows** (lower priority)
  - Windows 10/11
  - Windows Terminal or similar modern terminal

### AC-006.2: Terminal Emulators
The application SHALL work correctly in:
- **tmux** (critical - user's primary environment)
- **GNU Screen**
- **kitty**
- **alacritty**
- **iTerm2** (macOS)
- **GNOME Terminal**
- **xterm**
- **Windows Terminal**
- **SSH sessions** (local terminal → remote server)

### AC-006.3: Terminal Capabilities
The application SHALL adapt to terminal capabilities:
- **256-color terminals**: Use colors for visual distinction
- **Truecolor (24-bit) terminals**: Use full color palette
- **Basic 16-color terminals**: Fall back to standard colors
- **Monochrome terminals**: Use bold/underline for distinction
- **Unicode support**: Use Unicode characters for UI
- **ASCII-only terminals**: Fall back to ASCII characters

### AC-006.4: Terminal Size
- **Minimum size**: 40 columns × 10 rows
  - Should display basic UI (possibly cramped)
- **Recommended size**: 80 columns × 24 rows or larger
  - Comfortable reading and interaction
- **Large terminals**: Utilize space efficiently
  - Don't waste space with excessive padding
- **Responsive**: Adapt to terminal resize
  - Listen for SIGWINCH signal
  - Redraw UI on resize

### AC-006.5: Tmux Integration
- **Clipboard**: Work correctly with tmux clipboard
  - Detect tmux environment (`$TMUX` variable)
  - Use `tmux set-buffer` when appropriate
  - Fall back to OSC 52 or system clipboard
- **Colors**: Colors render correctly in tmux
  - Ensure `TERM` is set correctly (e.g., `tmux-256color`)
- **Keybindings**: No conflicts with default tmux bindings
  - Ctrl-p conflict: User can remap in config
- **Multiple panes**: Work correctly in split panes

### AC-006.6: SSH Sessions
- **Remote execution**: Work when SSH'd into remote server
  - Local terminal → remote server → run tlgram
- **Clipboard**: Use OSC 52 for clipboard sync to local machine
  - Works in: iTerm2, kitty, Windows Terminal, modern terminals
  - Configurable (some users disable for security)
- **Latency**: Remain usable with 50-200ms SSH latency
  - Optimize for low bandwidth
  - Buffer output efficiently

### AC-006.7: Shell Compatibility
The application SHALL work regardless of user's shell:
- **bash**
- **zsh**
- **fish**
- **sh** (minimal shell)
- No shell-specific dependencies

### AC-006.8: Environment Variables
Respect standard environment variables:
- `$TERM`: Terminal type
- `$COLORTERM`: Truecolor support indicator
- `$EDITOR`: Used for external editor (optional feature)
- `$TMUX`: Tmux detection
- `$HOME`: User's home directory
- `$XDG_CONFIG_HOME`: Config directory (default: `~/.config`)

### AC-006.9: Locale and Character Encoding
- **UTF-8**: Primary encoding (assume UTF-8 by default)
- **Emojis**: Display correctly in UTF-8 terminals
- **Right-to-left text**: Basic support (Telegram messages may contain RTL)
- **Locale**: Respect user's locale for date/time formatting

### AC-006.10: System Dependencies
Minimize external dependencies:
- **Required**: None (TDLib is bundled or statically linked)
- **Optional**: Clipboard utilities (xclip, xsel, wl-copy, pbcopy)
  - Application should work without them (fall back to OSC 52)
- **Optional**: External viewer for files (feh, mpv, etc.)
  - User configures in config file

### AC-006.11: Installation Methods
Support multiple installation methods:
- **Binary release**: Statically linked executable
  - Download and run, no installation needed
- **Package managers**:
  - Arch AUR
  - Homebrew (macOS, Linux)
  - APT repository (Debian/Ubuntu) - future
- **Build from source**: `go build`
  - Requires Go 1.21+
  - Requires TDLib (provide build instructions)

### AC-006.12: TDLib Compatibility
- Support TDLib 1.8.0+ (latest stable)
- Document which TDLib version is supported
- Handle TDLib API changes gracefully
- Provide build instructions for TDLib

### AC-006.13: Go Version
- Require Go 1.21 or later
- Use Go modules for compatibility
- Avoid using bleeding-edge Go features

### AC-006.14: File System
- Work on case-sensitive and case-insensitive filesystems
- Use forward slashes in code, let Go handle path conversion
- Respect file permissions (Unix vs. Windows)

### AC-006.15: Terminal Multiplexer Detection
Detect and adapt to environment:
```go
// Detect tmux
inTmux := os.Getenv("TMUX") != ""

// Detect screen
inScreen := os.Getenv("STY") != ""

// Detect SSH
inSSH := os.Getenv("SSH_CONNECTION") != ""
```

## Testing Matrix

### OS × Architecture:
| OS | Architecture | Priority |
|----|--------------|----------|
| Linux | x86_64 | High |
| Linux | ARM64 | Medium |
| macOS | x86_64 (Intel) | Medium |
| macOS | ARM64 (M1/M2) | Medium |
| Windows | x86_64 | Low |

### Terminal × Environment:
| Terminal | Environment | Priority |
|----------|-------------|----------|
| tmux | Native Linux | Critical |
| tmux | SSH session | High |
| kitty | Native Linux | Medium |
| alacritty | Native Linux | Medium |
| iTerm2 | Native macOS | Medium |
| Windows Terminal | Native Windows | Low |

### Manual Testing:
- Test in primary environments first (tmux on Linux)
- Test SSH sessions with various latencies
- Test terminal resize behavior
- Test clipboard in different environments
- Test with minimal terminal capabilities

## Build Configuration

### Cross-compilation:
```bash
# Linux x86_64
GOOS=linux GOARCH=amd64 go build -o tlgram-linux-amd64

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o tlgram-linux-arm64

# macOS x86_64
GOOS=darwin GOARCH=amd64 go build -o tlgram-darwin-amd64

# macOS ARM64
GOOS=darwin GOARCH=arm64 go build -o tlgram-darwin-arm64

# Windows x86_64
GOOS=windows GOARCH=amd64 go build -o tlgram-windows-amd64.exe
```

### Static Linking (for portability):
```bash
# Link TDLib statically (if possible)
CGO_ENABLED=1 go build -ldflags '-extldflags "-static"' -o tlgram
```

## Known Limitations

### Not Supported:
- Terminals without Unicode support (very rare)
- Terminals smaller than 40×10
- Very old TDLib versions (<1.8.0)
- Windows console without UTF-8 support

### Workarounds:
- If terminal too small: Display error message
- If Unicode not supported: Fall back to ASCII
- If TDLib too old: Display error with upgrade instructions

## Technical Notes
- Use `golang.org/x/term` for terminal capabilities
- Use `golang.org/x/sys/unix` for Unix-specific features
- Test with `TERM` set to different values
- Use Go's cross-platform abstractions (filepath, os)

## Dependencies
- FR-014: Copy Message Text (clipboard compatibility)
- FR-018: Multiple Instance Support (tmux panes)

## Related Requirements
- NFR-001: Performance (must be responsive even over SSH)
- NFR-004: Usability (works in user's environment)
