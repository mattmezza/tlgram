package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/pflag"

	"github.com/mattmezza/tlgram"
	"github.com/mattmezza/tlgram/internal/app"
	"github.com/mattmezza/tlgram/internal/config"
)

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
	defaultConfigFlag := pflag.Bool("default-config", false, "Print default config to stdout")
	updateFlag := pflag.Bool("update", false, "Update tlgram to the latest version")
	checkUpdateFlag := pflag.Bool("check-update", false, "Check if a new version is available")

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

	if *defaultConfigFlag {
		fmt.Print(tlgram.DefaultConfig)
		os.Exit(0)
	}

	if *checkUpdateFlag {
		checkForUpdate()
		os.Exit(0)
	}

	if *updateFlag {
		doSelfUpdate()
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
	lines := strings.Split(tlgram.Changelog, "\n")
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
        --default-config Print default config to stdout
        --check-update   Check if a new version is available
        --update         Update tlgram to the latest version
    -h, --help           Print this help message

EXAMPLES:
    tlgram                     Open chat list
    tlgram --chat @john_doe    Open DM with @john_doe
    tlgram --chat 1234567890   Open chat by ID (use I key to find ID)
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
    A           Jump to bottom and enter INSERT mode
    Escape      Return to NORMAL mode
    r           Reply to selected message
    R           Mark messages as read up to cursor
    o           Jump to original message (for replies)
    Ctrl-o      Jump back after jumping
    yy          Copy message to clipboard
    cc          Edit own message
    D           Delete own message
    u           Toggle full names / @usernames
    U           Mark dialog as unread
    I           Toggle chat ID display in header
    Ctrl-p      Open chat switcher
    q           Quit application

For more information: https://github.com/mattmezza/tlgram`)
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func getLatestRelease() (*githubRelease, error) {
	resp, err := http.Get("https://api.github.com/repos/mattmezza/tlgram/releases/latest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func getAssetURL(release *githubRelease) string {
	// Build expected asset name for current platform
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	expectedName := fmt.Sprintf("tlgram_%s_%s%s", runtime.GOOS, runtime.GOARCH, ext)

	for _, asset := range release.Assets {
		if asset.Name == expectedName {
			return asset.BrowserDownloadURL
		}
	}
	return ""
}

func checkForUpdate() {
	release, err := getLatestRelease()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	latestVersion := release.TagName
	if version == latestVersion || "v"+version == latestVersion {
		fmt.Printf("tlgram %s is already the latest version\n", version)
	} else {
		fmt.Printf("Current version: %s\n", version)
		fmt.Printf("Latest version:  %s\n", latestVersion)
		fmt.Println("\nRun 'tlgram --update' to update")
	}
}

func doSelfUpdate() {
	release, err := getLatestRelease()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	latestVersion := release.TagName
	if version == latestVersion || "v"+version == latestVersion {
		fmt.Printf("tlgram %s is already the latest version\n", version)
		return
	}

	assetURL := getAssetURL(release)
	if assetURL == "" {
		fmt.Fprintf(os.Stderr, "No release found for %s/%s\n", runtime.GOOS, runtime.GOARCH)
		os.Exit(1)
	}

	fmt.Printf("Updating tlgram from %s to %s...\n", version, latestVersion)

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding executable: %v\n", err)
		os.Exit(1)
	}

	// Download and extract the update
	if err := downloadAndReplace(assetURL, exe); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully updated to %s\n", latestVersion)
	fmt.Println("\nChangelog: https://github.com/mattmezza/tlgram/blob/main/CHANGELOG.md")
}

func downloadAndReplace(url, exePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// For Windows, we'd need to handle .zip files differently
	// For now, handle .tar.gz (Linux/macOS)
	if runtime.GOOS == "windows" {
		return fmt.Errorf("Windows self-update not yet supported, please download manually from GitHub")
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Look for the tlgram binary in the archive
		if header.Typeflag == tar.TypeReg && filepath.Base(header.Name) == "tlgram" {
			// Write to a temporary file first
			tmpFile := exePath + ".new"
			f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				os.Remove(tmpFile)
				return err
			}
			f.Close()

			// Replace the old binary
			if err := os.Rename(tmpFile, exePath); err != nil {
				os.Remove(tmpFile)
				return err
			}
			return nil
		}
	}

	return fmt.Errorf("tlgram binary not found in archive")
}
