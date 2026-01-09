# FR-003: Open Specific Chat via CLI

## Description
The application must support opening a specific chat directly via command-line argument using username, chat ID, or configured alias.

## Priority
**CRITICAL** - Core requirement for tmux/hop template integration

## User Story
As a user, I want to launch the application directly into a specific chat using a command-line argument, so that I can integrate it into my tmux session templates and start working immediately.

## Acceptance Criteria

### AC-003.1: Username Support
- The application SHALL accept the `--chat` flag followed by a Telegram username
- Username format SHALL be `@username` (e.g., `tlgram --chat @john_doe`)
- The application SHALL resolve the username to the corresponding chat
- The application SHALL open the chat view directly, bypassing the chat list

### AC-003.2: Chat ID Support
- The application SHALL accept the `--chat` flag followed by a numeric chat ID
- Chat ID format SHALL be a positive or negative integer (e.g., `tlgram --chat 123456789` for DMs, `tlgram --chat -1001234567890` for groups)
- The application SHALL open the chat corresponding to the ID directly

### AC-003.3: Alias Support
- The application SHALL accept the `--chat` flag followed by a configured alias
- Aliases SHALL be defined in the configuration file (see FR-012)
- Example: `tlgram --chat work` where `work` is defined as `@john_doe` in config
- The application SHALL resolve the alias to the corresponding chat identifier

### AC-003.4: Error Handling
- IF the specified chat does not exist or cannot be found, the application SHALL:
  - Display a clear error message
  - Exit with non-zero exit code (e.g., exit code 2)
- IF the username/alias is ambiguous, the application SHALL:
  - Display available matches
  - Prompt user to clarify or exit with error

### AC-003.5: Chat Not Found Fallback
- IF the chat identifier is valid but chat is not in recent list, the application SHALL:
  - Attempt to load the chat from Telegram
  - Open the chat if found
  - Display error if chat is inaccessible

### AC-003.6: Help Text
- The application SHALL document the `--chat` flag in `--help` output
- Help text SHALL include examples of all three formats (username, ID, alias)

## Example Usage
```bash
# Open by username
tlgram --chat @john_doe

# Open by chat ID (DM)
tlgram --chat 123456789

# Open by chat ID (Group)
tlgram --chat -1001234567890

# Open by alias
tlgram --chat work
```

## Technical Notes
- Parse CLI arguments before TUI initialization
- Resolve identifier to chat before rendering UI
- Maintain fast startup time even when resolving identifiers

## Dependencies
- FR-001: User Authentication (must be authenticated to access chats)
- FR-011: Configuration Management (for alias resolution)
- FR-012: Chat Aliases (defines alias format)

## Related Requirements
- FR-018: Multiple Instance Support (multiple instances with different --chat args)
- NFR-001: Performance (fast startup even with identifier resolution)
