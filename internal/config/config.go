package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const defaultConfigPath = "/app/www/config.json"

// Config matches the current JSON schema used by the charm.
type Config struct {
	Site          string            `json:"site"`
	Archive       string            `json:"archive"`
	PublicHTMLDir string            `json:"public_html_dir"`
	IndexDir      string            `json:"index_dir"`
	Releases      map[string]string `json:"releases"`
	Repos         []string          `json:"repos"`
	Arch          string            `json:"arch"`
}

func DefaultPath() string {
	if path := os.Getenv("MANPAGES_CONFIG_FILE"); path != "" {
		return path
	}
	return defaultConfigPath
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Site == "" {
		return errors.New("config site is required")
	}
	if c.Archive == "" {
		return errors.New("config archive is required")
	}
	if c.PublicHTMLDir == "" {
		return errors.New("config public_html_dir is required")
	}
	if len(c.Releases) == 0 {
		return errors.New("config releases is required")
	}
	if len(c.Repos) == 0 {
		return errors.New("config repos is required")
	}
	if c.Arch == "" {
		return errors.New("config arch is required")
	}
	return nil
}

func (c *Config) IndexPath() string {
	if c.IndexDir != "" {
		return c.IndexDir
	}
	return filepath.Join(c.PublicHTMLDir, "search.db")
}

func (c *Config) SiteURL() string {
	return strings.TrimRight(c.Site, "/")
}

func (c *Config) ReleaseKeys() []string {
	keys := make([]string, 0, len(c.Releases))
	for release := range c.Releases {
		keys = append(keys, release)
	}
	sort.Strings(keys)
	return keys
}
