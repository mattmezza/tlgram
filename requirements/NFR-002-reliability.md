# NFR-002: Reliability

## Description
The application must be stable, handle errors gracefully, and continue operating in the face of network issues, invalid input, and unexpected conditions.

## Priority
**HIGH** - Essential for production use

## Reliability Requirements

### AC-002.1: No Crashes
- The application SHALL NOT crash or panic under normal operation
- The application SHALL handle all errors gracefully
- Use Go's error handling, avoid uncaught panics
- Recover from panics at application boundary if they occur

### AC-002.2: Error Handling Strategy
- All errors SHALL be handled explicitly
- Errors SHALL be logged to log file with context
- User-facing errors SHALL be displayed in status bar
- Critical errors MAY cause graceful shutdown (not crash)

### AC-002.3: Input Validation
- Validate all user input:
  - Configuration file values
  - CLI arguments
  - Terminal input
- Reject invalid input with clear error messages
- Don't crash on malformed input

### AC-002.4: Network Resilience
- Handle network failures gracefully (see FR-017)
- No crashes on connection loss
- Queue operations that fail due to network issues
- Retry with exponential backoff

### AC-002.5: Corrupt Data Handling
- IF configuration file is corrupted:
  - Show error with line number
  - Fall back to defaults where possible
  - Don't crash
- IF session data is corrupted:
  - Prompt for re-authentication
  - Don't crash or enter infinite loop

### AC-002.6: Resource Exhaustion
- Handle resource exhaustion gracefully:
  - **Out of memory**: Limit cache size, clear old data
  - **Out of disk space**: Warn user, prevent writes, don't crash
  - **Too many open files**: Close unused files, warn user
- Don't crash or hang

### AC-002.7: TDLib Error Handling
- Handle all TDLib errors:
  - Auth errors: Prompt for re-authentication
  - Network errors: Trigger reconnection
  - Rate limit errors: Back off and retry
  - Unknown errors: Log and display to user
- Don't crash on TDLib errors

### AC-002.8: Concurrent Access Safety
- Protect shared data with mutexes/channels
- Avoid race conditions (test with `go test -race`)
- Handle multiple goroutines safely
- No deadlocks

### AC-002.9: Logging
- Comprehensive logging to file:
  - Error details with stack traces
  - Network events
  - Auth events
  - Performance metrics (optional)
- Log levels: DEBUG, INFO, WARN, ERROR
- Configurable log level (see FR-011)
- Log rotation to prevent unbounded growth

### AC-002.10: Log File Management
- Default location: `~/.config/tlgram/logs/app.log`
- Rotate when size exceeds threshold (e.g., 10 MB)
- Keep N old log files (e.g., 3)
- Delete oldest when limit exceeded
- Ensure log directory is writable (create if needed)

### AC-002.11: Graceful Shutdown
- Handle interrupt signals (SIGINT, SIGTERM):
  - Save state (queued messages, read positions)
  - Close TDLib session gracefully
  - Flush logs
  - Clean up resources
  - Exit with code 0
- User-initiated quit (`:q`) follows same process

### AC-002.12: Data Integrity
- Queued messages SHALL persist across restarts
- Unread state SHALL persist across restarts
- No data loss on crash (best effort)
- Use atomic writes for important data (e.g., SQLite transactions)

### AC-002.13: Edge Case Handling
Handle edge cases without crashing:
- Empty chat list (new account)
- Chat with no messages
- Very long messages (10,000+ characters)
- Rapid message flood (100+ messages/second)
- Username that doesn't exist
- Chat ID that doesn't exist
- Terminal resize during operation
- Terminal width < 40 columns (minimum width)

### AC-002.14: Terminal Compatibility
- Handle terminals with limited capabilities:
  - No color support: Fall back to plain text
  - No Unicode support: Fall back to ASCII
  - Small terminal size: Adjust layout, minimum 40x10
- Detect terminal capabilities and adapt

### AC-002.15: Recovery from Errors
- Application SHOULD recover from transient errors:
  - Temporary network failures
  - Rate limiting
  - Temporary file locks
- Retry operations with backoff
- Don't exit unless error is permanent

### AC-002.16: Testing
- Unit tests for critical components
- Integration tests with TDLib
- Error injection tests (simulate failures)
- Race condition detection (`go test -race`)
- Stress tests (long-running, high load)

## Error Handling Examples

### Network Error:
```
[TUI Status Bar] Failed to send message - Reconnecting...
[Log File] ERROR: Failed to send message: network unreachable, queuing for retry
```

### Invalid Config:
```
[Terminal Output]
Error: Invalid configuration at config.toml:15
  send_key = "invalid_value"
Expected "enter" or "ctrl-enter"

Falling back to default: "ctrl-enter"
```

### Chat Not Found:
```
[Terminal Output]
Error: Chat not found: @nonexistent_user
Please check the username and try again.

Exit code: 2
```

## Logging Format

### Log Entry Structure:
```
2026-01-09 14:23:45.123 [INFO] Application started
2026-01-09 14:23:45.456 [INFO] Config loaded from ~/.config/tlgram/config.toml
2026-01-09 14:23:46.789 [INFO] Authenticated as +1234567890
2026-01-09 14:25:10.123 [ERROR] Failed to send message: context deadline exceeded
2026-01-09 14:25:10.124 [INFO] Message queued for retry
2026-01-09 14:25:12.456 [INFO] Connection restored
2026-01-09 14:25:12.789 [INFO] Queued message sent successfully
```

## Technical Notes
- Use Go's standard `log` package or structured logging (e.g., `logrus`, `zap`)
- Implement panic recovery at main application level
- Use context.Context for cancellation and timeouts
- Test error paths explicitly
- Use Go's built-in error wrapping: `fmt.Errorf("context: %w", err)`

## Dependencies
- FR-017: Network Reconnection (network error handling)
- FR-011: Configuration Management (config validation)

## Related Requirements
- NFR-001: Performance (error handling shouldn't impact performance)
- NFR-003: Security (secure error messages, don't leak sensitive info)
