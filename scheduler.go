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

func SpawnReturnWorker(binaryPath, pidPath string) error {
	cmd := exec.Command(binaryPath, "_return-worker")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // detach from terminal
	}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawning worker: %w", err)
	}

	if err := writePID(pidPath, cmd.Process.Pid); err != nil {
		// Best-effort kill if we can't record the PID
		cmd.Process.Kill()
		return err
	}

	return nil
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

func RunReturnWorker(token, pidPath string) {
	time.Sleep(1 * time.Hour)

	expiration := todaySixPM()
	if err := SetStatus(token, "Working remotely", ":computer:", expiration); err != nil {
		fmt.Fprintf(os.Stderr, "return-worker: set status: %v\n", err)
	}

	os.Remove(pidPath)
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
