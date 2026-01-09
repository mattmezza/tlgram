# FR-015: Unread Message Indicators

## Description
The application must clearly indicate which chats have unread messages and visually separate unread messages within a chat, making it obvious what's new since the last time the user viewed the chat.

## Priority
**HIGH** - Critical for user awareness and workflow

## User Story
As a user, when I return to the application after being away, I want to immediately see which chats have new messages and which specific messages are new within a chat, so that I can quickly catch up on what I missed.

## Acceptance Criteria

### AC-015.1: Unread Count in Chat List
- Each chat in the chat list SHALL display an unread message count if > 0
- Display format: `Chat Name (3)` or `Chat Name [3]` or with badge
- Unread count SHALL be accurate and update in real-time
- Chats with unread messages SHALL be sorted higher in the list (optional: configurable)

### AC-015.2: Visual Distinction in Chat List
- Chats with unread messages SHALL be visually distinct:
  - **Bold** chat name
  - OR different color (e.g., bright white vs. gray)
  - OR unread indicator icon/symbol (e.g., ● or ★)
- The distinction SHALL be immediately noticeable

### AC-015.3: Unread Separator in Chat View
- When opening a chat with unread messages, the application SHALL display a visual separator
- The separator SHALL appear between:
  - Last message read by user
  - First unread message
- Separator format: horizontal line with label
  - Example: `───────── Unread Messages ─────────`
  - Example: `━━━━━━━━━ 5 New Messages ━━━━━━━━━`
  - Example: `========== NEW ==========`

### AC-015.4: Separator Placement
- The separator SHALL be positioned correctly:
  - IF user has read up to message N, separator appears after message N
  - Unread messages (N+1, N+2, ...) appear below the separator
- The separator SHALL scroll with messages (it's part of the message list)

### AC-015.5: Separator Persistence
- The unread separator SHALL remain visible until:
  - User scrolls past it (moves focus beyond the separator)
  - User sends a message (implies chat is focused and read)
  - User explicitly marks chat as read (if such feature exists)
- After removal, opening the chat again should show no separator (all caught up)

### AC-015.6: New Message Since Last Focus
When the user has been away for hours and returns:
- The separator clearly shows: "This is where you left off"
- All messages below the separator are what you haven't seen yet
- This is the critical feature requested by user

### AC-015.7: Real-time Unread Updates
- As new messages arrive in other chats (not currently focused):
  - Unread count SHALL increment immediately
  - Chat list SHALL update in real-time
  - Visual indicators SHALL update
- For the currently open chat:
  - New messages appearing below are automatically "read" (no separator needed)
  - The separator is only for messages that arrived while chat was NOT focused

### AC-015.8: Unread State Tracking
- The application SHALL persist unread state between sessions:
  - Store last_read_message_id per chat
  - Store locally (not necessarily sync with Telegram's read state)
- On startup, the application SHALL:
  - Load unread state from storage
  - Display unread counts and separators based on stored state

### AC-015.9: Auto-Mark-as-Read Behavior
- When user opens a chat and scrolls through unread messages:
  - Messages SHALL be marked as read as user scrolls past them
  - This updates Telegram's read state (other clients will see messages as read)
  - Configurable via `auto_mark_read` in config (see FR-011)
- IF `auto_mark_read = false`:
  - Messages stay unread until manually marked
  - Allows "I'll read this later" workflow

### AC-015.10: Multiple Unread Sections (Edge Case)
- IF user leaves a chat, new messages arrive, user returns briefly, leaves again, more messages arrive:
  - Show the most recent unread separator only (simpler UX)
  - OR show multiple separators (complex but accurate)
  - Recommended: single separator for simplicity in v1

## Example Display

### Chat List with Unreads:
```
──────────────────────────
  Chats
──────────────────────────
  **project_alpha (5)**     ← bold, unread count
  work_dm (2)               ← unread indicator
  daily_standup
  random_chat               ← no unreads, normal style
──────────────────────────
```

### Chat View with Unread Separator:
```
@john_doe
Hey, did you see the update?

@matteo [You]
Not yet, checking now.

───────── 3 New Messages ─────────

@john_doe
We deployed the new feature!

@john_doe
Check staging when you have a moment.

@jane_doe
Looks good on my end 👍
```

## Technical Notes
- Store last_read_message_id per chat in local database (e.g., SQLite)
- Use TDLib's `viewMessages` API to mark messages as read
- Track focus state: which chat is currently open
- Subscribe to TDLib updates for unread count changes
- Calculate separator position based on last_read_message_id

## Dependencies
- FR-002: Chat List Display (shows unread indicators)
- FR-005: Receive and Display Messages (displays separator in chat)
- FR-011: Configuration Management (auto_mark_read setting)

## Related Requirements
- NFR-004: Usability (clear awareness of new messages)
- NFR-001: Performance (real-time updates)
