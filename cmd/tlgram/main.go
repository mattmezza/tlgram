package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/pflag"

	"github.com/mattmezza/tlgram/internal/app"
	"github.com/mattmezza/tlgram/internal/config"
)

//go:embed changelog.txt
var changelogFS embed.FS

var (
	version = "dev"
	commit  = "none"
)

const (
	changelogMaxLines = 50
	changelogURL      = "https://github.com/mattmezza/tlgram/blob/main/CHANGELOG.md"
)

func main() {
	// CLI flags
	chatFlag := pflag.StringP("chat", "c", "", "Open specific chat by @username, chat ID, or alias")
	versionFlag := pflag.BoolP("version", "v", false, "Print version and exit")
	helpFlag := pflag.BoolP("help", "h", false, "Print help and exit")
	changelogFlag := pflag.Bool("changelog", false, "Print changelog and exit")

	pflag.Parse()

	if *helpFlag {
		printHelp()
		os.Exit(0)
	}

	if *versionFlag {
		fmt.Printf("tlgram %s (%s)\n", version, commit)
		os.Exit(0)
	}

	if *changelogFlag {
		printChangelog()
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Ensure session directory exists with proper permissions
	sessionDir := filepath.Join(cfg.ConfigDir, "session")
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating session directory: %v\n", err)
		os.Exit(1)
	}

	// Create and run the application
	model := app.New(cfg, *chatFlag)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running application: %v\n", err)
		os.Exit(1)
	}
}

func printChangelog() {
	content, err := changelogFS.ReadFile("changelog.txt")
	if err != nil {
		fmt.Println("Changelog not available in this build.")
		fmt.Printf("\nRead the full changelog at: %s\n", changelogURL)
		return
	}

	lines := strings.Split(string(content), "\n")
	truncated := false

	if len(lines) > changelogMaxLines {
		lines = lines[:changelogMaxLines]
		truncated = true
	}

	fmt.Println(strings.Join(lines, "\n"))

	if truncated {
		fmt.Println("\n...")
	}
	fmt.Printf("\nRead the full changelog at: %s\n", changelogURL)
}

func printHelp() {
	fmt.Println(`tlgram - Terminal Telegram Client

USAGE:
    tlgram [OPTIONS]

OPTIONS:
    -c, --chat <CHAT>    Open specific chat by:
                         - @username (e.g., --chat @john_doe)
                         - Chat ID (e.g., --chat 123456789)
                         - Alias (e.g., --chat work)
    -v, --version        Print version and exit
        --changelog      Print changelog and exit
    -h, --help           Print this help message

EXAMPLES:
    tlgram                     Open chat list
    tlgram --chat @john_doe    Open DM with @john_doe
    tlgram --chat work         Open chat aliased as "work" in config

CONFIGURATION:
    Config file: ~/.config/tlgram/config.toml
    Session:     ~/.config/tlgram/session/
    Logs:        ~/.config/tlgram/logs/

VIM KEYBINDINGS:
    j/k         Move down/up
    gg/G        Jump to top/bottom
    Ctrl-d/u    Scroll half page down/up
    i           Enter INSERT mode (compose message)
    Escape      Return to NORMAL mode
    r           Reply to selected message
    R           Mark messages as read up to cursor
    o           Jump to original message (for replies)
    Ctrl-o      Jump back after jumping
    yy          Copy message to clipboard
    u           Toggle full names / @usernames
    U           Mark dialog as unread
    Ctrl-p      Open chat switcher
    q           Quit application

For more information: https://github.com/mattmezza/tlgram`)
}
