# FR-004: Send Text Messages

## Description
The application must allow users to compose and send text messages in the currently open chat.

## Priority
**CRITICAL** - Core messaging functionality

## User Story
As a user, I want to type and send text messages in a chat, so that I can communicate with my contacts and groups.

## Acceptance Criteria

### AC-004.1: Input Area
- The application SHALL provide a text input area at the bottom of the chat view
- The input area SHALL support multi-line text entry
- The input area SHALL expand vertically as needed for multi-line messages (up to reasonable limit, e.g., 10 lines visible)
- The input area SHALL wrap text at word boundaries

### AC-004.2: Message Composition
- Users SHALL be able to type regular text
- Users SHALL be able to use Backspace/Delete for editing
- Users SHALL be able to navigate within the input using arrow keys
- Users SHALL be able to use Ctrl-a to jump to start of line
- Users SHALL be able to use Ctrl-e to jump to end of line
- Users SHALL be able to use Ctrl-u to clear the line
- Users SHALL be able to use Ctrl-k to delete from cursor to end of line

### AC-004.3: Send Behavior (Configurable)
- The application SHALL support configurable send keybinding via config file
- Option 1: `Enter` sends message, `Shift-Enter` inserts newline
- Option 2: `Ctrl-Enter` sends message, `Enter` inserts newline
- The default SHALL be Option 2 (`Ctrl-Enter` sends, `Enter` for newline)
- The configuration option SHALL be documented in default config

### AC-004.4: Message Sending
- When send keybinding is pressed, the application SHALL:
  - Send the message content to Telegram
  - Clear the input area
  - Show the sent message in the message list immediately (optimistic UI)
  - Display "Sending..." indicator next to the message
  - Update to "Sent" or "✓" when confirmed by server

### AC-004.5: Send Failure Handling
- IF message send fails, the application SHALL:
  - Display error indicator next to the message
  - Show error details in status bar
  - Keep message in input area for editing/retry
  - Provide keybinding to retry send (e.g., `r` when message is selected)

### AC-004.6: Empty Message Prevention
- The application SHALL NOT send empty messages
- The application SHALL NOT send messages containing only whitespace
- The application SHALL trim leading/trailing whitespace before sending

### AC-004.7: Message Queuing (Offline)
- IF network is disconnected, the application SHALL:
  - Queue outgoing messages
  - Display "Queued" status
  - Automatically send when connection is restored

### AC-004.8: Special Characters and Emojis
- The application SHALL support UTF-8 text including emojis
- The application SHALL properly display composed text in input area
- Emojis SHALL be sent as unicode characters

## Technical Notes
- Use TDLib's `sendMessage` API
- Implement optimistic UI updates for perceived performance
- Handle message IDs for tracking send status
- Queue mechanism for offline support

## Dependencies
- FR-001: User Authentication
- FR-003: Open Specific Chat via CLI (to have a chat open)
- FR-011: Configuration Management (for send keybinding config)

## Related Requirements
- FR-005: Receive and Display Messages
- FR-017: Network Reconnection (for queued message sending)
- NFR-001: Performance (instant UI feedback)
