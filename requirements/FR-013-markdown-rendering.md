# FR-013: Markdown Rendering

## Description
The application must render Telegram's markdown formatting (bold, italic, code, code blocks, links) in message text.

## Priority
**MEDIUM** - Important for readable communication

## User Story
As a user, I want to see formatted text in messages (bold, italic, code), so that messages are readable and convey emphasis as intended by the sender.

## Acceptance Criteria

### AC-013.1: Supported Formatting Types
The application SHALL render the following Telegram formatting:
- **Bold** text
- *Italic* text
- `Inline code` (monospace)
- ```Code blocks``` (multi-line monospace)
- [Links](https://example.com) (underlined or colored)
- ~~Strikethrough~~ text (if Telegram supports it)

### AC-013.2: Bold Text
- Bold text SHALL be displayed using terminal bold attribute
- IF bold not supported, use color or other visual indicator
- Example: `**hello**` displays as **hello**

### AC-013.3: Italic Text
- Italic text SHALL be displayed using terminal italic attribute
- IF italic not supported, use underline or other visual indicator
- Example: `*hello*` displays as *hello*

### AC-013.4: Inline Code
- Inline code SHALL be displayed with:
  - Different background color (e.g., gray)
  - OR different text color (e.g., cyan)
  - Monospace font (already terminal default)
- Example: `` `code` `` displays distinctly from regular text

### AC-013.5: Code Blocks
- Multi-line code blocks SHALL be displayed with:
  - Indentation or left border to distinguish from regular text
  - Different background or foreground color
  - Preserve all whitespace and newlines
  - Preserve indentation
- Example:
  ```
  def hello():
      print("world")
  ```
  Should be visually distinct from regular message text

### AC-013.6: Links
- Links SHALL be displayed with:
  - Underline or different color (e.g., blue)
  - Show URL text or anchor text depending on format
- Example: `[Example](https://example.com)` displays as "Example" (underlined/colored)
- IF terminal supports it, make clickable (OSC 8 hyperlinks)
- Clicking or pressing 'o' on selected message with link could open link (optional v2)

### AC-013.7: Strikethrough (Optional)
- IF Telegram supports strikethrough, render with strikethrough attribute
- IF terminal doesn't support, use different color or prefix/suffix indicator

### AC-013.8: Nested Formatting
- The application SHALL handle nested formatting:
  - Bold + italic: ***text***
  - Bold + code (where applicable)
- Render with all applicable attributes

### AC-013.9: Fallback for Unsupported Terminals
- IF terminal doesn't support certain attributes (bold, italic), the application SHALL:
  - Use color as fallback
  - OR use prefix/suffix markers (e.g., `*bold*`, `_italic_`)
  - Ensure formatting is still distinguishable

### AC-013.10: Formatting Detection
- The application SHALL use Telegram's entity data (not parse markdown manually)
- Telegram provides formatting information via message entities
- Extract entity type, offset, length from TDLib message object

### AC-013.11: Performance
- Rendering formatted text SHALL NOT significantly impact performance
- Target: <5ms additional rendering time per message with formatting
- Pre-process formatting when message is received, not during display

### AC-013.12: Composing Formatted Messages (Optional v2)
- v1 focus: Display formatted messages from others
- Future: Allow composing with markdown syntax
- For v1, sent messages are plain text only

## Example Display

### Input (from Telegram):
```
Check out this *important* feature:
- **Bold text** for emphasis
- `code snippets` inline
- Full code blocks:
```
func main() {
    fmt.Println("Hello")
}
```
See [documentation](https://example.com) for details.
```

### Expected Output (in TUI):
```
@john_doe
Check out this important feature:  (italic rendered)
- Bold text for emphasis          (bold rendered)
- code snippets inline            (code with different color/bg)
- Full code blocks:
  ┃ func main() {                 (code block with border)
  ┃     fmt.Println("Hello")
  ┃ }
See documentation for details.    (link underlined/colored)
```

## Technical Notes
- Use TDLib's message entities: `message.content.text.entities`
- Entity types: `bold`, `italic`, `code`, `pre`, `textUrl`, `mention`, etc.
- Apply terminal attributes using ANSI escape codes
- Consider using a Go library for terminal styling (e.g., `github.com/charmbracelet/lipgloss`)
- Test with terminals that have varying capability (basic, 256-color, truecolor)

## Dependencies
- FR-005: Receive and Display Messages

## Related Requirements
- NFR-004: Usability (readable formatted text)
- NFR-001: Performance (efficient rendering)
