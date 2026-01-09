# tlgram Requirements Specification

## Overview

This directory contains the complete requirements specification for **tlgram**, a terminal-based Telegram client built with Go, designed for integration with tmux workflows and vim-style navigation.

## Project Context

### User Need
A lightweight, snappy TUI Telegram client that:
- Opens specific chats via CLI arguments (for tmux template integration)
- Supports vim keybindings for efficient navigation
- Runs multiple instances simultaneously (different chats in different tmux panes)
- Provides fuzzy search for quick chat switching
- Works seamlessly in terminal/tmux/SSH environments

### Target Users
- Terminal power users
- Vim users
- Developers using tmux for project management
- Users who want Telegram integrated into their terminal workflow

### Technology Stack
- **Language**: Go (golang)
- **Telegram API**: TDLib (official Telegram client library)
- **UI Framework**: Terminal UI library (e.g., bubbletea/lipgloss)
- **Target Platforms**: Linux (primary), macOS, Windows

## Requirements Index

### Functional Requirements (FR)

#### Core Messaging
- **[FR-001](FR-001-user-authentication.md)**: User Authentication
  - Phone + SMS + 2FA support
  - Session persistence and reuse
  - Multi-instance session sharing

- **[FR-004](FR-004-send-text-messages.md)**: Send Text Messages
  - Multi-line message composition
  - Configurable send keybinding (Enter vs Ctrl-Enter)
  - Message queuing for offline support

- **[FR-005](FR-005-receive-display-messages.md)**: Receive and Display Messages
  - Real-time message reception (<50ms latency)
  - Minimal display format (sender + text)
  - Unread message separator

- **[FR-006](FR-006-reply-to-messages.md)**: Reply to Messages
  - Quote/reply to specific messages
  - Clear visual reply context
  - Essential for group chat usability

#### Chat Management
- **[FR-002](FR-002-chat-list-display.md)**: Chat List Display
  - Show DMs and Groups
  - Unread indicators and counts
  - Real-time updates

- **[FR-003](FR-003-open-specific-chat-cli.md)**: Open Specific Chat via CLI
  - Support username, chat ID, and alias
  - Critical for tmux/hop template integration
  - `tlgram --chat @username` or `tlgram --chat work`

- **[FR-008](FR-008-chat-switching.md)**: Chat Switching (fzf-like fuzzy search)
  - Ctrl-p to open chat switcher
  - Fuzzy search across chat names
  - Fast, responsive (<50ms updates)

- **[FR-009](FR-009-search-within-chat.md)**: Search Within Chat
  - Vim-style search (/ to search, n/N to navigate)
  - Search message text
  - Highlight matches

#### Navigation & Interaction
- **[FR-010](FR-010-vim-keybindings.md)**: Vim Keybindings
  - hjkl navigation, gg/G, Ctrl-d/u
  - Modal interface (NORMAL, INSERT, SEARCH, COMMAND)
  - Vim muscle memory support

- **[FR-014](FR-014-copy-message-text.md)**: Copy Message Text to Clipboard
  - 'y' key to yank/copy
  - System clipboard integration
  - Works in tmux and SSH sessions (OSC 52)

#### Media & Files
- **[FR-007](FR-007-media-file-handling.md)**: Media File Handling
  - Display file info (type, name, size)
  - Download on demand ('d' key)
  - No inline preview (keep it lightweight)

#### Configuration & Customization
- **[FR-011](FR-011-configuration-management.md)**: Configuration Management
  - TOML config file at `~/.config/tlgram/config.toml`
  - Customize keybindings, download path, send behavior
  - Well-documented with examples

- **[FR-012](FR-012-chat-aliases.md)**: Chat Aliases
  - Define short names for chats (e.g., `work = "@john_doe"`)
  - Use in CLI and commands: `tlgram --chat work`
  - Essential for hop template integration

#### Advanced Features
- **[FR-013](FR-013-markdown-rendering.md)**: Markdown Rendering
  - Display bold, italic, code, links
  - Use Telegram's entity data
  - Readable formatted messages

- **[FR-015](FR-015-unread-message-indicators.md)**: Unread Message Indicators
  - Unread count in chat list
  - Visual separator for unread messages in chat
  - Clear indication of new messages since last focus

- **[FR-016](FR-016-status-bar-display.md)**: Status Bar Display
  - Show current chat name + connection status
  - Temporary notifications (Copied!, Sent!, etc.)
  - Mode indicator (NORMAL, INSERT, etc.)

#### Reliability & Network
- **[FR-017](FR-017-network-reconnection.md)**: Network Reconnection
  - Auto-reconnect with exponential backoff
  - Message queuing during offline
  - Status updates in status bar

- **[FR-018](FR-018-multiple-instance-support.md)**: Multiple Instance Support
  - Run multiple instances simultaneously
  - Shared authentication, isolated UI state
  - Critical for tmux multi-pane workflow

### Non-Functional Requirements (NFR)

- **[NFR-001](NFR-001-performance.md)**: Performance
  - **"Snappy as fuck"** - <50ms UI updates
  - Startup time <2s
  - Efficient rendering and data structures
  - Multiple instances without degradation

- **[NFR-002](NFR-002-reliability.md)**: Reliability
  - No crashes or panics
  - Graceful error handling
  - Comprehensive logging
  - Data integrity (queued messages, unread state)

