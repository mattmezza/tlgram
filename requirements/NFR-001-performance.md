# NFR-001: Performance

## Description
The application must provide a highly responsive, "snappy as fuck" user experience with minimal latency across all operations.

## Priority
**CRITICAL** - Explicitly stated user requirement

## Performance Targets

### AC-001.1: UI Responsiveness
- **Target**: All UI updates SHALL complete within 50ms
- This applies to:
  - Keystroke to screen update
  - Message arrival to display
  - Switching between messages/chats
  - Mode transitions
  - Status bar updates
- **Measurement**: 95th percentile latency < 50ms
- **User Experience**: Should feel instant, no perceivable lag

### AC-001.2: Startup Time
- **Target**: Application SHALL start and display UI within 2 seconds
- This includes:
  - Loading configuration
  - Initializing TDLib
  - Authenticating (if session exists)
  - Loading initial chat list or opening specified chat
- Cold start (first-time auth) may take longer (acceptable)
- Warm start (session exists) SHALL be < 2 seconds

### AC-001.3: Message Rendering
- **Target**: New message SHALL appear on screen within 50ms of receipt
- This includes:
  - Receiving update from TDLib
  - Processing message data
  - Rendering formatted text
  - Updating display
- Should work for both text and media messages (media may load thumbnail later)

### AC-001.4: Scrolling Performance
- **Target**: Scrolling SHALL be smooth with no stutter
- Support smooth scrolling through 1000+ message history
- Use virtual scrolling / lazy loading if needed
- No frame drops during fast scrolling (holding j/k)

### AC-001.5: Search Performance
- **Target**: Client-side search SHALL return results within 100ms
- Searching through loaded messages (100-1000 messages)
- Server-side search (older messages) may take longer (acceptable)
- Incremental search (as-you-type) should feel responsive

### AC-001.6: Chat Switching
- **Target**: Switching between chats SHALL complete within 200ms
- This includes:
  - Loading chat history
  - Rendering messages
  - Updating UI
- Use caching to speed up switching back to recent chats

### AC-001.7: Memory Usage
- **Target**: Memory footprint SHALL remain reasonable
- Single instance: < 100 MB RAM (including TDLib)
- Should not grow unbounded (no memory leaks)
- Cache size should be limited and configurable

### AC-001.8: CPU Usage
- **Target**: Idle CPU usage SHALL be < 1%
- When receiving messages: Brief CPU spike acceptable
- No busy-waiting or polling loops
- Use event-driven architecture (TDLib updates)

### AC-001.9: Network Efficiency
- Use TDLib's efficient protocol (MTProto)
- Don't fetch unnecessary data
- Use message pagination (don't load entire chat history)
- Respect rate limits

### AC-001.10: Multiple Instances
- Running 3-5 instances SHALL NOT significantly degrade performance
- Each instance should maintain 50ms responsiveness target
- Total resource usage should scale linearly

## Optimization Strategies

### Rendering Optimization:
- Use efficient terminal rendering (only update changed areas)
- Implement double-buffering to prevent flicker
- Use incremental rendering for large message lists
- Consider using a TUI framework with optimized rendering (e.g., bubbletea)

### Data Structure Optimization:
- Use efficient data structures (ring buffers for messages)
- Index messages for fast lookup
- Cache rendered message strings

### Concurrency:
- Use goroutines for background tasks (network, file downloads)
- Never block the UI thread
- Use channels for safe communication between goroutines

### Profiling:
- Profile application during development
- Identify and eliminate bottlenecks
- Test with realistic data (100+ chats, 1000+ messages per chat)

## Performance Testing

### Test Scenarios:
1. **Startup test**: Time from launch to first UI render
2. **Keystroke latency test**: Measure input lag
3. **Message flood test**: Receive 100 messages rapidly, measure display latency
4. **Scroll stress test**: Scroll through 10,000 messages, measure smoothness
5. **Multi-instance test**: Run 5 instances, measure responsiveness
6. **Memory leak test**: Run for 24 hours, monitor memory usage

### Benchmarking:
- Use Go's built-in benchmarking: `go test -bench`
- Profile with pprof
- Measure with realistic data and load

## Acceptance Criteria Summary

| Metric | Target | Critical? |
|--------|--------|-----------|
| UI update latency | < 50ms | YES |
| Startup time | < 2s | NO |
| Message display latency | < 50ms | YES |
| Search latency (local) | < 100ms | NO |
| Memory usage (single) | < 100 MB | NO |
| Idle CPU usage | < 1% | NO |
| Multi-instance overhead | Linear scaling | NO |

Critical metrics MUST be met. Non-critical metrics are targets but can be relaxed if necessary.

## Technical Notes
- Use Go's performance profiling tools (pprof, trace)
- Consider using `github.com/charmbracelet/bubbletea` for efficient TUI rendering
- Optimize hot paths (message rendering, input handling)
- Use buffered channels to prevent blocking
- Implement efficient terminal refresh (only changed regions)

## Dependencies
- All functional requirements (performance affects all features)

## Related Requirements
- FR-018: Multiple Instance Support (scaling performance)
- NFR-002: Reliability (performance under load)
