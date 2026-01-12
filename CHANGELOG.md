# Changelog

All notable changes to tlgram will be documented in this file.

## [Unreleased]

## [0.1.0] - 2026-01-12

### Added
- Terminal-based Telegram client with vim keybindings
- Line-based cursor navigation (j/k moves through wrapped text)
- Multi-line message composition (Enter for newline, Alt-Enter to send)
- Chat switcher with fuzzy search (Ctrl-p)
- CLI chat opening (`tlgram --chat @username` or `tlgram --chat alias`)
- Chat aliases support in config
- Environment variable support for API credentials (TLGRAM_API_ID, TLGRAM_API_HASH)
- Smart header bar showing chat type, members, and connection status
- Unread message indicators
- Message reply support (r key)
- Copy message to clipboard (yy key)
- Toggle between full names and @usernames (u key)
- Scroll indicator with new message count
- Cross-platform builds (Linux, macOS, Windows)

### Technical
- Pure Go implementation using gotd/td library
- No CGO dependencies
- Bubbletea TUI framework
- Lipgloss styling

## [0.0.1] - 2026-01-09

### Added
- Initial project structure
- Basic authentication flow
- Message display and sending
