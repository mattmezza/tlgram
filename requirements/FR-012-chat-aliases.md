# FR-012: Chat Aliases

## Description
The application must support user-defined aliases for chats, enabling quick access via short, memorable names in CLI arguments and commands.

## Priority
**HIGH** - Critical for hop template integration and workflow efficiency

## User Story
As a user, I want to define short aliases for my frequently used chats, so that I can quickly open them using memorable names instead of usernames or numeric IDs.

## Acceptance Criteria

### AC-012.1: Alias Definition
- Aliases SHALL be defined in the `[chat_aliases]` section of `config.toml`
- Format: `alias_name = "chat_identifier"`
- Chat identifier can be:
  - Telegram username with @ (e.g., `"@john_doe"`)
  - Numeric chat ID (e.g., `"123456789"` for DMs, `"-1001234567890"` for groups)
- Alias names SHALL:
  - Be alphanumeric plus underscores/hyphens
  - Be case-insensitive
  - Not conflict with special commands

### AC-012.2: Example Alias Configuration
```toml
[chat_aliases]
work = "@john_doe"
project = "-1001234567890"
team = "@project_alpha_group"
boss = "987654321"
```

### AC-012.3: Using Aliases in CLI
- Aliases SHALL work with the `--chat` flag:
  - `tlgram --chat work` opens the chat defined by `work` alias
  - `tlgram --chat project` opens the chat defined by `project` alias
- The application SHALL resolve the alias to the actual chat identifier
- IF alias doesn't exist, treat it as a direct username/ID

### AC-012.4: Using Aliases in Commands
- Aliases SHALL work in command mode:
  - `:chat work` switches to the chat defined by `work` alias
- Same resolution logic applies

### AC-012.5: Alias Resolution
- On startup with `--chat <alias>`:
  1. Check if the argument matches an alias name (case-insensitive)
  2. If yes, resolve to the configured chat identifier
  3. If no, treat as direct username or chat ID
  4. Proceed with normal chat opening logic

### AC-012.6: Alias Validation
- On config load, the application SHALL validate:
  - Alias names are valid (alphanumeric, _, -)
  - Chat identifiers are well-formed (@ + username or numeric ID)
  - No duplicate alias names
  - No empty alias names or identifiers
- Invalid aliases SHALL trigger a warning but not prevent startup
- Invalid aliases SHALL be ignored (not usable)

### AC-012.7: Error Handling
- IF an alias points to a non-existent chat:
  - Show error: "Chat not found for alias 'work' (@john_doe)"
  - Exit with code 2
- IF alias definition is invalid:
  - Show warning on startup
  - List which aliases are invalid
  - Continue with valid aliases only

### AC-012.8: Alias Display
- When using an alias to open a chat, the status bar SHALL show:
  - The actual chat name (resolved)
  - Not the alias
- This prevents confusion about which chat is actually open

### AC-012.9: Hop Template Integration
Users can easily define aliases in their hop templates:
```bash
# In hop template
cat >> ~/.config/tlgram/config.toml <<EOF

[chat_aliases]
project_chat = "@team_alpha"
EOF

# Then start with alias
tlgram --chat project_chat
```

## Example Workflow

### Setup (in config.toml):
```toml
[chat_aliases]
john = "@john_doe"
team = "-1001234567890"
```

### Usage:
```bash
# Open via alias
$ tlgram --chat john
# → Opens DM with @john_doe

$ tlgram --chat team
# → Opens group -1001234567890

# Still works with direct identifiers
$ tlgram --chat @jane_doe
# → Opens DM with @jane_doe (no alias needed)

# In command mode
:chat john
# → Switches to chat defined by 'john' alias
```

## Technical Notes
- Parse aliases during config load
- Store in a map[string]string for fast lookup
- Perform case-insensitive lookup using strings.ToLower()
- Validate alias names with regex: `^[a-zA-Z0-9_-]+$`

## Dependencies
- FR-003: Open Specific Chat via CLI (alias resolution happens here)
- FR-011: Configuration Management (aliases defined in config)

## Related Requirements
- NFR-004: Usability (memorable shortcuts improve workflow)