- **[NFR-003](NFR-003-security.md)**: Security
  - Proper Telegram authentication (TDLib)
  - Encrypted network communication (MTProto)
  - Secure session file permissions (0600)
  - No credential logging

- **[NFR-004](NFR-004-usability.md)**: Usability
  - Minimal, clean UI
  - Vim muscle memory support
  - Clear error messages
  - Efficient workflows (minimal keystrokes)

- **[NFR-005](NFR-005-maintainability.md)**: Maintainability
  - Clean code organization
  - Comprehensive documentation
  - Unit and integration tests
  - Standard Go practices

- **[NFR-006](NFR-006-compatibility.md)**: Compatibility
  - Linux, macOS, Windows support
  - Works in tmux, screen, SSH sessions
  - Adapts to terminal capabilities
  - Minimal dependencies

## Requirements Summary

### Critical Requirements (Must Have)
1. User authentication with session persistence (FR-001)
2. Open specific chat via CLI argument (FR-003)
3. Send and receive text messages (FR-004, FR-005)
4. Vim keybindings (FR-010)
5. Multiple instance support (FR-018)
6. Performance: <50ms UI updates (NFR-001)
7. Security: proper auth and encryption (NFR-003)

### High Priority Requirements (Should Have)
1. Chat list display with unread indicators (FR-002)
2. Chat switching with fuzzy search (FR-008)
3. Reply to messages (FR-006)
4. Chat aliases (FR-012)
5. Unread message indicators (FR-015)
6. Network reconnection (FR-017)
7. Usability: vim-like experience (NFR-004)

### Medium Priority Requirements (Nice to Have)
1. Media file handling (FR-007)
2. Search within chat (FR-009)
3. Configuration management (FR-011)
4. Markdown rendering (FR-013)
5. Copy message text (FR-014)
6. Status bar (FR-016)
7. Reliability and error handling (NFR-002)
8. Maintainability (NFR-005)
9. Compatibility (NFR-006)

## Development Phases

### Phase 1: MVP (Minimum Viable Product)
**Goal**: Basic usable client for single chat workflow
- FR-001: Authentication
- FR-003: Open chat via CLI
- FR-004: Send text messages
- FR-005: Receive and display messages
- FR-010: Basic vim keybindings (hjkl, gg/G)
- NFR-001: Performance basics
- NFR-003: Security basics

**Deliverable**: Can open a specific chat, send/receive messages, vim navigation works

### Phase 2: Multi-Chat & Workflow
**Goal**: Full tmux workflow support
- FR-002: Chat list
- FR-008: Chat switcher
- FR-012: Chat aliases
- FR-018: Multiple instances
- FR-015: Unread indicators
- FR-016: Status bar

**Deliverable**: Can run multiple instances, switch chats, see unread counts

### Phase 3: Polish & Features
**Goal**: Complete feature set
- FR-006: Reply to messages
- FR-007: Media handling
- FR-009: Search
- FR-011: Configuration
- FR-013: Markdown rendering
- FR-014: Copy to clipboard
- FR-017: Network reconnection

**Deliverable**: Feature-complete v1.0

### Phase 4: Quality & Release
**Goal**: Production-ready release
- NFR-002: Comprehensive error handling and logging
- NFR-004: UX refinement
- NFR-005: Code cleanup and documentation
- NFR-006: Multi-platform testing
- Testing, bug fixes, documentation

**Deliverable**: Stable v1.0 release

## Success Criteria

### User Acceptance
- ✅ User can integrate tlgram into hop templates
- ✅ User can run multiple chat windows in tmux simultaneously
- ✅ User can switch between chats quickly (<2 seconds)
- ✅ Application feels "snappy as fuck" (user's words)
- ✅ Vim keybindings work as expected (muscle memory)

### Technical Metrics
- ✅ UI responsiveness: 95th percentile <50ms
- ✅ Startup time: <2 seconds (warm start)
- ✅ No crashes or data loss in normal operation
- ✅ Works in tmux, SSH, various terminals
- ✅ Can run 3-5 instances without performance issues

### Quality Metrics
- ✅ Test coverage >50% for critical paths
- ✅ All critical requirements implemented
- ✅ No critical bugs in issue tracker
- ✅ Documentation complete and clear

## Next Steps for Engineer

1. **Review Requirements**: Read all FR and NFR documents thoroughly
2. **Technology Evaluation**:
   - Choose TUI framework (bubbletea recommended)
   - Set up TDLib integration
   - Evaluate Go clipboard libraries
3. **Architecture Design**:
   - High-level system design
   - Component breakdown
   - Data flow diagrams
4. **Prototype**: Build minimal proof-of-concept
   - TDLib connection
   - Basic TUI with message display
   - Vim keybinding handling
5. **Iterative Development**: Follow phased approach above
6. **Testing**: Continuous testing throughout development
7. **Documentation**: Keep docs updated as code evolves

## Questions for Clarification

Before starting implementation, engineer should clarify:
- [ ] Preferred TUI framework (bubbletea vs. tview vs. termbox-go)
- [ ] TDLib build approach (static linking vs. dynamic)
- [ ] Package distribution preferences
- [ ] CI/CD setup expectations
- [ ] Any specific Go coding standards

## Contact

For questions or clarifications about these requirements, contact the project stakeholder (Matteo).

---

**Document Version**: 1.0
**Last Updated**: 2026-01-09
**Status**: Complete - Ready for Development
