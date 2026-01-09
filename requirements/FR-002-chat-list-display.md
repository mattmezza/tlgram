# FR-002: Chat List Display

## Description
When launched without the `--chat` argument, the application must display a navigable list of all chats (DMs and groups).

## Priority
**HIGH** - Core functionality for chat discovery

## User Story
As a user, when I launch the application without specifying a chat, I want to see a list of my recent chats so that I can select which conversation to open.

## Acceptance Criteria

### AC-002.1: Chat List Content
- The application SHALL display all Direct Message (DM) chats
- The application SHALL display all Group chats the user is a member of
- The application SHALL NOT display Channels (read-only broadcasts)
- The application SHALL sort chats by most recent activity (newest first)

### AC-002.2: Chat Information Display
- Each chat entry SHALL display:
  - Chat name (contact name for DMs, group name for groups)
  - Unread message indicator (if unread messages exist)
  - Last message preview (optional, first 50 characters)

### AC-002.3: Unread Indicators
- Chats with unread messages SHALL be visually distinguished (e.g., bold text, color, indicator)
- The application SHALL display unread message count for each chat with unreads
- The unread indicator SHALL update in real-time when new messages arrive

### AC-002.4: Navigation
- Users SHALL be able to navigate the list using vim-style `j`/`k` keys or arrow keys
- Users SHALL be able to jump to top with `gg` and bottom with `G`
- Users SHALL be able to scroll page-wise with `Ctrl-d` (down) and `Ctrl-u` (up)

### AC-002.5: Chat Selection
- Pressing `Enter` on a selected chat SHALL open that chat
- The chat list SHALL be replaced with the single-pane chat view

### AC-002.6: Real-time Updates
- The chat list SHALL update in real-time when:
  - New messages arrive
  - Chat positions change due to activity
  - New chats are created

## Technical Notes
- Fetch chat list via TDLib API
- Implement efficient rendering to maintain <50ms update time
- Use efficient data structures for real-time updates

## Dependencies
- FR-001: User Authentication (must be authenticated to fetch chats)

## Related Requirements
- FR-008: Chat Switching (allows returning to this list)
- FR-015: Unread Message Indicators
- NFR-001: Performance (<50ms updates)
