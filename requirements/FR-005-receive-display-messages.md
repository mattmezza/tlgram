# FR-005: Receive and Display Messages

## Description
The application must receive messages from Telegram in real-time and display them in the chat view with minimal, clean formatting.

## Priority
**CRITICAL** - Core messaging functionality

## User Story
As a user, I want to see messages from my contacts and groups as they arrive in real-time, so that I can stay informed and respond promptly.

## Acceptance Criteria

### AC-005.1: Message Reception
- The application SHALL receive new messages in real-time via Telegram's push notifications
- The application SHALL display new messages immediately upon receipt (<50ms latency)
- The application SHALL work for both DMs and group chats

### AC-005.2: Message List Display
- Messages SHALL be displayed in chronological order (oldest at top, newest at bottom)
- The message list SHALL auto-scroll to the newest message when a new message arrives
- IF user has scrolled up to view history, auto-scroll SHALL be disabled until user returns to bottom

### AC-005.3: Message Format - Minimal Display
- Each message SHALL display:
  - **Sender name**: Contact name (for DMs) or member name (for groups)
  - **Message text**: The message content
- Messages SHALL NOT display timestamp by default
- When a message is selected/focused, the application SHALL display:
  - Full timestamp (e.g., "2026-01-09 14:23:45")
  - Read status (if applicable)
  - Edit indicator (if message was edited)

### AC-005.4: Visual Message Grouping
- Consecutive messages from the same sender SHALL be visually grouped
- Only the first message in a group SHALL show the sender name
- Subsequent messages SHALL be indented or shown without sender name

### AC-005.5: Unread Message Separator
- The application SHALL display a visual separator/marker indicating unread messages
- The separator SHALL appear between the last read message and first unread message
- The separator SHALL persist until the user focuses the chat and scrolls past it
- When returning to a chat after hours away, it SHALL be clear which messages are new

### AC-005.6: Message History Loading
- When opening a chat, the application SHALL load the most recent messages (e.g., last 50-100)
- When scrolling to the top, the application SHALL load older messages automatically
- The application SHALL maintain scroll position when loading older messages
- Loading indicator SHALL be shown during history fetch

### AC-005.7: Text Content Display
- The application SHALL support UTF-8 text display including emojis
- Long messages SHALL wrap at word boundaries
- The application SHALL NOT truncate messages
- Newlines in messages SHALL be preserved

### AC-005.8: Service Messages
- System messages (user joined, left, etc.) SHALL be displayed
- Service messages SHALL be visually distinct from regular messages
- Service messages SHALL be minimal and non-intrusive

## Technical Notes
- Use TDLib's update subscription for real-time messages
- Implement efficient scrollback buffer for message history
- Use virtual scrolling for large chat histories (performance)
- Maintain message cache per chat

## Dependencies
- FR-001: User Authentication
- FR-003: Open Specific Chat via CLI

## Related Requirements
- FR-004: Send Text Messages
- FR-006: Reply to Messages
- FR-013: Markdown Rendering
- FR-015: Unread Message Indicators
- NFR-001: Performance (<50ms message display)
