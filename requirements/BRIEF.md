# Contractor Brief: tlgram - Terminal Telegram Client

## Project Overview

**Project Name**: tlgram
**Type**: Terminal User Interface (TUI) application
**Language**: Go (golang)
**Platform**: Linux (primary), macOS, Windows
**Purpose**: Lightweight, vim-keybinding Telegram client for terminal/tmux workflows

## Background

I run multiple tmux sessions for different projects, using [hop](https://github.com/mattmezza/hop) to template and script session creation. I need a TUI Telegram client that can:

1. **Open specific chats via CLI** (for tmux template integration)
2. **Run multiple instances simultaneously** (different chats in different tmux panes)
3. **Support vim keybindings** (for efficiency and muscle memory)
4. **Be snappy as fuck** (responsive, <50ms UI updates)
5. **Work seamlessly in tmux/SSH environments**

**The Problem**: Existing Telegram TUI clients (tg, nchat, arigram, tgt) don't support opening specific chats via CLI arguments, which breaks the tmux workflow.

**The Solution**: Build a custom client that fits my exact needs.

## What Needs to Be Built

A terminal-based Telegram client with:

### Core Features (Must Have)
- ✅ **Telegram authentication** (phone + SMS + 2FA via TDLib)
- ✅ **CLI chat opening**: `tlgram --chat @username` or `tlgram --chat work`
- ✅ **Send and receive text messages** (real-time)
- ✅ **Vim keybindings** (hjkl, gg/G, modal interface)
- ✅ **Multiple instances** (shared auth, independent UI state)
- ✅ **Chat list with unread indicators**
- ✅ **Fuzzy chat switcher** (built-in fzf-like, Ctrl-p)
- ✅ **Chat aliases** (define shortcuts in config: `work="@john"`)

### Important Features (Should Have)
- ✅ **Reply to messages** (essential for group chats)
- ✅ **Markdown rendering** (bold, italic, code blocks)
- ✅ **Unread message separator** (visual line showing what's new)
- ✅ **Search within chat** (vim-style /, n, N)
- ✅ **Media file handling** (display info, download on demand)
- ✅ **Copy to clipboard** (y to yank, works in tmux/SSH)
- ✅ **Auto-reconnect** (handle network issues gracefully)
- ✅ **Status bar** (chat name + connection status)

### Configuration
- ✅ **TOML config file** at `~/.config/tlgram/config.toml`
- ✅ **Configurable keybindings, download paths, behavior**

## Comprehensive Requirements

**All detailed requirements are in the `requirements/` directory:**

- **[requirements/README.md](requirements/README.md)** - Start here for complete overview
- **18 Functional Requirements (FR-001 to FR-018)** - What the app must do
- **6 Non-Functional Requirements (NFR-001 to NFR-006)** - How it must perform

**Please read all requirement documents before providing a quote.**

## Technical Constraints

### Technology Stack (Required)
- **Language**: Go 1.21+
- **Telegram API**: TDLib (official Telegram client library)
- **TUI Framework**: Your choice, but recommend:
  - [bubbletea](https://github.com/charmbracelet/bubbletea) + [lipgloss](https://github.com/charmbracelet/lipgloss) (modern, popular)
  - [tview](https://github.com/rivo/tview) (mature, feature-rich)
  - Custom with [tcell](https://github.com/gdamore/tcell)

### Performance Requirements (Critical)
- **UI responsiveness**: <50ms for all operations (95th percentile)
- **Startup time**: <2 seconds (warm start with existing session)
- **Memory usage**: <100MB per instance
- **Multiple instances**: 3-5 instances without performance degradation

### Security Requirements (Critical)
- Use TDLib's authentication (don't roll your own)
- Session files with 0600 permissions
- No logging of sensitive data (passwords, auth codes)
- Encrypted communication (TDLib handles this)

### Compatibility Requirements
- **Primary**: Linux with tmux (my environment)
- **Secondary**: macOS, native terminals
- **Nice to have**: Windows
- Must work over SSH with clipboard support (OSC 52)

## Deliverables

### Code
1. **Source code** on GitHub (or similar)
   - Clean, well-organized Go code
   - Follow Go best practices and idioms
   - MIT or similar permissive license
2. **Build system**
   - Simple `go build` or Makefile
   - Cross-compilation for Linux (amd64, arm64), macOS
3. **Tests**
   - Unit tests for core logic
   - >50% coverage for critical paths
   - Pass `go vet` and linting

### Documentation
1. **README.md**
   - Installation instructions
   - Quick start guide
   - Basic usage examples
   - Configuration guide
2. **Config file template** with comments
3. **Code documentation** (godoc comments)

### Binary Releases
1. **Linux x86_64** (primary)
2. **Linux ARM64**
3. **macOS Intel + Apple Silicon**

## Development Approach

### Phased Development (Recommended)

**Phase 1: MVP** (~40% of effort)
- Authentication + session management
- Open specific chat via CLI
- Send/receive text messages
- Basic vim navigation (hjkl, gg/G)
- Minimal viable UI

**Milestone**: Can open a chat, send/receive messages, navigate with vim keys

**Phase 2: Multi-Chat** (~30% of effort)
- Chat list display
- Chat switcher with fuzzy search
- Chat aliases
- Multiple instance support
- Unread indicators

**Milestone**: Full tmux workflow supported

**Phase 3: Polish** (~20% of effort)
- Reply, search, markdown
- Media handling, clipboard
- Configuration system
- Network reconnection

**Milestone**: Feature-complete v1.0

**Phase 4: Testing & Release** (~10% of effort)
- Comprehensive testing
- Bug fixes
- Documentation polish
- Release packaging

**Milestone**: Production-ready release

### Alternative: All-at-once
If you prefer to deliver everything in one go, that's fine too. Discuss your preferred approach.

## Project Management

### Communication
- **Preferred**: GitHub issues + pull requests
- **Updates**: Weekly progress updates (or as agreed)
- **Questions**: I'm available for clarification via email/chat

### Code Review
- I'll review pull requests promptly
- Focus on: correctness, performance, usability
- Open to your architectural decisions

### Timeline
- **Estimate required**: Please provide realistic timeline
- **Preferred**: Phased delivery with milestones
- **Flexibility**: I understand software development is unpredictable

## Evaluation Criteria

### Your Proposal Should Include
1. **Timeline**: Estimated time to complete (by phase or total)
2. **Cost**: Your rate and total cost estimate
3. **Approach**: Brief description of how you'll build it
4. **Experience**: Relevant experience with Go, TUI apps, TDLib
5. **Questions**: Any clarifying questions or concerns
6. **Availability**: When you can start, hours per week

### What I'm Looking For
- ✅ **Strong Go experience** (not learning on the job)
- ✅ **TUI development experience** (or willingness to learn)
- ✅ **Attention to detail** (requirements are comprehensive for a reason)
- ✅ **Performance-conscious** (snappy UI is critical)
- ✅ **Good communicator** (able to ask questions, provide updates)
- ✅ **Ownership mindset** (take initiative, suggest improvements)

### Nice to Have
- Experience with TDLib or Telegram API
- Experience with tmux workflows
- vim user (understands vim muscle memory)
- Terminal enthusiast (understands terminal limitations/capabilities)

## Success Criteria

The project is successful when:

1. **Functional**: All critical requirements implemented
   - Can open specific chats via CLI
   - Multiple instances work correctly
   - Vim keybindings feel natural
   - All core messaging features work

2. **Performance**: Meets performance targets
   - UI feels snappy (<50ms updates)
   - Startup is fast (<2s)
   - Works smoothly with 3-5 instances

3. **Quality**: Code is maintainable
   - Clean, readable Go code
   - Reasonable test coverage
   - Good documentation

4. **Integration**: Works in my environment
   - tmux integration seamless
   - Clipboard works in SSH sessions
   - hop template integration straightforward

5. **User Acceptance**: I can use it daily
   - Replaces my need for desktop Telegram
   - Doesn't slow down my workflow
   - Feels like a natural part of my terminal environment

## Getting Started

### Step 1: Review Requirements
- Read [requirements/README.md](requirements/README.md)
- Skim all FR and NFR documents
- Note any questions or concerns

### Step 2: Evaluate Feasibility
- Assess TDLib integration complexity
- Evaluate chosen TUI framework
- Identify any technical risks

### Step 3: Submit Proposal
Include:
- Timeline and cost
- Your approach
- Relevant experience
- Any questions
- Availability

### Step 4: Discussion (If Selected)
- Clarify any requirements
- Discuss architecture approach
- Agree on milestones and payment terms
- Establish communication channels

### Step 5: Development
- Set up repository and development environment
- Implement phased approach (or all-at-once)
- Regular check-ins and demos
- Iterate based on feedback

## Questions?

Feel free to reach out with questions about:
- Requirements clarification
- Technical approach
- Timeline expectations
- Budget constraints
- Anything else

## Repository Structure

This repository contains:
```
tlgram/
├── requirements/              # All requirement documents (READ THIS FIRST)
│   ├── README.md             # Requirements overview
│   ├── FR-001-*.md           # Functional requirements (18 files)
│   └── NFR-001-*.md          # Non-functional requirements (6 files)
├── CONTRACTOR_BRIEF.md       # This file
└── [Your code will go here]
```

## Final Notes

This is a real project for my daily workflow. I've spent significant time defining requirements because I know exactly what I need. I value:

- **Quality over speed**: Get it right, not just done
- **Performance**: The "snappy" requirement is non-negotiable
- **Usability**: If vim keybindings don't feel natural, it's not done
- **Communication**: Keep me informed, ask questions early


Looking forward to your proposal!

---

**Last Updated**: 2026-01-09
**Status**: Seeking contractor for development
**Expected Start**: As soon as possible


Final note: i plan to release open sourced the result of your work. Pay extra attention to making sure the repository is ripe for collaboration.
