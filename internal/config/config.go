package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds user preferences persisted across sessions.
type Config struct {
	Theme string `json:"theme"` // base16 | tech | charm | dracula | catppuccin
	Dir   string `json:"dir"`   // default download directory
}

// Themes lists all valid theme names.
var Themes = []string{"base16", "tech", "charm", "dracula", "catppuccin"}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "muze", "config.json"), nil
}

// Load reads the config file. Returns defaults if the file does not exist.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return &Config{Theme: "base16"}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Theme: "base16"}, nil
		}
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return &Config{Theme: "base16"}, nil
	}
	if c.Theme == "" {
		c.Theme = "base16"
	}
	return &c, nil
}

// Save writes the config file, creating parent directories as needed.
func Save(c *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
