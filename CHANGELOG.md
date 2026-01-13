# Changelog

All notable changes to tlgram will be documented in this file.

## [Unreleased]

## [0.1] - 2026-01-13

### Added
- Terminal-based Telegram client with vim keybindings
- Line-based cursor navigation (j/k moves through wrapped text)
- Multi-line message composition (Enter for newline, Alt-Enter to send)
- Dynamic textarea height (grows up to half terminal height)
- Chat switcher with fuzzy search (Ctrl-p)
- Mark messages as read up to cursor (R key in chat view)
- Mark dialog as unread (U key in chat view, Ctrl-u in switcher)
- Mark chat as read from switcher (Ctrl-r)
- Visual styling for unread messages (red bold bar indicator)
- Reply context preview showing original message snippet above replies
- Jump to original message with `o` key when on a reply
- Jump back with `Ctrl-o` after jumping to original (vim-style jump stack)
- CLI chat opening (`tlgram --chat @username` or `tlgram --chat alias`)
- Chat aliases support in config
- Environment variable support for API credentials (TLGRAM_API_ID, TLGRAM_API_HASH)
- Smart header bar showing chat type, members, and connection status
- Unread message indicators
- Message reply support (r key)
- Copy message to clipboard (yy key)
- Toggle between full names and @usernames (u key)
- Scroll indicator with new message count
- Configurable reply preview length (`reply_preview_length` in config)
- Cross-platform builds (Linux, macOS, Windows)

### Technical
- Pure Go implementation using gotd/td library
- No CGO dependencies
- Bubbletea TUI framework
- Lipgloss styling
