# Contributing to tlgram

Thank you for your interest in contributing to tlgram! This document provides guidelines for contributing to the project.

## Development Setup

### Prerequisites

- Go 1.22 or later
- Make (optional, but recommended)

### Getting Started

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/tlgram.git
   cd tlgram
   ```
3. Install dependencies:
   ```bash
   make deps
   # or: go mod download
   ```
4. Build the project:
   ```bash
   make build
   # or: go build -o tlgram ./cmd/tlgram
   ```
5. Run tests:
   ```bash
   make test
   ```

## Making Changes

### Branch Naming

Create a branch with a descriptive name:

- `feat/add-emoji-reactions` - for new features
- `fix/message-display-bug` - for bug fixes
- `docs/update-readme` - for documentation changes
- `refactor/cleanup-client` - for code refactoring

### Commit Messages

Follow the conventional commits format:

```
type: short description

Optional longer description explaining the change.
```

**Types:**
- `feat` - new feature
- `fix` - bug fix
- `docs` - documentation only
- `refactor` - code change that neither fixes a bug nor adds a feature
- `test` - adding or updating tests
- `chore` - maintenance tasks (dependencies, CI, etc.)

**Examples:**
```
feat: add message search functionality
fix: resolve crash when switching chats rapidly
docs: add tmux integration examples
```

### Code Style

- Run `make fmt` before committing to format code
- Run `make lint` to check for common issues
- Run `make check` to run all checks (fmt, vet, lint, test)

The CI pipeline will verify these checks on every pull request.

## Pull Request Process

1. Update your fork with the latest changes from main:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. Make sure all checks pass:
   ```bash
   make check
   ```

3. Update `CHANGELOG.md` under the `[Unreleased]` section describing your changes

4. Push your branch and create a pull request on GitHub

5. Fill out the pull request template with:
   - A clear description of what changed
   - Why the change was needed
   - How to test the change

6. Wait for review - maintainers may request changes or ask questions

## Reporting Issues

### Bug Reports

When reporting bugs, please include:

- tlgram version (`tlgram --version`)
- Operating system and version
- Terminal emulator being used
- Steps to reproduce the issue
- Expected vs actual behavior
- Any relevant error messages or logs

### Feature Requests

For feature requests, please describe:

- The problem you're trying to solve
- Your proposed solution
- Any alternatives you've considered

## Project Structure

```
tlgram/
├── cmd/tlgram/       # Main application entry point
├── internal/
│   ├── app/          # Application model and update logic
│   ├── clipboard/    # Clipboard operations
│   ├── config/       # Configuration management
│   ├── keybind/      # Vim keybinding handler
│   ├── store/        # Data persistence
│   ├── telegram/     # Telegram client wrapper
│   └── ui/           # UI components
│       ├── auth/     # Authentication screens
│       ├── chatlist/ # Chat list view
│       ├── chatview/ # Message view
│       ├── statusbar/# Status bar
│       └── switcher/ # Chat switcher
├── requirements/     # Project requirements documents
└── Makefile          # Build automation
```

## Testing

- Add tests for new functionality when possible
- Run `make test-race` to check for race conditions
- Run `make test-cover` to generate a coverage report

## Questions?

Feel free to open an issue if you have questions about contributing!
