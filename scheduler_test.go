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
