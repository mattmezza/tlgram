# FR-018: Multiple Instance Support

## Description
The application must support running multiple instances simultaneously, each with isolated UI state but sharing the same Telegram authentication session.

## Priority
**HIGH** - Critical for tmux workflow with multiple chat windows

## User Story
As a user, I want to run multiple instances of the application in different tmux panes or windows, each showing a different chat, so that I can monitor and interact with multiple conversations simultaneously without switching between chats.

## Acceptance Criteria

### AC-018.1: Multiple Instances Allowed
- The application SHALL allow multiple instances to run concurrently
- No instance limit (user can run as many as needed)
- Each instance SHALL be a separate process
- Instances SHALL NOT conflict or crash each other

### AC-018.2: Shared Authentication Session
- All instances SHALL share the same Telegram authentication session
- Session data location: `~/.config/tlgram/session/`
- Only one authentication needed (first instance authenticates, others reuse)
- IF one instance authenticates, all instances become authenticated
- Session SHALL be safely shared (concurrent access handled)

### AC-018.3: Isolated UI State
Each instance SHALL maintain independent:
- Current chat view (different chats in different instances)
- Cursor position and scroll position
- Input buffer (composing different messages)
- Mode state (one in INSERT, another in NORMAL, etc.)
- Search state (one searching, another not)
- Selected message

### AC-018.4: Shared Data Updates
All instances SHALL receive and display:
- New messages in real-time (if chat is open in an instance)
- Unread count updates (chat list reflects new messages)
- Connection status updates (all instances show same connection state)
- Typing indicators (if supported)

### AC-018.5: Instance Startup
- Each instance can be started with `--chat` argument:
  - `tmux new-window "tlgram --chat work"`
  - `tmux split-window "tlgram --chat project"`
- Each instance opens its specified chat independently
- Instances can be started without `--chat` (shows chat list)

### AC-018.6: Message Synchronization
- When a message is sent from one instance:
  - All other instances viewing the same chat SHALL show the new message immediately
  - The sending instance shows optimistic UI update
  - Other instances receive update via TDLib
- Message status updates (sent, read) SHALL sync across instances

### AC-018.7: Read State Synchronization
- When a user marks messages as read in one instance:
  - Other instances viewing the same chat SHALL update read indicators
  - Unread counts in chat lists SHALL update across all instances
  - Unread separators SHALL update accordingly

### AC-018.8: No Instance Interference
- Instances SHALL NOT:
  - Steal focus from each other
  - Block each other's operations
  - Corrupt shared data
  - Cause race conditions or deadlocks

### AC-018.9: Graceful Session Sharing
- TDLib session SHALL be shared safely:
  - Use TDLib's built-in multi-instance support (if available)
  - OR use file locking for session access
  - OR use separate TDLib instances sharing same session files
- Handle edge case: Two instances starting simultaneously

### AC-018.10: Instance Independence
- IF one instance crashes:
  - Other instances SHALL continue running normally
  - Crashed instance SHALL NOT corrupt shared session
  - User can restart crashed instance

### AC-018.11: Performance
- Multiple instances SHALL NOT significantly impact performance
- Each instance SHALL maintain <50ms UI responsiveness
- Total resource usage SHALL scale linearly (not exponentially) with instance count
- Acceptable: 3-5 instances running without noticeable slowdown

### AC-018.12: Configuration
- All instances SHALL read from the same config file
- Config changes require restart of all instances (no live reload)
- Each instance uses the same chat aliases, keybindings, etc.

## Example Tmux Layout
```
┌─────────────────────────────────────────┐
│ tmux window 1: Work Project             │
│ ┌─────────────┬─────────────────────┐   │
│ │ tlgram      │ tlgram              │   │
│ │ --chat work │ --chat project      │   │
│ │             │                     │   │
│ │ Chat with   │ Project group chat  │   │
│ │ @john_doe   │ with team           │   │
│ └─────────────┴─────────────────────┘   │
└─────────────────────────────────────────┘
```

Both instances:
- Use same authentication
- Show different chats
- Update independently
- Receive messages in real-time

## Technical Notes

### TDLib Multi-Instance Approach:
- **Option 1**: Single TDLib instance shared by all processes (complex, not recommended)
- **Option 2**: Each process has its own TDLib instance, all pointing to same session directory (recommended)
  - TDLib handles concurrent session access internally
  - Each process maintains its own connection and state
  - Session data is shared via filesystem
- **Option 3**: Use TDLib's database encryption and multi-client support

### Recommended Implementation:
- Each application instance creates its own TDLib client instance
- All instances use same session directory: `~/.config/tlgram/session/`
- TDLib's session management handles concurrent access
- Test with 2-3 instances to verify no corruption

### Testing:
- Start 2-3 instances in separate terminals
- Open different chats in each
- Send messages from each instance
- Verify real-time updates across instances
- Test with same chat open in multiple instances
- Test instance crashes and recovery

## Dependencies
- FR-001: User Authentication (session sharing)
- FR-003: Open Specific Chat via CLI (each instance can open different chat)

## Related Requirements
- NFR-001: Performance (multiple instances remain responsive)
- NFR-002: Reliability (instance isolation prevents cascading failures)
