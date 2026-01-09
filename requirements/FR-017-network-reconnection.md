# FR-017: Network Reconnection

## Description
The application must automatically handle network disconnections by attempting to reconnect in the background while keeping the UI responsive and informing the user of connection status.

## Priority
**HIGH** - Critical for reliability and user experience

## User Story
As a user, when my network connection is lost or unstable, I want the application to automatically reconnect without crashing or requiring a restart, so that I can continue working seamlessly once connectivity is restored.

## Acceptance Criteria

### AC-017.1: Connection State Detection
- The application SHALL detect when connection to Telegram is lost
- Detection methods:
  - TDLib connection state updates
  - Network error responses
  - Timeout on operations
- The application SHALL distinguish between:
  - Temporary network issues (reconnect automatically)
  - Authentication issues (show error, may require user action)
  - Permanent errors (show error, may exit)

### AC-017.2: Automatic Reconnection
- When connection is lost, the application SHALL:
  - Update status bar to "Reconnecting..."
  - Begin reconnection attempts in the background
  - NOT block the UI (user can still navigate, view cached messages)
  - NOT crash or exit

### AC-017.3: Reconnection Strategy
- Use exponential backoff for reconnection attempts:
  - 1st attempt: Immediate (0 seconds)
  - 2nd attempt: 2 seconds after failure
  - 3rd attempt: 4 seconds after failure
  - 4th attempt: 8 seconds after failure
  - Max delay: 30 seconds
  - Continue attempting indefinitely (or configurable max attempts)
- Configurable via `network.reconnect_delay` in config (base delay)

### AC-017.4: Status Updates
- During reconnection, the status bar SHALL show:
  - "Reconnecting..." (with optional attempt count)
  - Example: "Reconnecting... (attempt 3)"
- When reconnected, the status bar SHALL show:
  - "Connected" (normal color)
  - Brief success message: "Reconnected!" (temporary, 2 seconds)

### AC-017.5: Offline Functionality
- While offline, the application SHALL:
  - Allow viewing cached messages
  - Allow scrolling through message history (cached portion)
  - Allow composing messages (they will be queued)
  - Allow navigation (switching views, chat switcher)
  - Disable operations that require network:
    - Sending messages (queued instead)
    - Downloading files (show error)
    - Loading new message history (show error)

### AC-017.6: Message Queuing
- Messages composed while offline SHALL be queued
- Queued messages SHALL:
  - Be displayed in chat with "Queued" status indicator
  - Be stored persistently (survive app restart)
  - Automatically send when connection is restored
  - Maintain original order
- IF sending fails after reconnect, retry with backoff

### AC-017.7: Reconnection Success
- When connection is restored, the application SHALL:
  - Update status bar to "Connected"
  - Send all queued messages automatically
  - Update message status from "Queued" to "Sending..." to "Sent"
  - Fetch any new messages received while offline
  - Update unread counts

### AC-017.8: User Notification
- When reconnected after being offline, the application MAY:
  - Display temporary status bar message: "Reconnected! Syncing..."
  - Show count of queued messages being sent
  - Clear notification after sync complete

### AC-017.9: Persistent Connection Issues
- IF reconnection fails repeatedly (e.g., 20 attempts over 10 minutes):
  - Continue attempting (don't give up)
  - Display in status bar: "Offline - Retrying..."
  - User can force retry with a keybinding (optional: `Ctrl-r` to retry now)
  - User can quit application normally (`:q` still works)

### AC-017.10: Manual Reconnection
- Optional: Provide manual reconnection trigger
  - Keybinding: `Ctrl-r` in NORMAL mode
  - Command: `:reconnect`
  - Action: Reset backoff, attempt immediate reconnection

### AC-017.11: Graceful Degradation
- The application SHALL remain usable during network issues:
  - No crashes or panics
  - UI remains responsive
  - Clear feedback about what's happening
  - Cached data remains accessible

### AC-017.12: Configuration
- Reconnection behavior SHALL be configurable:
  - `network.auto_reconnect = true/false` (default: true)
  - `network.reconnect_delay = N` (base delay in seconds, default: 2)
  - IF `auto_reconnect = false`, show error and exit on disconnect

## Connection State Flow
```
[Connected]
    ↓ (network lost)
[Disconnected]
    ↓ (automatic)
[Reconnecting] ← retry with backoff
    ↓ (success)
[Connected] → send queued messages, sync new messages
    OR ↓ (failure)
[Reconnecting] ← retry again
```

## Technical Notes
- Use TDLib's connection state updates: `updateConnectionState`
- Implement exponential backoff algorithm
- Store queued messages in local database (SQLite)
- Handle concurrent operations safely (mutex for queue access)
- Use goroutines for background reconnection (don't block main thread)
- Test with simulated network failures (unplug ethernet, disable wifi)

## Dependencies
- FR-001: User Authentication (may need to re-auth after reconnect)
- FR-004: Send Text Messages (for message queuing)
- FR-016: Status Bar Display (shows connection status)

## Related Requirements
- NFR-001: Performance (reconnection doesn't block UI)
- NFR-002: Reliability (graceful error handling)
- NFR-004: Usability (clear feedback, remains usable)
