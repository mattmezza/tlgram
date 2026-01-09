# NFR-003: Security

## Description
The application must implement proper authentication, encryption, and secure data handling to protect user privacy and account security.

## Priority
**CRITICAL** - User explicitly stated requirement for "proper auth and encryption"

## Security Requirements

### AC-003.1: Telegram Authentication
- Use Telegram's official authentication protocol
- Support standard auth flow:
  - Phone number + SMS code
  - Two-factor authentication (2FA) if enabled
- Never store password or 2FA password in any form
- Use TDLib's built-in auth mechanism (secure by design)

### AC-003.2: Session Data Protection
- Session files SHALL be stored with restrictive permissions:
  - **Unix/Linux**: Mode 0600 (readable/writable by owner only)
  - **macOS**: Mode 0600
  - **Windows**: Appropriate ACLs
- Session directory SHALL be mode 0700 (accessible by owner only)
- Check and fix permissions on startup if incorrect

### AC-003.3: Encryption
- All network communication SHALL use TDLib's encrypted MTProto protocol
- End-to-end encryption for secret chats (TDLib handles this)
- Session data SHALL be encrypted by TDLib
- No custom crypto implementation (rely on TDLib)

### AC-003.4: Credential Handling
- NEVER log sensitive data:
  - Phone numbers (can log masked: +1234***890)
  - Auth codes
  - 2FA passwords
  - Session keys
  - API keys (if any)
- Sanitize logs to prevent credential leakage

### AC-003.5: Configuration Security
- API ID and API Hash (required for TDLib):
  - If included in binary: Not a secret (public knowledge for custom clients)
  - If user-provided: Load from config or env vars
  - Don't log API credentials
- No hardcoded secrets in source code
- Don't commit session files or config with sensitive data to git

### AC-003.6: Local Data Storage
- Queued messages (local database):
  - Store with restrictive file permissions (0600)
  - Encrypt if storing on shared systems (optional for v1)
- Downloaded files:
  - Store in user-specified location
  - Respect umask for permissions
- Logs:
  - May contain usernames/chat names (not sensitive)
  - Should NOT contain message content or auth data
  - Restrictive permissions (0600) for logs

### AC-003.7: Clipboard Security
- When copying messages to clipboard:
  - Be aware clipboard is accessible by other applications
  - No additional security needed (user action)
- Don't automatically copy sensitive data

### AC-003.8: Terminal History
- Passwords/codes entered in terminal:
  - Use TDLib's input methods (not command-line args)
  - Don't require sensitive input via CLI args
  - Prevent shell history recording (use `read -s` equivalent)

### AC-003.9: Memory Security
- Don't keep sensitive data in memory longer than necessary
- Overwrite sensitive buffers when done (best effort)
- Go's garbage collector limitations: Can't guarantee zeroing

### AC-003.10: Dependency Security
- Use official TDLib from Telegram
- Keep TDLib updated for security patches
- Vet third-party Go dependencies
- Use Go modules for dependency pinning
- Regular security audits of dependencies

### AC-003.11: Code Injection Prevention
- Validate and sanitize all user input:
  - Config file values
  - CLI arguments
  - Terminal input
- Prevent command injection (e.g., in shell-out for clipboard)
- Escape special characters appropriately

### AC-003.12: Information Disclosure
- Error messages SHALL NOT leak sensitive information:
  - Don't reveal file paths unnecessarily
  - Don't reveal internal state details
  - Be helpful but not revealing
- Logs SHALL NOT contain message content (just metadata)

### AC-003.13: Shared System Security
- On shared systems (multi-user servers):
  - Session files protected by file permissions
  - Config protected by file permissions
  - Other users cannot access Telegram session
- Warn if running on system with weak permissions (optional)

### AC-003.14: Build Security
- Binary should be built with:
  - Latest stable Go compiler
  - All security features enabled
  - No debug symbols in release builds (optional)
- Reproducible builds (nice to have)
- Sign releases (future)

## Threat Model

### Threats Considered:
1. **Local attacker with user privileges**: Mitigated by file permissions
2. **Network eavesdropping**: Mitigated by TDLib's encryption
3. **Malicious Telegram servers**: Out of scope (trust Telegram)
4. **Dependency vulnerabilities**: Mitigated by vetting dependencies
5. **Clipboard snooping**: Accepted risk (user action)

### Threats NOT Considered (Out of Scope):
1. **Root/Administrator attacker**: Cannot defend against root access
2. **Memory forensics**: Go doesn't support secure memory zeroing
3. **Advanced persistent threats**: Not the target user scenario
4. **Physical access to unlocked system**: User responsibility

## Security Best Practices

### File Permissions Check:
```go
// On startup, verify session directory permissions
sessionDir := "~/.config/tlgram/session"
info, err := os.Stat(sessionDir)
if info.Mode().Perm() != 0700 {
    // Fix permissions
    os.Chmod(sessionDir, 0700)
}
```

### Sanitized Logging:
```go
// Good: Mask phone number
log.Printf("Authenticated as +1234***890")

// Bad: Don't log full phone
// log.Printf("Authenticated as +1234567890")

// Never log:
// log.Printf("Auth code: %s", code)
```

### Secure Input:
```
// For 2FA password, use terminal with echo disabled
// Similar to: read -s PASSWORD
```

## Compliance

### Telegram API Terms:
- Comply with Telegram API terms of service
- Use official API properly
- Don't abuse rate limits
- Don't scrape data
- Respect user privacy

## Security Testing

### Tests:
- Verify file permissions are correct
- Test with invalid/malicious input
- Review logs for sensitive data leakage
- Static analysis for common vulnerabilities
- Dependency vulnerability scanning

## Technical Notes
- TDLib handles most security concerns (encryption, auth, protocol)
- Focus on secure local data handling
- Use Go's standard library for file operations (security-reviewed)
- Regular security updates for dependencies

## Dependencies
- FR-001: User Authentication (security critical)
- NFR-002: Reliability (security through error handling)

## Related Requirements
- All FRs (security is cross-cutting concern)
