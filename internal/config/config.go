// Package config loads sidekick's TOML configuration and resolves its directories.
//
// sidekick uses XDG-style paths on every platform, macOS included, so the same
// dotfiles (e.g. managed by chezmoi) work everywhere:
//
//	config: $XDG_CONFIG_HOME/sidekick or ~/.config/sidekick
//	data:   $XDG_DATA_HOME/sidekick   or ~/.local/share/sidekick
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const appName = "sidekick"

// Config is the user configuration read from config.toml.
type Config struct {
	Theme string     `toml:"theme"`
	Sync  SyncConfig `toml:"sync"`
}

// SyncConfig selects how the data directory is synchronized between machines.
type SyncConfig struct {
	// Backend is one of "git", "folder" or "none".
	Backend string `toml:"backend"`
}

// Default returns the configuration used when no file exists.
func Default() Config {
	return Config{
		Theme: "default",
		Sync:  SyncConfig{Backend: "none"},
	}
}

// Load reads the config file at path, or the default location when path is empty.
// A missing file is not an error: defaults are returned.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		dir, err := ConfigDir()
		if err != nil {
			return cfg, err
		}
		path = filepath.Join(dir, "config.toml")
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading config %s: %w", path, err)
	}
	return cfg, nil
}

// ConfigDir returns the directory holding config.toml.
func ConfigDir() (string, error) {
	return xdgDir("XDG_CONFIG_HOME", ".config")
}

// DataDir returns the directory holding notes, runbooks, connections and history.
func DataDir() (string, error) {
	return xdgDir("XDG_DATA_HOME", filepath.Join(".local", "share"))
}

func xdgDir(env, fallback string) (string, error) {
	if base := os.Getenv(env); base != "" {
		return filepath.Join(base, appName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, fallback, appName), nil
}
