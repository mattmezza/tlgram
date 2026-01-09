# tlgram Implementation Plan

## Summary

Terminal Telegram client in Go using **Bubbletea + Lipgloss** with **TDLib statically linked**. Focus on Linux first, full CI/CD with GitHub Actions.

---

## Architecture

```
cmd/tlgram/main.go          # CLI entry, bootstrap
internal/
  app/model.go              # Main Bubbletea model, message routing
  ui/
    auth/                   # Phone/SMS/2FA flow
    chatlist/               # Chat list with unread badges
    chatview/               # Messages, input, vim modes
    switcher/               # Ctrl-p fuzzy search overlay
    statusbar/              # Chat name, connection, mode indicator
  telegram/
    client.go               # TDLib wrapper, session management
    auth.go                 # Auth state machine
    chat.go                 # Chat operations
    message.go              # Message send/receive
  config/config.go          # TOML parsing, defaults
  keybind/vim.go            # Vim state machine (gg, counts, modes)
  clipboard/                # OSC 52, tmux, xclip fallbacks
  store/                    # Local state (msg queue, read positions)
scripts/build-tdlib.sh      # TDLib static build
Dockerfile.tdlib            # Reproducible TDLib builds
```

---

## TDLib Integration (Static Linking)

**Wrapper**: `zelenin/go-tdlib` (CGO binding)

**Build TDLib**:
```bash
# In Docker or locally
git clone --branch v1.8.60 https://github.com/tdlib/td.git
cd td && mkdir build && cd build
cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/opt/tdlib -DTD_ENABLE_LTO=ON ..
cmake --build . --target tdjson_static -j$(nproc)
cmake --install .
```

**CGO Flags**:
```
CGO_CFLAGS="-I/opt/tdlib/include"
CGO_LDFLAGS="-L/opt/tdlib/lib -ltdjson_static -ltdjson_private -ltdclient -ltdcore -ltdapi -ltdactor -ltddb -ltdsqlite -ltdnet -ltdutils -lstdc++ -lssl -lcrypto -ldl -lz -lm -lpthread"
```

**Multi-instance**: Each process creates own TDLib client, all share `~/.config/tlgram/session/` (TDLib handles concurrency via file locking).

---

## Phase 1: MVP (3-4 weeks)

**Goal**: Open specific chat, send/receive messages, vim navigation works

| Step | Task | Days |
|------|------|------|
| 1 | Project scaffolding, deps (bubbletea, lipgloss, bubbles, go-tdlib, toml, pflag) | 1 |
| 2 | TDLib build system (Makefile, Docker, scripts) | 2 |
| 3 | TDLib client wrapper (init, session, updates channel) | 3 |
| 4 | Auth module (phone, SMS code, 2FA) | 2 |
| 5 | Basic config (TOML loading, defaults, path expansion) | 1 |
| 6 | CLI entry (--chat, --help, --version, graceful shutdown) | 1 |
| 7 | Auth UI (phone input, code input, 2FA masked input) | 2 |
| 8 | Chat view (message list viewport, text input, basic styling) | 4 |
| 9 | Vim navigation (j/k, gg/G, Ctrl-d/u, NORMAL/INSERT modes) | 2 |
| 10 | Message send/receive (Ctrl-Enter, optimistic UI, TDLib updates) | 2 |
| 11 | Status bar (chat name, connection, mode indicator) | 1 |
| 12 | Integration testing, polish | 2 |

**Deliverable**: `tlgram --chat @username` works with vim navigation and real-time messaging.

---

## Phase 2: Multi-Chat (2 weeks)

| Feature | Description |
|---------|-------------|
| Chat list | All chats sorted by recency, unread badges, j/k navigation, Enter to open |
| Chat switcher | Ctrl-p overlay, fuzzy search, ranked results, Enter/Escape |
| Chat aliases | `[chat_aliases]` in config, resolve in CLI and switcher |
| Unread separator | Visual line between read/unread, persists until scrolled past |
| Multi-instance testing | 3-5 instances, verify real-time sync |

---

## Phase 3: Polish (2 weeks)

| Feature | Key |
|---------|-----|
| Reply to messages | `r` key, reply context above input |
| Vim search | `/` to search, `n`/`N` next/prev, highlight matches |
| Markdown rendering | Bold, italic, code via Telegram entities + lipgloss |
| Media handling | `[Image] name (size)`, `d` to download, progress |
| Clipboard | `y` to yank, OSC 52 for SSH/tmux, xclip/pbcopy fallback |
| Network reconnect | Detect disconnect, exponential backoff, status updates, message queue |
| Full config | All options, keybinding customization |

---

## Phase 4: Release (1 week)

- Comprehensive testing (unit, integration, manual)
- README with quick start, config docs
- CONTRIBUTING.md, ARCHITECTURE.md
- GitHub Actions release workflow with goreleaser
- Pre-built Linux x86_64 binary

---

## CI/CD Pipeline

**`.github/workflows/ci.yml`**:
- Lint (golangci-lint)
- Test (with cached TDLib)
- Build artifact

**`.github/workflows/release.yml`**:
- Trigger on `v*` tags
- goreleaser for Linux x86_64, arm64
- Checksums, changelog

---

## Key Design Decisions

### Bubbletea Model Structure
```go
type Model struct {
    mode        Mode  // NORMAL, INSERT, SEARCH, COMMAND
    currentView View  // Auth, ChatList, Chat, Switcher
    auth        auth.Model
    chatList    chatlist.Model
    chatView    chatview.Model
    switcher    switcher.Model
    statusBar   statusbar.Model
    telegram    *telegram.Client
    config      *config.Config
    keyBuffer   []string  // For gg, 5j, etc.
}
```

### Vim State Machine
- Buffer keys with 500ms timeout
- Handle count prefixes (5j = move down 5)
- Sequence detection (gg = jump top)
- Mode-aware key handling

### Message Flow
```
Terminal Input -> tea.KeyMsg -> Model.Update() -> State Change -> Model.View()
TDLib Update -> TelegramMsg (via channel) -> Model.Update() -> State Change -> Model.View()
```

---

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| TDLib static linking complexity | Docker for reproducible builds, thorough docs |
| Multi-instance session conflicts | TDLib built-in file locking, extensive testing |
| Performance with large histories | Virtual scrolling, lazy loading, limit loaded messages |
| macOS/Windows builds | Phase 1 Linux only, add platforms incrementally |

**Fallbacks**:
- If static linking fails: dynamic linking with bundled .so
- If zelenin/go-tdlib issues: switch to Arman92/go-tdlib
- If Bubbletea perf insufficient: consider tview

---

## Critical Files (First Implementation)

1. `scripts/build-tdlib.sh` - Dev environment setup
2. `internal/telegram/client.go` - TDLib wrapper foundation
3. `internal/app/model.go` - Main Bubbletea coordinator
4. `internal/ui/chatview/model.go` - Core user-facing component
5. `internal/keybind/vim.go` - Vim mode state machine

---

## Verification

After each phase:
1. **Manual test**: Auth flow, send/receive, vim navigation
2. **Performance test**: UI latency <50ms, startup <2s
3. **Multi-instance test**: 3 instances, same session, no conflicts
4. **CI passing**: lint, tests, build
