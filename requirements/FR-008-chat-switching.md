# FR-008: Chat Switching (fzf-like fuzzy search)

## Description
The application must provide a fast, fuzzy-search interface for switching between chats without leaving the current session.

## Priority
**HIGH** - Critical for efficient workflow

## User Story
As a user, I want to quickly switch between different chats using fuzzy search, so that I can temporarily check another conversation without losing my place in the current chat.

## Acceptance Criteria

### AC-008.1: Trigger Keybinding
- Pressing `Ctrl-p` (or configurable keybinding) SHALL open the chat switcher
- The chat switcher SHALL work from any view (chat list or chat view)
- The current view SHALL be temporarily replaced with the switcher interface

### AC-008.2: Chat Switcher Interface
- The switcher SHALL display:
  - An input field at the top for search query
  - A filtered list of chats below (max 10-15 visible)
  - Unread indicators for each chat
  - Match highlighting (show which parts of chat name match the query)

### AC-008.3: Fuzzy Search Behavior
- As user types, the list SHALL filter in real-time
- Search SHALL match against:
  - Chat names
  - Usernames (for DMs)
  - Group names
- Fuzzy matching SHALL allow non-consecutive character matches
  - Example: typing "jnd" should match "john_doe"
  - Example: typing "proj" should match "project_alpha"
- Matches SHALL be ranked by relevance:
  - Exact prefix matches first
  - Word boundary matches second
  - Other fuzzy matches last
  - Chats with unread messages boosted in ranking

### AC-008.4: Navigation in Switcher
- Users SHALL navigate the filtered list with:
  - `j`/`k` or arrow keys to move up/down
  - `Ctrl-n`/`Ctrl-p` as alternative navigation
  - `Enter` to select the highlighted chat
  - `Escape` to cancel and return to previous view

### AC-008.5: Chat Selection
- Pressing `Enter` SHALL:
  - Close the switcher
  - Open the selected chat (replacing current view)
  - Jump to the latest message in the selected chat
  - Mark the chat as focused (for unread separator tracking)

### AC-008.6: Cancellation
- Pressing `Escape` SHALL:
  - Close the switcher
  - Return to the previous view (chat list or chat)
  - Restore the exact previous state (scroll position, etc.)

### AC-008.7: Empty Search
- With empty search query, the switcher SHALL display:
  - All chats sorted by recent activity
  - Unread chats at the top
- This provides a quick way to see all chats

### AC-008.8: Performance
- The search SHALL feel instant (<50ms from keystroke to UI update)
- The fuzzy matching algorithm SHALL be efficient for 100+ chats
- UI SHALL NOT lag or stutter during typing

### AC-008.9: Visual Design
- The switcher SHALL be visually distinct (e.g., bordered box, different background)
- Current selected item SHALL be highlighted
- Match highlights SHALL be clearly visible (e.g., bold, color)

## Example Display
```
╔════════════════════════════════════════╗
║ Search: proj_                          ║
╠════════════════════════════════════════╣
║ > [1] project_alpha (3)                ║
║   [2] project_beta                     ║
║   [3] @john_doe (project discussion)   ║
╚════════════════════════════════════════╝
```

## Technical Notes
- Implement efficient fuzzy matching algorithm (e.g., fzf-style)
- Pre-compute searchable strings for each chat
- Use incremental search to avoid re-scanning full list
- Consider caching search results for performance

## Dependencies
- FR-002: Chat List Display (provides list of chats)
- FR-010: Vim Keybindings (for navigation keys)

## Related Requirements
- FR-003: Open Specific Chat via CLI
- NFR-001: Performance (<50ms updates)
- NFR-004: Usability (efficient workflow)
