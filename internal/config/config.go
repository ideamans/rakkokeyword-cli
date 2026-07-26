// Package config stores the API key and output defaults on disk.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	appDirName   = "rakkokeyword-cli"
	fileName     = "config.json"
	envConfigDir = "RAKKOKEYWORD_CLI_HOME"

	// EnvAPIKey is the documented environment variable for the API key.
	EnvAPIKey = "RAKKOKEYWORD_API_KEY"
	// EnvAPIKeyAlt is a shorter alias accepted for convenience.
	EnvAPIKeyAlt = "RAKKO_API_KEY"
	// EnvBaseURL overrides the API base URL (tests, staging).
	EnvBaseURL = "RAKKOKEYWORD_API_BASE"
)

// Config is the on-disk configuration file.
type Config struct {
	APIKey        string `json:"api_key"`
	DefaultFormat string `json:"default_format"`
	BaseURL       string `json:"base_url,omitempty"`
}

// Default returns the configuration used when no file exists.
func Default() *Config {
	return &Config{DefaultFormat: "table"}
}

// Dir returns the config directory.
// Priority: RAKKOKEYWORD_CLI_HOME > $XDG_CONFIG_HOME/rakkokeyword-cli >
// ~/.config/rakkokeyword-cli. On macOS this uses ~/.config (not ~/Library) to
// match the CLI convention.
func Dir() string {
	if d := os.Getenv(envConfigDir); d != "" {
		return d
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, appDirName)
}

// Path returns the config file path.
func Path() string { return filepath.Join(Dir(), fileName) }

// Load reads the config file, returning defaults when it does not exist.
func Load() (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes the config file with owner-only permissions; it holds a secret.
func Save(cfg *Config) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), data, 0o600)
}

// APIKeyResolved returns the effective API key.
// Priority: flag > RAKKOKEYWORD_API_KEY > RAKKO_API_KEY > config file.
func (c *Config) APIKeyResolved(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv(EnvAPIKey); env != "" {
		return env
	}
	if env := os.Getenv(EnvAPIKeyAlt); env != "" {
		return env
	}
	return c.APIKey
}

// APIKeySource names where APIKeyResolved would take the key from, for
// `rakkokeyword auth status`. It never returns the key itself.
func (c *Config) APIKeySource(flagValue string) string {
	switch {
	case flagValue != "":
		return "--api-key"
	case os.Getenv(EnvAPIKey) != "":
		return EnvAPIKey + " env"
	case os.Getenv(EnvAPIKeyAlt) != "":
		return EnvAPIKeyAlt + " env"
	case c.APIKey != "":
		return "config file"
	}
	return ""
}

// BaseURLResolved returns the effective API base URL.
// Priority: RAKKOKEYWORD_API_BASE env > config file > the production API.
func (c *Config) BaseURLResolved() string {
	if env := os.Getenv(EnvBaseURL); env != "" {
		return env
	}
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return "https://api.rakkokeyword.com"
}
