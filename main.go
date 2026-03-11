package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	options, err := parseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		usage()
		os.Exit(1)
	}

	if options.Command == "" {
		usage()
		os.Exit(1)
	}

	paths := ResolvePaths()
	cmd := options.Command

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
	case "start":
		result, err := runStart(cfg.Token, paths, options.Until)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if options.JSON {
			printJSON(result)
		}

	case "work":
		result, err := runWork(cfg.Token, paths, false, options.Until)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if options.JSON {
			printJSON(result)
		}

	case "lunch":
		result, err := runLunch(cfg.Token, paths, options.Until)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if options.JSON {
			printJSON(result)
		}

	case "clear":
		result, err := runClear(cfg.Token, paths)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if options.JSON {
			printJSON(result)
		}

	case "status":
		state, err := loadOrDefaultState(paths)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		printJSON(commandResult{Command: "status", OK: true, State: state})

	case "_return-worker":
		RunReturnWorker(cfg.Token, paths)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

type options struct {
	Command string
	JSON    bool
	Until   string
}

type commandResult struct {
	Command string     `json:"command"`
	OK      bool       `json:"ok"`
	State   LocalState `json:"state"`
}

func parseOptions(args []string) (options, error) {
	args = normalizeJSONFlag(args)
	fs := flag.NewFlagSet("slack-status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonMode := fs.Bool("json", false, "output machine-readable JSON")
	until := fs.String("until", "", "expiration time for start/work, e.g. 18:00 or 2026-03-11T18:00")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		return options{JSON: *jsonMode, Until: *until}, nil
	}
	return options{Command: rest[0], JSON: *jsonMode, Until: *until}, nil
}

func normalizeJSONFlag(args []string) []string {
	if len(args) >= 2 && args[1] == "--json" {
		return []string{"--json", args[0:1][0]}
	}
	return args
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "error: encoding JSON output: %v\n", err)
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

func runWork(token string, paths Paths, postMessage bool, until string) (commandResult, error) {
	KillWorker(paths.PIDFile)
	now := time.Now()
	expirationTime, err := resolveWorkExpiration(now, postMessage, until)
	if err != nil {
		return commandResult{}, err
	}
	expiration := expirationTime.Unix()
	if err := SetStatus(token, "Working remotely", ":computer:", expiration); err != nil {
		return commandResult{}, err
	}
	state := withDerivedState(LocalState{CurrentStatus: workingStatusState("cli", expiration), UpdatedAt: now.Format(time.RFC3339)}, now)
	if err := SaveLocalState(paths, state); err != nil {
		return commandResult{}, err
	}
	fmt.Println("Status set: Working remotely")
	fmt.Printf("  Expires:  %s\n", formatFriendlyDateTime(expirationTime, now))
	command := "work"
	if postMessage {
		command = "start"
	}
	return commandResult{Command: command, OK: true, State: state}, nil
}

func runStart(token string, paths Paths, until string) (commandResult, error) {
	state, err := loadOrDefaultState(paths)
	if err != nil {
		return commandResult{}, err
	}
	now := time.Now()
	state = withDerivedState(state, now)
	if !state.StartAvailableToday {
		return commandResult{}, fmt.Errorf("start has already been used today")
	}

	result, err := runWork(token, paths, true, until)
	if err != nil {
		return commandResult{}, err
	}
	if err := PostMessage(token, "#remote_work", ":wave:"); err != nil {
		return commandResult{}, fmt.Errorf("sending message: %w", err)
	}

	result.State.LastStartAt = now.Format(time.RFC3339)
	result.State.LastStartDay = localDayStamp(now)
	result.State = withDerivedState(result.State, now)
	if err := SaveLocalState(paths, result.State); err != nil {
		return commandResult{}, err
	}
	return result, nil
}

func runLunch(token string, paths Paths, until string) (commandResult, error) {
	KillWorker(paths.PIDFile)
	now := time.Now()
	priorState, err := loadOrDefaultState(paths)
	if err != nil {
		return commandResult{}, err
	}
	returnAt, err := resolveLunchReturnTime(now, until)
	if err != nil {
		return commandResult{}, err
	}
	expiration := returnAt.Unix()
	if err := SetStatus(token, "Lunch", ":hamburger:", expiration); err != nil {
		return commandResult{}, err
	}
	binaryPath, err := os.Executable()
	if err != nil {
		return commandResult{}, fmt.Errorf("resolving binary path: %w", err)
	}
	workerPID, workerErr := SpawnReturnWorker(binaryPath, paths.PIDFile, returnAt)
	if workerErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not spawn return worker: %v\n", workerErr)
	}
	state := withDerivedState(LocalState{
		CurrentStatus: StateStatus{
			Command:         "lunch",
			Text:            "Lunch",
			Emoji:           ":hamburger:",
			StatusExpiresAt: time.Unix(expiration, 0).Format(time.RFC3339),
			WillReturnTo:    "work",
			Source:          "cli",
		},
		WorkerScheduled: workerErr == nil,
		WorkerPID:       workerPID,
		LastStartAt:     priorState.LastStartAt,
		LastStartDay:    priorState.LastStartDay,
		UpdatedAt:       now.Format(time.RFC3339),
	}, now)
	if err := SaveLocalState(paths, state); err != nil {
		return commandResult{}, err
	}
	fmt.Printf("Status set: Lunch\n")
	fmt.Printf("  Started:  %s\n", formatFriendlyDateTime(now, now))
	fmt.Printf("  Returns:  %s (Working remotely)\n", formatFriendlyDateTime(returnAt, now))
	return commandResult{Command: "lunch", OK: true, State: state}, nil
}

