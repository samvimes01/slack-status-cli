package main

import (
	"testing"
	"time"
)

// Regression: flags must be honored regardless of whether they appear before or
// after the subcommand. Previously parseOptions used the flag package directly
// on the raw args, so the parser stopped at the first non-flag token (the
// command) and silently ignored any flag placed after it — e.g.
// "work --until 16:47" set the default 6pm instead of 16:47.
func TestParseOptions_FlagsAfterCommand(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		wantCmd   string
		wantUntil string
		wantJSON  bool
	}{
		{"until after command", []string{"work", "--until", "16:47"}, "work", "16:47", false},
		{"until before command", []string{"--until", "16:47", "work"}, "work", "16:47", false},
		{"until equals form after command", []string{"work", "--until=16:47"}, "work", "16:47", false},
		{"json after command", []string{"work", "--json"}, "work", "", true},
		{"json before command", []string{"--json", "work"}, "work", "", true},
		{"json and until after command", []string{"work", "--json", "--until", "16:47"}, "work", "16:47", true},
		{"command only", []string{"work"}, "work", "", false},
		{"no args", []string{}, "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseOptions(tc.args)
			if err != nil {
				t.Fatalf("parseOptions(%v) error: %v", tc.args, err)
			}
			if got.Command != tc.wantCmd {
				t.Errorf("Command = %q, want %q", got.Command, tc.wantCmd)
			}
			if got.Until != tc.wantUntil {
				t.Errorf("Until = %q, want %q", got.Until, tc.wantUntil)
			}
			if got.JSON != tc.wantJSON {
				t.Errorf("JSON = %v, want %v", got.JSON, tc.wantJSON)
			}
		})
	}
}

// The work command should honor a stored future work_day_ends_at as its default
// expiration (matching the lunch auto-return), falling back to 6pm only when no
// usable end is stored. An explicit --until always wins.
func TestResolveWorkExpiration(t *testing.T) {
	now := time.Date(2026, 6, 24, 12, 30, 0, 0, time.Local)

	t.Run("honors stored future day-end when no --until", func(t *testing.T) {
		end := time.Date(2026, 6, 24, 19, 0, 0, 0, time.Local)
		state := LocalState{WorkDayEndsAt: end.Format(time.RFC3339)}
		got, err := resolveWorkExpiration(now, false, "", state)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if got.Unix() != end.Unix() {
			t.Fatalf("got %s, want %s", got, end)
		}
	})

	t.Run("falls back to 6pm when no stored end", func(t *testing.T) {
		got, err := resolveWorkExpiration(now, false, "", LocalState{})
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if got.Unix() != todaySixPMAt(now).Unix() {
			t.Fatalf("got %s, want %s", got, todaySixPMAt(now))
		}
	})

	t.Run("explicit --until overrides stored end", func(t *testing.T) {
		end := time.Date(2026, 6, 24, 19, 0, 0, 0, time.Local)
		state := LocalState{WorkDayEndsAt: end.Format(time.RFC3339)}
		got, err := resolveWorkExpiration(now, false, "16:47", state)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		want := time.Date(2026, 6, 24, 16, 47, 0, 0, time.Local)
		if got.Unix() != want.Unix() {
			t.Fatalf("got %s, want %s", got, want)
		}
	})

	t.Run("start uses 9h regardless of stored end", func(t *testing.T) {
		end := time.Date(2026, 6, 24, 19, 0, 0, 0, time.Local)
		state := LocalState{WorkDayEndsAt: end.Format(time.RFC3339)}
		got, err := resolveWorkExpiration(now, true, "", state)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if got.Unix() != now.Add(9*time.Hour).Unix() {
			t.Fatalf("got %s, want %s", got, now.Add(9*time.Hour))
		}
	})
}
