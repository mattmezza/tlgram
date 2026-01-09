# FR-016: Status Bar Display

## Description
The application must display a status bar showing the current chat name and connection status.

## Priority
**MEDIUM** - Important for user awareness

## User Story
As a user, I want to see a status bar that shows which chat I'm in and whether I'm connected to Telegram, so that I always have context and know if there are connectivity issues.

## Acceptance Criteria

### AC-016.1: Status Bar Position
- The status bar SHALL be positioned at the top or bottom of the screen
- Recommended: Top of screen (below is input area in chat view)
- The status bar SHALL be always visible
- Height: Single line

### AC-016.2: Status Bar Content - Chat View
When in chat view, the status bar SHALL display:
- **Left side**: Current chat name
  - Format: `Chat: @john_doe` or `Chat: Project Alpha`
  - Use actual chat name (resolved from username/ID/alias)
- **Right side**: Connection status
  - `Connected` - Normal state, connected to Telegram
  - `Reconnecting...` - Lost connection, attempting to reconnect
  - `Offline` - Not connected, not attempting to reconnect

### AC-016.3: Status Bar Content - Chat List View
When in chat list view, the status bar SHALL display:
- **Left side**: `Chats` or `Chat List` or app name
- **Right side**: Connection status (same as above)

### AC-016.4: Status Bar Content - Other Views
When in other views (chat switcher, search mode, command mode):
- **Left side**: View name (e.g., `Chat Switcher`, `Search Mode`, `Command Mode`)
- **Right side**: Connection status

### AC-016.5: Connection Status Indicators
- **Connected**: Normal color (e.g., white/green)
- **Reconnecting...**: Warning color (e.g., yellow/orange)
- **Offline**: Error color (e.g., red)
- The status SHALL update in real-time as connection state changes

### AC-016.6: Status Bar Styling
- The status bar SHALL be visually distinct from content:
  - Different background color (e.g., inverted colors)
  - OR border/separator line
- Text SHALL be clearly readable
- Use terminal attributes for styling (bold, colors)

### AC-016.7: Temporary Messages
- The status bar SHALL support temporary messages/notifications:
  - "Copied!" (after copying message)
  - "Downloaded to: /path/to/file" (after download)
  - "Sending..." (while sending message)
  - Error messages (e.g., "Failed to send message")
- Temporary messages SHALL:
  - Replace connection status temporarily
  - Auto-clear after 2-3 seconds
  - Return to showing connection status after clear

### AC-016.8: Mode Indicator (Optional)
- Optionally, the status bar MAY include the current mode:
  - `NORMAL`, `INSERT`, `SEARCH`, `COMMAND`
- Position: Left side, before chat name
  - Format: `[NORMAL] Chat: @john_doe    Connected`
- This helps users know which keybindings are active

### AC-016.9: Truncation Handling
- IF chat name is too long for available space:
  - Truncate with ellipsis: `Chat: Very Long Group Name Th...`
  - Ensure connection status always visible (right side has priority)
- Calculate truncation based on terminal width

### AC-016.10: Real-time Updates
- Status bar SHALL update immediately (<50ms) when:
  - Switching chats
  - Connection status changes
  - Temporary message appears

## Example Displays

### Chat View - Connected:
```
╔════════════════════════════════════════════════════════╗
║ Chat: @john_doe                         Connected     ║
╚════════════════════════════════════════════════════════╝
```

### Chat View - Reconnecting:
```
╔════════════════════════════════════════════════════════╗
║ Chat: Project Alpha                  Reconnecting... ║
╚════════════════════════════════════════════════════════╝
```

### Chat View - With Mode Indicator:
```
╔════════════════════════════════════════════════════════╗
║ [NORMAL] Chat: @john_doe                Connected     ║
╚════════════════════════════════════════════════════════╝
```

### Temporary Message:
```
╔════════════════════════════════════════════════════════╗
║ Chat: @john_doe                           Copied!     ║
╚════════════════════════════════════════════════════════╝
(After 2 seconds, "Copied!" is replaced with "Connected")
```

### Chat List View:
```
╔════════════════════════════════════════════════════════╗
║ Chats                                      Connected   ║
╚════════════════════════════════════════════════════════╝
```

## Technical Notes
- Use terminal width to calculate available space
- Implement a status bar component that can be updated independently
- Use ANSI color codes for connection status coloring
- Implement a timer for temporary message auto-clear
- Consider using a Go TUI library's status bar component (e.g., from bubbletea/lipgloss)

## Dependencies
- FR-017: Network Reconnection (provides connection status)
- FR-010: Vim Keybindings (mode indicator optional)

## Related Requirements
- NFR-001: Performance (instant updates)
- NFR-004: Usability (always visible context)
