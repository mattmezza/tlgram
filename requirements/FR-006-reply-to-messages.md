# FR-006: Reply to Messages

## Description
The application must allow users to reply to specific messages, creating a threaded conversation context.

## Priority
**HIGH** - Critical for group chat usability

## User Story
As a user, I want to reply to specific messages in a chat, so that I can maintain context in conversations, especially in busy group chats.

## Acceptance Criteria

### AC-006.1: Initiating a Reply
- Users SHALL be able to select a message in the message list using vim navigation (j/k)
- Pressing `r` key on a selected message SHALL initiate reply mode
- The application SHALL display a reply indicator showing:
  - The sender of the original message
  - First 50 characters of the original message text
  - A visual indicator that reply mode is active (e.g., "Replying to: @john: Hello world...")

### AC-006.2: Reply Indicator Display
- The reply indicator SHALL appear above the input area
- The indicator SHALL be visually distinct (e.g., colored, bordered)
- Users SHALL be able to cancel reply mode by pressing `Escape`
- Canceling SHALL return to normal compose mode

### AC-006.3: Sending a Reply
- Users SHALL compose the reply message in the input area as normal
- When send keybinding is pressed, the application SHALL:
  - Send the message as a reply to the selected message
  - Include the reply context in the Telegram message
  - Clear reply mode indicator
  - Return to normal compose mode

### AC-006.4: Reply Display - Received Replies
- When displaying a message that is a reply, the application SHALL show:
  - A visual indication that the message is a reply (e.g., "↳" or "└─")
  - The sender and preview of the original message (e.g., "↳ @john: Hello world")
  - The reply message content below
- Replies SHALL be clearly associated with their parent message visually

### AC-006.5: Reply Display - Own Replies
- Replies sent by the user SHALL be displayed with the same format
- Users SHALL be able to see the context of their own replies

### AC-006.6: Reply in Groups
- In group chats, replies SHALL clearly show both:
  - Who sent the original message
  - Who sent the reply
- Reply threading SHALL work correctly even when multiple conversations occur simultaneously

### AC-006.7: Navigation to Original Message (Optional v2)
- This is marked optional for v1
- Future: When a reply is selected, pressing `g` + `r` could jump to the original message

## Technical Notes
- Use TDLib's `InputMessageReplyToMessage` for reply functionality
- Store reply-to message ID when initiating reply
- Handle edge case where original message is not in current message buffer

## Dependencies
- FR-005: Receive and Display Messages
- FR-004: Send Text Messages
- FR-010: Vim Keybindings (for 'r' keybinding)

## Related Requirements
- NFR-004: Usability (clear visual indication of reply context)
