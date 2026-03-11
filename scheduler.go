package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func SpawnReturnWorker(binaryPath, pidPath string, returnAt time.Time) (int, error) {
	cmd := exec.Command(binaryPath, "_return-worker", "--until", returnAt.Format(time.RFC3339))
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // detach from terminal
	}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("spawning worker: %w", err)
	}

	if err := writePID(pidPath, cmd.Process.Pid); err != nil {
		// Best-effort kill if we can't record the PID
		cmd.Process.Kill()
		return 0, err
	}

	return cmd.Process.Pid, nil
}

func KillWorker(pidPath string) {
	pid, err := readPID(pidPath)
	if err != nil {
		return // no worker running
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidPath)
		return
	}

	proc.Signal(syscall.SIGTERM)
	os.Remove(pidPath)
}

func RunReturnWorker(token string, paths Paths) {
	returnAt, err := loadWorkerReturnTime(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "return-worker: resolve return time: %v\n", err)
		os.Remove(paths.PIDFile)
		return
	}

	if sleepDuration := time.Until(returnAt); sleepDuration > 0 {
		time.Sleep(sleepDuration)
	}

	expiration := todaySixPM()
	if err := SetStatus(token, "Working remotely", ":computer:", expiration); err != nil {
		fmt.Fprintf(os.Stderr, "return-worker: set status: %v\n", err)
	} else {
		priorState := preservedStartState(paths)
		state := withDerivedState(LocalState{
			CurrentStatus: workingStatusState("worker", expiration),
			LastStartAt:   priorState.LastStartAt,
			LastStartDay:  priorState.LastStartDay,
			UpdatedAt:     time.Now().Format(time.RFC3339),
		}, time.Now())
		if err := SaveLocalState(paths, state); err != nil {
			fmt.Fprintf(os.Stderr, "return-worker: save state: %v\n", err)
		}
	}

	os.Remove(paths.PIDFile)
}

func loadWorkerReturnTime(args []string) (time.Time, error) {
	options, err := parseOptions(args)
	if err != nil {
		return time.Time{}, err
	}
	if strings.TrimSpace(options.Until) == "" {
		return time.Time{}, fmt.Errorf("missing --until return timestamp")
	}
	return parseUntilTime(time.Now(), options.Until)
}

func writePID(pidPath string, pid int) error {
	if err := os.MkdirAll(filepath.Dir(pidPath), 0o700); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}
	return os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0o600)
}

func readPID(pidPath string) (int, error) {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}
