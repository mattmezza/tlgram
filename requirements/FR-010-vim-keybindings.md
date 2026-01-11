# FR-010: Vim Keybindings

## Description
The application must support core vim-style keybindings for navigation and interaction throughout the interface.

## Priority
**HIGH** - Critical for user workflow efficiency

## User Story
As a user familiar with vim, I want to use vim keybindings throughout the application, so that I can navigate and interact efficiently using muscle memory.

## Acceptance Criteria

### AC-010.1: Basic Navigation (All Views)
- `j` or `Down Arrow` - Move down one item/line
- `k` or `Up Arrow` - Move up one item/line
- `h` or `Left Arrow` - Move left (when applicable)
- `l` or `Right Arrow` - Move right (when applicable)
- `gg` - Jump to top (first message/chat)
- `G` (Shift-g) - Jump to bottom (last message/chat)

### AC-010.2: Page-wise Navigation
- `Ctrl-d` - Scroll down half page
- `Ctrl-u` - Scroll up half page
- `Ctrl-f` - Scroll down full page (optional)
- `Ctrl-b` - Scroll up full page (optional)

### AC-010.3: Search (DEFERRED)
**STATUS: Not implemented in current version. See FR-009.**
- `/` - Enter search mode
- `n` - Next search match
- `N` (Shift-n) - Previous search match
- `Escape` - Exit search mode

### AC-010.4: Command Mode (DEFERRED)
**STATUS: Not implemented in current version. Planned for future release.**
- `:` - Enter command mode
- Commands SHALL include:
  - `:q` or `:quit` - Quit application
  - `:h` or `:help` - Show help
  - `:chat <name>` - Switch to chat (alternative to Ctrl-p)
- `Escape` - Exit command mode without executing

### AC-010.5: Message Actions
- `r` - Reply to selected message (see FR-006)
- `y` - Yank (copy) selected message text to clipboard (see FR-014)
- `i` - Enter insert/compose mode (focus input area)
- `Escape` - Exit insert mode, return focus to message list

### AC-010.6: Chat Switching
- `Ctrl-p` - Open chat switcher (see FR-008)
- `:ls` - List all chats (optional, alternative to Ctrl-p)

### AC-010.7: Mode Indicators
- The application SHALL clearly indicate current mode:
  - **NORMAL** - Default mode, vim navigation active
  - **INSERT** - Composing message, text input active
- Mode indicator SHALL be visible in status bar or dedicated area
- Note: SEARCH and COMMAND modes are deferred (see AC-010.3, AC-010.4)

### AC-010.8: Mode Switching
- In NORMAL mode:
  - `i` enters INSERT mode
- In INSERT mode:
  - `Escape` returns to NORMAL mode
- Send keybinding (Alt-Enter, default) in INSERT mode sends message and stays in INSERT mode
- Note: Multi-line messages supported via Enter key (inserts newline), Alt-Enter sends

### AC-010.9: Input Area Keybindings
When in INSERT mode (composing message):
- Regular vim keybindings SHALL NOT apply (allow normal text entry)
- Emacs-style line editing SHALL be available:
  - `Ctrl-a` - Move to start of line
  - `Ctrl-e` - Move to end of line
  - `Ctrl-u` - Delete from cursor to start of line
  - `Ctrl-k` - Delete from cursor to end of line
  - `Ctrl-w` - Delete word before cursor
- Arrow keys SHALL work for navigation within input

### AC-010.10: Visual Feedback
- Currently selected message/chat SHALL be visually highlighted
- Cursor position SHALL be clearly visible
- Mode transitions SHALL be immediate (<50ms)

### AC-010.11: Keybinding Conflicts
- The application SHALL handle keybinding conflicts gracefully:
  - Context-aware bindings (e.g., `/` searches in NORMAL mode, types "/" in INSERT mode)
  - Clear mode separation prevents ambiguity

### AC-010.12: No Visual Mode (v1)
- Visual mode is explicitly excluded from v1
- No character/line selection with `v` or `V`
- Actions operate on currently selected message only

## Keybinding Summary Table

| Key | Mode | Action |
|-----|------|--------|
| `j` / `↓` | NORMAL | Move down |
| `k` / `↑` | NORMAL | Move up |
| `h` / `←` | NORMAL | Move left |
| `l` / `→` | NORMAL | Move right |
| `gg` | NORMAL | Jump to top (loads older messages) |
| `G` | NORMAL | Jump to bottom |
| `H` | NORMAL | Move to top of viewport |
| `L` | NORMAL | Move to bottom of viewport |
| `Ctrl-d` | NORMAL | Scroll down half page |
| `Ctrl-u` | NORMAL | Scroll up half page |
| `Ctrl-f` | NORMAL | Scroll down full page |
| `Ctrl-b` | NORMAL | Scroll up full page |
| `r` | NORMAL | Reply to message |
| `yy` | NORMAL | Yank/copy message |
| `u` | NORMAL | Toggle fullname/@username display |
| `i` | NORMAL | Enter insert mode |
| `Ctrl-p` | NORMAL | Chat switcher |
| `q` | NORMAL | Quit |
| `Escape` | INSERT | Return to NORMAL |
| `Enter` | INSERT | Insert newline |
| `Alt-Enter` | INSERT | Send message |

## Technical Notes
- Implement mode state machine
- Use context-aware keybinding handlers
- Ensure mode transitions are atomic and instant
- Consider using a keybinding library or implementing custom handler

## Dependencies
- FR-006: Reply to Messages (for 'r' key)
- FR-008: Chat Switching (for Ctrl-p)
- FR-009: Search Within Chat (DEFERRED - for /, n, N)
- FR-014: Copy Message Text (for 'yy' key)

## Related Requirements
- NFR-001: Performance (instant mode transitions)
- NFR-004: Usability (vim muscle memory)
