package main

import (
	"testing"
	"time"
)

// Regression: the worker must be able to parse the return time from the exact
// args it is spawned with. Previously the --until flag was placed after the
// _return-worker subcommand, so the flag parser stopped before reading it and
// the worker exited immediately without restoring the status.
func TestLoadWorkerReturnTime_WorkerArgOrder(t *testing.T) {
	returnAt := time.Now().Add(time.Hour).Truncate(time.Second)
	args := workerArgs(returnAt)

	got, err := loadWorkerReturnTime(args)
	if err != nil {
		t.Fatalf("loadWorkerReturnTime(%v) returned error: %v", args, err)
	}
	if !got.Equal(returnAt) {
		t.Fatalf("got return time %v, want %v", got, returnAt)
	}
}

// The lunch return worker must restore work status to the day-end recorded by
// start/work, not a hardcoded 6pm. A 10am start (→ 7pm end) followed by lunch
// should leave work status expiring at 7pm.
func TestResolveWorkDayEnd(t *testing.T) {
	now := time.Date(2026, 6, 24, 12, 30, 0, 0, time.Local)

	t.Run("uses stored future day-end", func(t *testing.T) {
		end := time.Date(2026, 6, 24, 19, 0, 0, 0, time.Local)
		state := LocalState{WorkDayEndsAt: end.Format(time.RFC3339)}
		if got := resolveWorkDayEnd(state, now); got != end.Unix() {
			t.Fatalf("got %d, want %d (%s)", got, end.Unix(), end)
		}
	})

	t.Run("falls back to 6pm when unset", func(t *testing.T) {
		want := todaySixPMAt(now).Unix()
		if got := resolveWorkDayEnd(LocalState{}, now); got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
	})

	t.Run("falls back to 6pm when stored end is in the past", func(t *testing.T) {
		past := now.Add(-2 * time.Hour)
		state := LocalState{WorkDayEndsAt: past.Format(time.RFC3339)}
		want := todaySixPMAt(now).Unix()
		if got := resolveWorkDayEnd(state, now); got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
	})
}
