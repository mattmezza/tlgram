# FR-001: User Authentication

## Description
The application must support Telegram authentication with phone number, SMS code, and optional two-factor authentication (2FA).

## Priority
**CRITICAL** - Cannot function without authentication

## User Story
As a user, I want to authenticate with my Telegram account using my phone number and receive a verification code, so that I can access my chats securely.

## Acceptance Criteria

### AC-001.1: Phone Number Entry
- The application SHALL prompt for phone number in international format (e.g., +1234567890) on first launch
- The application SHALL validate phone number format before submitting
- The application SHALL display clear error messages for invalid formats

### AC-001.2: SMS Verification Code
- The application SHALL prompt for the SMS verification code sent by Telegram
- The application SHALL allow code re-send after timeout
- The application SHALL validate code format (typically 5 digits)

### AC-001.3: Two-Factor Authentication
- IF the account has 2FA enabled, the application SHALL prompt for the 2FA password
- The application SHALL handle 2FA password validation
- The application SHALL display appropriate error messages for incorrect passwords

### AC-001.4: Session Persistence
- The application SHALL store authentication session data in `~/.config/tlgram/session/`
- The application SHALL reuse existing session if valid
- The application SHALL NOT require re-authentication on subsequent launches
- Session files SHALL be stored with secure file permissions (0600)

### AC-001.5: Multiple Instance Support
- Multiple instances of the application SHALL share the same authentication session
- The application SHALL handle concurrent session access without conflicts

## Technical Notes
- Use Telegram TDLib for authentication
- Store session data using TDLib's native session management
- Ensure session directory is created with appropriate permissions

## Dependencies
- None

## Related Requirements
- NFR-003: Security (encryption and secure storage)
- FR-018: Multiple Instance Support
