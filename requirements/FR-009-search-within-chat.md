# FR-009: Search Within Chat

## Description
The application must provide the ability to search for specific messages within the currently open chat.

## Priority
**MEDIUM** - Important for finding information in long conversations

## User Story
As a user, I want to search for specific text within a chat, so that I can quickly find past messages or information shared in the conversation.

## Acceptance Criteria

### AC-009.1: Trigger Search
- Pressing `/` key in chat view SHALL enter search mode
- The bottom of the screen SHALL show a search prompt: `Search: _`
- The cursor SHALL move to the search input

### AC-009.2: Search Input
- Users SHALL type their search query
- The query SHALL be case-insensitive by default
- Users SHALL be able to use Backspace to edit the query
- Users SHALL be able to press `Escape` to cancel search and return to normal view

### AC-009.3: Search Execution
- Pressing `Enter` SHALL execute the search
- The application SHALL search through:
  - All loaded message history in current chat
  - Message text content
  - Sender names (optional, but useful)
- The application SHALL highlight all matching messages in the current view

### AC-009.4: Search Results Display
- Matching messages SHALL be visually highlighted (e.g., background color, bold)
- The matching text within messages SHALL be emphasized (e.g., inverse colors)
- The application SHALL display search result count: `Match 1 of 15`
- IF no matches found, display: `No matches found for "query"`

### AC-009.5: Navigate Search Results
- After executing search, users SHALL navigate between matches:
  - `n` to jump to next match (forward in time)
  - `N` (or `Shift-n`) to jump to previous match (backward in time)
  - The view SHALL automatically scroll to show the matched message
  - The current match SHALL be indicated: `Match 3 of 15`

### AC-009.6: Search History Loading
- IF search matches exist in message history not yet loaded, the application SHALL:
  - Load older messages automatically
  - Continue searching in the newly loaded messages
  - Update match count
- This SHALL happen seamlessly during next/previous navigation

### AC-009.7: Exit Search Mode
- Pressing `Escape` SHALL exit search mode:
  - Clear search highlights
  - Clear search prompt
  - Return to normal chat view
  - Remain at current scroll position

### AC-009.8: Search Persistence
- Search highlights SHALL remain visible until:
  - User exits search mode
  - User performs a new search
  - User switches to a different chat

### AC-009.9: Performance
- Search SHALL be fast for loaded messages (<100ms)
- Search SHALL NOT block the UI
- Background history loading for search SHALL NOT affect UI responsiveness

## Example Display
```
[Normal chat view]
@john_doe: Hey, what was the API endpoint for users?
@matteo: It's /api/v1/users/list
@john_doe: Thanks!

[After typing '/api' and pressing Enter]
Search: api
Match 1 of 3

@john_doe: Hey, what was the API endpoint for users?
@matteo: It's /**api**/v1/users/list  ← highlighted
@john_doe: Thanks!
```

## Technical Notes
- Use TDLib's `searchChatMessages` for server-side search if needed
- Implement client-side search for loaded messages (faster)
- Combine both approaches: instant results from loaded messages, then expand to server search
- Optimize for common case (search within recent messages)

## Dependencies
- FR-005: Receive and Display Messages
- FR-010: Vim Keybindings (for /, n, N keys)

## Related Requirements
- NFR-001: Performance (fast search)
- NFR-004: Usability (vim-like search)
