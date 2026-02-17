package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Token string `json:"token"`
}

type Paths struct {
	ConfigFile string
	PIDFile    string
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
	}
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