func runClear(token string, paths Paths) (commandResult, error) {
	KillWorker(paths.PIDFile)
	if err := ClearStatus(token); err != nil {
		return commandResult{}, err
	}
	state := withDerivedState(LocalState{
		CurrentStatus: StateStatus{Command: "clear", Source: "cli"},
		UpdatedAt:     time.Now().Format(time.RFC3339),
	}, time.Now())
	if err := SaveLocalState(paths, state); err != nil {
		return commandResult{}, err
	}
	fmt.Println("Status cleared")
	return commandResult{Command: "clear", OK: true, State: state}, nil
}

func loadOrDefaultState(paths Paths) (LocalState, error) {
	now := time.Now()
	state, err := LoadLocalState(paths)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return withDerivedState(LocalState{}, now), nil
		}
		return LocalState{}, err
	}
	return withDerivedState(*state, now), nil
}

func preservedStartState(paths Paths) LocalState {
	state, err := loadOrDefaultState(paths)
	if err != nil {
		return withDerivedState(LocalState{}, time.Now())
	}
	return state
}

func workingStatusState(source string, expiration int64) StateStatus {
	return StateStatus{
		Command:         "work",
		Text:            "Working remotely",
		Emoji:           ":computer:",
		StatusExpiresAt: time.Unix(expiration, 0).Format(time.RFC3339),
		Source:          source,
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: slack-status [--json] [--until <time>] <command>")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  login  Authenticate with Slack")
	fmt.Fprintln(os.Stderr, "  start  Set status to 'Working remotely' for 9 hours by default, and sends a message to #remote_work")
	fmt.Fprintln(os.Stderr, "  work   Set status to 'Working remotely' until 6pm today by default")
	fmt.Fprintln(os.Stderr, "  lunch  Set status to 'Lunch' for 1 hour by default, or until --until, then auto-restore")
	fmt.Fprintln(os.Stderr, "  clear  Clear status")
	fmt.Fprintln(os.Stderr, "  status Print machine-readable local state")
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr, "  --json          Output machine-readable JSON for app integration")
	fmt.Fprintln(os.Stderr, "  --until <time>  Override expiration for start/work/lunch with HH:MM, YYYY-MM-DD HH:MM, YYYY-MM-DDTHH:MM, or RFC3339")
}

func openBrowser(url string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	exec.CommandContext(ctx, "open", url).Run()
}

func todaySixPM() int64 {
	now := time.Now()
	return todaySixPMAt(now).Unix()
}

func todaySixPMAt(now time.Time) time.Time {
	sixPM := time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, now.Location())
	if now.After(sixPM) {
		sixPM = sixPM.Add(24 * time.Hour)
	}
	return sixPM
}

func resolveWorkExpiration(now time.Time, postMessage bool, until string) (time.Time, error) {
	if strings.TrimSpace(until) != "" {
		return parseUntilTime(now, until)
	}
	if postMessage {
		return now.Add(9 * time.Hour), nil
	}
	return todaySixPMAt(now), nil
}

func resolveLunchReturnTime(now time.Time, until string) (time.Time, error) {
	if strings.TrimSpace(until) != "" {
		return parseUntilTime(now, until)
	}
	return now.Add(1 * time.Hour), nil
}

func parseUntilTime(now time.Time, value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("--until cannot be empty")
	}

	if parsed, err := time.ParseInLocation("15:04", value, now.Location()); err == nil {
		candidate := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())
		if !candidate.After(now) {
			candidate = candidate.Add(24 * time.Hour)
		}
		return candidate, nil
	}

	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02T15:04", time.RFC3339} {
		if parsed, err := time.ParseInLocation(layout, value, now.Location()); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid --until value %q; use HH:MM, YYYY-MM-DD HH:MM, YYYY-MM-DDTHH:MM, or RFC3339", value)
}

func formatFriendlyDateTime(target, now time.Time) string {
	day := target.Format("Mon Jan 2")
	if sameDay(target, now) {
		day = "Today"
	} else if sameDay(target, now.Add(24*time.Hour)) {
		day = "Tomorrow"
	}
	return fmt.Sprintf("%s, %s", day, target.Format("3:04 PM"))
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
