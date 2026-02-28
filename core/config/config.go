package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the global Enveil configuration
type Config struct {
	VaultPath      string `json:"vault_path"`
	Salt           string `json:"salt"`
	ActiveProject  string `json:"active_project,omitempty"`
	ActiveEnv      string `json:"active_env,omitempty"`
}

// enveilDir returns the path to the ~/.enveil director
func enveilDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("error getting home directory: %w", err)
	}
	return filepath.Join(home, ".enveil"), nil
}

// ConfigPath returns the full path to the config file
func ConfigPath() (string, error) {
	dir, err := enveilDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// DefaultVaultPath returns the full path to the vault
func DefaultVaultPath() (string, error) {
	dir, err := enveilDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vault.db"), nil
}

// Load reads the config from disk. Returns an empty config if it does not exist
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return &cfg, nil
}

//Save writes the config to disk
func (c *Config) Save() error {
	dir, err := enveilDir()
	if err != nil {
		return err
	}

	// Create ~/.enveil if it does not exist
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("error creating enveil directory: %w", err)
	}

	path, err := ConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializing config: %w", err)
	}

	// 0600 means only the current user can read and write the file
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("error saving config: %w", err)
	}

	return nil
}

// IsInitialized returns whether Enveil has been set up for the first time
func (c *Config) IsInitialized() bool {
	return c.VaultPath != "" && c.Salt != ""
}