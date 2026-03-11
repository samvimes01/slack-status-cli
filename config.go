package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	Token string `json:"token"`
}

type Paths struct {
	ConfigFile string
	PIDFile    string
	StateFile  string
}

func ResolvePaths() Paths {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}

	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		home, _ := os.UserHomeDir()
		stateHome = filepath.Join(home, ".local", "state")
	}

	return Paths{
		ConfigFile: filepath.Join(configHome, "slack-status", "config.json"),
		PIDFile:    filepath.Join(stateHome, "slack-status", "worker.pid"),
		StateFile:  filepath.Join(stateHome, "slack-status", "status.json"),
	}
}

type LocalState struct {
	CurrentStatus       StateStatus `json:"current_status"`
	WorkerScheduled     bool        `json:"worker_scheduled"`
	WorkerPID           int         `json:"worker_pid,omitempty"`
	StartAvailableToday bool        `json:"start_available_today"`
	LastStartAt         string      `json:"last_start_at,omitempty"`
	LastStartDay        string      `json:"last_start_day,omitempty"`
	UpdatedAt           string      `json:"updated_at"`
}

type StateStatus struct {
	Command         string `json:"command"`
	Text            string `json:"text"`
	Emoji           string `json:"emoji"`
	StatusExpiresAt string `json:"status_expires_at,omitempty"`
	WillReturnTo    string `json:"will_return_to,omitempty"`
	Source          string `json:"source"`
}

func LoadLocalState(paths Paths) (*LocalState, error) {
	data, err := os.ReadFile(paths.StateFile)
	if err != nil {
		return nil, fmt.Errorf("reading state %s: %w", paths.StateFile, err)
	}

	var state LocalState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}

	return &state, nil
}

func SaveLocalState(paths Paths, state LocalState) error {
	if err := os.MkdirAll(filepath.Dir(paths.StateFile), 0o700); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}
	return os.WriteFile(paths.StateFile, data, 0o600)
}

func SaveConfig(paths Paths, token string) error {
	if err := os.MkdirAll(filepath.Dir(paths.ConfigFile), 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.Marshal(Config{Token: token})
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	return os.WriteFile(paths.ConfigFile, data, 0o600)
}

func withDerivedState(state LocalState, now time.Time) LocalState {
	state.StartAvailableToday = startAvailableToday(state, now)
	return state

}

func startAvailableToday(state LocalState, now time.Time) bool {
	return state.LastStartDay != localDayStamp(now)
}

func localDayStamp(now time.Time) string {
	return now.Format("2006-01-02")
}

func LoadConfig(paths Paths) (*Config, error) {
	data, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", paths.ConfigFile, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Token == "" {
		return nil, fmt.Errorf("config missing token field")
	}

	return &cfg, nil
}
