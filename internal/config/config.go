package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds user-level configuration stored at ~/.skill-arena/config.json.
type Config struct {
	APIBaseURL   string `json:"api_base_url"`
	APIKey       string `json:"api_key"`
	DefaultModel string `json:"default_model"`
	LinterPath   string `json:"linter_path,omitempty"`
}

// ConfigPath returns the absolute path to the config file.
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".skill-arena", "config.json")
}

// Load reads the config file and returns the parsed config.
// If the file does not exist, it returns a default config.
func Load() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{
			APIBaseURL:   "https://api.anthropic.com",
			DefaultModel: "claude-sonnet-4-6",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}

// Save writes the config to disk, creating parent directories as needed.
func Save(cfg *Config) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}
