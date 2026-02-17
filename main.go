package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	paths := ResolvePaths()
	cmd := os.Args[1]

	if cmd == "login" {
		if err := runLogin(paths); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	cfg, err := LoadConfig(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'slack-status login' to set up your token.\n")
		os.Exit(1)
	}

	switch cmd {
	case "work":
		KillWorker(paths.PIDFile)
		expiration := todaySixPM()
		if err := SetStatus(cfg.Token, "Working remote", ":computer:", expiration); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Status set: Working remote (until 6pm)")

	case "lunch":
		KillWorker(paths.PIDFile)
		expiration := time.Now().Add(1 * time.Hour).Unix()
		if err := SetStatus(cfg.Token, "Lunch", ":hamburger:", expiration); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		binaryPath, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error resolving binary path: %v\n", err)
			os.Exit(1)
		}
		if err := SpawnReturnWorker(binaryPath, paths.PIDFile); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not spawn return worker: %v\n", err)
		}
		now := time.Now()
		returnAt := now.Add(1 * time.Hour)
		fmt.Printf("Status set: Lunch\n")
		fmt.Printf("  Started:  %s\n", now.Format("Mon Jan 2, 3:04 PM"))
		fmt.Printf("  Returns:  %s (Working remote)\n", returnAt.Format("Mon Jan 2, 3:04 PM"))

	case "clear":
		KillWorker(paths.PIDFile)
		if err := ClearStatus(cfg.Token); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Status cleared")

	case "_return-worker":
		RunReturnWorker(cfg.Token, paths.PIDFile)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

// oauthURL is set at build time via -ldflags "-X main.oauthURL=..."
// Falls back to the generic Slack apps page if not provided.
var oauthURL = "https://api.slack.com/apps"

func runLogin(paths Paths) error {
	scanner := bufio.NewScanner(os.Stdin)

	if _, err := os.Stat(paths.ConfigFile); err == nil {
		fmt.Printf("Config already exists at %s\n", paths.ConfigFile)
		fmt.Print("Overwrite? [y/N]: ")
		scanner.Scan()
		if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	fmt.Println("Opening Slack OAuth page in your browser...")
	fmt.Println()
	// best-effort browser open; ignore error if it fails
	openBrowser(oauthURL)
	fmt.Printf("If the browser didn't open, visit:\n  %s\n\n", oauthURL)
	fmt.Println("1. Click 'Install to Workspace' and authorize")
	fmt.Println("2. Copy the User OAuth Token (starts with xoxp-)")
	fmt.Println()
	fmt.Print("Paste token: ")

	scanner.Scan()
	token := strings.TrimSpace(scanner.Text())

	if token == "" {
		return fmt.Errorf("no token entered")
	}
	if !strings.HasPrefix(token, "xoxp-") {
		return fmt.Errorf("token should start with xoxp-")
	}

	if err := SaveConfig(paths, token); err != nil {
		return err
	}

	fmt.Printf("\nToken saved to %s\n", paths.ConfigFile)
	fmt.Println("You're all set! Try: slack-status work")
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: slack-status <command>")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  login  Authenticate with Slack")
	fmt.Fprintln(os.Stderr, "  work   Set status to 'Working remote' until 6pm today")
	fmt.Fprintln(os.Stderr, "  lunch  Set status to 'Lunch' for 1 hour, then auto-restore")
	fmt.Fprintln(os.Stderr, "  clear  Clear status")
}

func openBrowser(url string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	exec.CommandContext(ctx, "open", url).Run()
}

func todaySixPM() int64 {
	now := time.Now()
	sixPM := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())
	if now.After(sixPM) {
		// If already past 6pm, set to 6pm tomorrow
		sixPM = sixPM.Add(24 * time.Hour)
	}
	return sixPM.Unix()
}
