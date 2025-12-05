package config

import (
	"os"
	"path/filepath"
)

const (
	AppName              = "statping"
	DefaultCheckInterval = 60
	DefaultTimeout       = 10
	DefaultMaxFailures   = 3
	NotificationCooldown = 300
)

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(home, ".config", AppName)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return configDir, nil
}

func GetDatabasePath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "statping.db"), nil
}
