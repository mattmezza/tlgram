# NFR-004: Usability

## Description
The application must provide an intuitive, efficient, and pleasant user experience that aligns with vim workflows and terminal usage patterns.

## Priority
**HIGH** - Core to user satisfaction and adoption

## Usability Requirements

### AC-004.1: Minimal UI Philosophy
- The interface SHALL be clean and uncluttered
- Focus on content (messages), not chrome
- Single-pane design (not overwhelming)
- No unnecessary borders, decorations, or visual noise
- Information density: High but readable

### AC-004.2: Vim Muscle Memory
- Keybindings SHALL follow vim conventions (see FR-010)
- Navigation SHALL feel natural to vim users:
  - hjkl navigation works everywhere
  - gg/G for jump to top/bottom
  - / for search
  - : for commands
- Modal interface (NORMAL, INSERT, SEARCH, COMMAND)
- Clear mode indicators

### AC-004.3: Discoverability
- Essential keybindings SHALL be discoverable:
  - Help command: `:help` or `?`
  - Status bar shows current mode
  - Error messages guide user to correct usage
- Common actions should be intuitive (even without reading docs)

### AC-004.4: Consistency
- Keybindings SHALL be consistent across views
- Visual design SHALL be consistent throughout
- Error messages SHALL follow consistent format
- Terminology SHALL be consistent (chat vs conversation, message vs text)

### AC-004.5: Feedback and Visibility
- User actions SHALL provide immediate feedback:
  - Keystrokes update UI instantly
  - Operations show progress (e.g., "Sending...", "Downloading 45%")
  - Errors display clearly in status bar
  - Success confirmations (e.g., "Copied!", "Sent!")
- System status SHALL always be visible:
  - Connection status in status bar
  - Current chat name in status bar
  - Mode indicator visible

### AC-004.6: Error Messages
- Error messages SHALL be:
  - **Clear**: Explain what went wrong
  - **Actionable**: Suggest how to fix
  - **Non-technical**: Avoid jargon where possible
- Examples:
  - Good: "Chat not found: @john_doe. Check the username and try again."
  - Bad: "Error 404: Chat entity not found in database."

### AC-004.7: Graceful Degradation
- Application SHALL work in constrained environments:
  - Small terminals (minimum 40x10)
  - Basic terminal capabilities (no truecolor)
  - Slow SSH connections
- Adapt UI to terminal capabilities (detect and adjust)

### AC-004.8: Efficient Workflows
- Common tasks SHALL require minimal keystrokes:
  - Open chat: `tlgram --chat work` (via alias)
  - Switch chat: `Ctrl-p` → type → `Enter`
  - Send message: Type → `Ctrl-Enter`
  - Reply: `r` → type → `Ctrl-Enter`
  - Copy: `y`
- No unnecessary confirmations or dialogs

### AC-004.9: Cognitive Load
- Don't overwhelm user with information:
  - Show what's needed, hide what's not
  - Timestamps on hover/select (not always visible)
  - Status bar is concise (not cluttered)
- Progressive disclosure: More details on demand

### AC-004.10: Predictability
- Application SHALL behave predictably:
  - Same action always produces same result
  - No surprising side effects
  - Keybindings don't change based on context (unless modal)
- Follows principle of least surprise

### AC-004.11: Recovery from Mistakes
- User SHALL be able to recover from mistakes:
  - Edit messages after sending (if Telegram supports)
  - Delete messages (if Telegram supports)
  - Cancel actions with Escape
  - Undo would be nice but complex (v2 feature)

### AC-004.12: Documentation
- Provide clear documentation:
  - README with quick start
  - Config file well-commented with examples
  - `--help` output comprehensive
  - In-app help (`:help`) for keybindings
- Examples for common use cases

### AC-004.13: First-Run Experience
- First-time users SHALL have smooth onboarding:
  - Prompt for phone number clearly
  - Explain what's happening during auth
  - Create default config with comments
  - Guide to basic usage after first auth

### AC-004.14: Accessibility
- Use standard terminal conventions:
  - Work with screen readers (basic support)
  - Support terminal zoom
  - Respect terminal color preferences
- High contrast mode (future)

### AC-004.15: Tmux Integration
- Work seamlessly in tmux:
  - Clipboard integration works
  - Mouse support optional (keyboard is primary)
  - Terminal size detection correct
  - Window titles update (optional)

## Usability Testing

### User Testing Scenarios:
1. **New user**: Install, authenticate, send first message (< 5 minutes)
2. **Hop integration**: Create hop template with chat aliases (< 10 minutes)
3. **Multi-chat workflow**: Run 3 instances, switch between them, send messages
4. **Search and copy**: Find old message, copy to clipboard
5. **Network failure**: Disconnect network, compose message, reconnect, verify sent

### Metrics:
- Time to first message (new user)
- Time to complete common tasks
- Error rate (user mistakes)
- User satisfaction (subjective feedback)

## Design Principles

### 1. Minimize Friction
- Reduce steps to accomplish tasks
- No unnecessary confirmation dialogs
- Fast shortcuts for common actions

### 2. Respect User's Time
- Fast performance (see NFR-001)
- Efficient keybindings
- No waiting for UI

### 3. Stay Out of the Way
- Minimal UI
- No distractions
- Focus on conversation

### 4. Be Forgiving
- Handle errors gracefully
- Provide clear error messages
- Allow recovery from mistakes

### 5. Be Consistent
- Follow conventions (vim, terminal)
- Predictable behavior
- Consistent visual design

## Example Good UX Flows

### Sending a Message:
```
1. User opens chat: tlgram --chat work
2. User presses 'i' to enter insert mode
   [Status bar shows: INSERT mode]
3. User types message
   [Message appears in input area in real-time]
4. User presses Ctrl-Enter
   [Message appears in chat with "Sending..." indicator]
5. After 200ms, indicator changes to "Sent ✓"
   [User can continue typing immediately]
```

### Handling an Error:
```
1. User tries to open non-existent chat: tlgram --chat @invalid
2. Application displays:
   "Error: Chat not found: @invalid
    Please check the username and try again.

    Tip: Use 'tlgram' without --chat to see your chat list."
3. Application exits with code 2
4. User runs: tlgram
5. Chat list appears, user can select correct chat
```

## Technical Notes
- Use established TUI libraries (e.g., bubbletea) for good UX
- Test with real users (even if just yourself)
- Iterate based on feedback
- Follow terminal UX best practices

## Dependencies
- FR-010: Vim Keybindings (core usability feature)
- All FRs (usability affects all features)

## Related Requirements
- NFR-001: Performance (snappiness is part of good UX)
- NFR-006: Compatibility (works in various environments)
