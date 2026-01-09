# FR-014: Copy Message Text to Clipboard

## Description
The application must allow users to copy selected message text to the system clipboard using a vim-style keybinding.

## Priority
**MEDIUM** - Useful for workflow integration

## User Story
As a user, I want to copy message text to my clipboard with a simple keybinding, so that I can paste it into other applications or terminal commands.

## Acceptance Criteria

### AC-014.1: Copy Keybinding
- Pressing `y` (yank) in NORMAL mode on a selected message SHALL copy the message text to clipboard
- The keybinding SHALL follow vim convention (y = yank)
- SHALL work in chat view when a message is selected/focused

### AC-014.2: Copy Message Content
- The application SHALL copy the plain text content of the selected message
- Formatting SHALL be stripped (no markdown symbols, just the text)
- Newlines SHALL be preserved in multi-line messages
- Emojis SHALL be copied as unicode characters

### AC-014.3: Clipboard Integration
- The application SHALL use the system clipboard:
  - **Primary clipboard** (middle-click paste) on Linux/Unix
  - **System clipboard** (Ctrl-V paste) on all platforms
- Both clipboards should be populated on Linux for maximum compatibility
- Work correctly in tmux sessions

### AC-014.4: Visual Feedback
- After successful copy, the application SHALL:
  - Display "Copied!" or "Yanked!" message in status bar
  - Show feedback for ~1 second, then clear
  - NOT interrupt user workflow (no modal dialog)

### AC-014.5: Copy Failure Handling
- IF clipboard access fails, the application SHALL:
  - Display error in status bar: "Failed to copy to clipboard"
  - Log the error to log file
  - Continue operating normally

### AC-014.6: Copy Media Message Info
- IF selected message is a media message (photo, video, file):
  - Copy the caption text if present
  - IF no caption, copy the filename
  - IF no filename, copy descriptive text like "[Image - 1.2 MB]"

### AC-014.7: Copy Reply Context
- IF selected message is a reply:
  - Copy only the reply message text (not the original message)
  - Do NOT include reply context in clipboard

### AC-014.8: Copy System Messages
- IF selected message is a system message (user joined, etc.):
  - Copy the system message text as displayed
  - Example: "John Doe joined the group"

### AC-014.9: Tmux Integration
- The application SHALL work correctly within tmux:
  - Detect if running in tmux
  - Use tmux's clipboard if available (`tmux set-buffer`)
  - Fall back to system clipboard methods
  - Ensure OSC 52 escape sequences work through tmux (for remote sessions)

### AC-014.10: Remote Session Support
- For SSH sessions, use OSC 52 escape sequence to copy to local clipboard
- This enables clipboard functionality even when SSH'd into a remote machine
- Should be configurable (some terminals don't support OSC 52)

## Technical Implementation Options

### Clipboard Access Methods (in priority order):
1. **OSC 52 escape sequence** (works in tmux, SSH, most modern terminals)
2. **tmux set-buffer** (when running in tmux)
3. **xclip / xsel** (Linux/Unix with X11)
4. **wl-copy** (Linux/Unix with Wayland)
5. **pbcopy** (macOS)
6. **clip.exe** (Windows)

### Detection Logic:
- Check if running in tmux: `$TMUX` environment variable
- Detect terminal capabilities for OSC 52
- Fall back through methods until one succeeds

## Example Usage

### User Action:
1. Navigate to message with `j`/`k`
2. Press `y` to copy
3. See status bar: "Copied!"
4. Switch to terminal, paste with Ctrl-Shift-V or middle-click

### Example Message:
```
@john_doe
Check this API endpoint: https://api.example.com/v1/users
```

### After pressing `y`:
- Clipboard contains: `Check this API endpoint: https://api.example.com/v1/users`
- Status bar shows: `Copied!`

## Technical Notes
- Use Go library for clipboard access (e.g., `github.com/atotto/clipboard`)
- For OSC 52, write escape sequence directly to terminal
- Test in multiple environments: native terminal, tmux, SSH
- Respect security settings (some terminals block clipboard for security)

## Dependencies
- FR-005: Receive and Display Messages (to have messages to copy)
- FR-010: Vim Keybindings (for 'y' keybinding)

## Related Requirements
- NFR-004: Usability (seamless clipboard integration)
- NFR-006: Compatibility (tmux, SSH support)
