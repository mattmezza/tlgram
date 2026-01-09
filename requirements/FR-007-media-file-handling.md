# FR-007: Media File Handling

## Description
The application must handle media files (images, videos, documents, audio) by displaying file information and providing download capability.

## Priority
**MEDIUM** - Important for complete messaging experience

## User Story
As a user, I want to see when someone sends me a file and be able to download it easily, so that I can access shared content without leaving my terminal workflow.

## Acceptance Criteria

### AC-007.1: Media Message Detection
- The application SHALL detect when a message contains media:
  - Photos/Images
  - Videos
  - Documents/Files
  - Audio files
  - Voice messages
  - Video messages (round videos)
  - Stickers (displayed as text indicator)

### AC-007.2: Media Message Display
- For messages containing media, the application SHALL display:
  - **File type indicator**: e.g., `[Image]`, `[Video]`, `[Document]`, `[Audio]`, `[Voice]`, `[Sticker]`
  - **File name**: if available (for documents)
  - **File size**: in human-readable format (e.g., "2.3 MB")
  - **Caption**: if the media has a text caption, display it below the file info
  - **Download status**: `[Not downloaded]`, `[Downloading... 45%]`, `[Downloaded]`

### AC-007.3: Download Functionality
- Users SHALL select a media message using vim navigation (j/k)
- Pressing `d` key SHALL initiate download
- During download, the application SHALL:
  - Display download progress percentage
  - Update progress in real-time
  - Allow cancellation via `Ctrl-c` (but keep app running)

### AC-007.4: Download Location
- Files SHALL be downloaded to a configurable directory
- Default download location SHALL be `~/Downloads/tlgram/` or `~/Downloads/`
- The configuration SHALL allow specifying custom download path
- After successful download, the application SHALL:
  - Display "Downloaded to: /path/to/file"
  - Update the message to show `[Downloaded]` status

### AC-007.5: Re-download Prevention
- IF a file has already been downloaded, the application SHALL:
  - Show `[Downloaded]` status
  - Display the saved file path
  - NOT re-download unless explicitly requested
- Users SHALL be able to force re-download with `Shift-d` or similar

### AC-007.6: No Automatic Downloads
- The application SHALL NOT automatically download media files
- All downloads SHALL be user-initiated
- This keeps the application lightweight and prevents unwanted bandwidth usage

### AC-007.7: No Inline Preview
- The application SHALL NOT render images inline (no ASCII art, no kitty protocol)
- The application SHALL NOT preview videos or audio
- Focus is on showing file information only

### AC-007.8: Opening Downloaded Files (Optional v2)
- Marked optional for v1
- Future: `o` key could open downloaded file with system default application

## Example Display
```
@john_doe
Hey, check this out!
[Image] screenshot.png (1.2 MB) [Not downloaded]
  Caption: Our new dashboard design

@matteo
[You]
Looks good! Small typo in the header though.

@john_doe
[Document] requirements.pdf (456 KB) [Downloaded]
  Downloaded to: ~/Downloads/tlgram/requirements.pdf
```

## Technical Notes
- Use TDLib's file download APIs
- Implement progress tracking via TDLib updates
- Store download status per file ID in local state
- Check file existence before marking as downloaded

## Dependencies
- FR-005: Receive and Display Messages
- FR-011: Configuration Management (for download path)

## Related Requirements
- NFR-001: Performance (downloads should not block UI)
- NFR-002: Reliability (handle failed downloads gracefully)
