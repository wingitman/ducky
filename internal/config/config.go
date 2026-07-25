// Package config manages ducky's user configuration and keybindings.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Keys contains every configurable ducky action.
type Keys struct {
	Up         string `toml:"up"`
	Down       string `toml:"down"`
	TabNext    string `toml:"tab_next"`
	TabPrev    string `toml:"tab_prev"`
	Start      string `toml:"start"`
	Stop       string `toml:"stop"`
	Remove     string `toml:"remove"`
	Inspect    string `toml:"inspect"`
	Logs       string `toml:"logs"`
	New        string `toml:"new"`
	Edit       string `toml:"edit"`
	OpenConfig string `toml:"open_config"`
	Quit       string `toml:"quit"`
	ScrollUp   string `toml:"scroll_up"`
	ScrollDown string `toml:"scroll_down"`
}

// Config is ducky's persisted configuration.
type Config struct {
	Keys Keys `toml:"keys"`
}

// Default returns ducky's default configuration.
func Default() Config {
	return Config{Keys: Keys{
		Up: "up", Down: "down", TabNext: "tab", TabPrev: "shift+tab",
		Start: "s", Stop: "x", Remove: "d", Inspect: "enter", Logs: "g",
		New: "n", Edit: "e", OpenConfig: "o", Quit: "q",
		ScrollUp: "ctrl+u", ScrollDown: "ctrl+d",
	}}
}

// Path returns the platform-appropriate ducky config path.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "delbysoft", "ducky.toml"), nil
}

// Load reads the config, creating a fully populated default when needed.
func Load() (Config, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := Save(cfg); err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	if err := errIfUnexpectedStat(err); err != nil {
		return cfg, err
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("decode config: %w", err)
	}
	applyDefaults(&cfg)
	if data, readErr := os.ReadFile(path); readErr == nil && missingKeys(string(data), cfg) {
		_ = Save(cfg)
	}
	return cfg, nil
}

// Save writes the config while preserving a readable commented layout.
func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(render(cfg)), 0o644)
}

func applyDefaults(cfg *Config) {
	d := Default().Keys
	if cfg.Keys.Up == "" {
		cfg.Keys.Up = d.Up
	}
	if cfg.Keys.Down == "" {
		cfg.Keys.Down = d.Down
	}
	if cfg.Keys.TabNext == "" {
		cfg.Keys.TabNext = d.TabNext
	}
	if cfg.Keys.TabPrev == "" {
		cfg.Keys.TabPrev = d.TabPrev
	}
	if cfg.Keys.Start == "" {
		cfg.Keys.Start = d.Start
	}
	if cfg.Keys.Stop == "" {
		cfg.Keys.Stop = d.Stop
	}
	if cfg.Keys.Remove == "" {
		cfg.Keys.Remove = d.Remove
	}
	if cfg.Keys.Inspect == "" {
		cfg.Keys.Inspect = d.Inspect
	}
	if cfg.Keys.Logs == "" {
		cfg.Keys.Logs = d.Logs
	}
	if cfg.Keys.New == "" {
		cfg.Keys.New = d.New
	}
	if cfg.Keys.Edit == "" {
		cfg.Keys.Edit = d.Edit
	}
	if cfg.Keys.OpenConfig == "" {
		cfg.Keys.OpenConfig = d.OpenConfig
	}
	if cfg.Keys.Quit == "" {
		cfg.Keys.Quit = d.Quit
	}
	if cfg.Keys.ScrollUp == "" {
		cfg.Keys.ScrollUp = d.ScrollUp
	}
	if cfg.Keys.ScrollDown == "" {
		cfg.Keys.ScrollDown = d.ScrollDown
	}
}

func render(cfg Config) string {
	k := cfg.Keys
	return "# ducky configuration file\n# Key values use Bubble Tea names such as up, enter, ctrl+u, or single letters.\n\n[keys]\n" +
		fmt.Sprintf("up = %q\ndown = %q\ntab_next = %q\ntab_prev = %q\nstart = %q\nstop = %q\nremove = %q\ninspect = %q\nlogs = %q\nnew = %q\nedit = %q\nopen_config = %q\nquit = %q\nscroll_up = %q\nscroll_down = %q\n", k.Up, k.Down, k.TabNext, k.TabPrev, k.Start, k.Stop, k.Remove, k.Inspect, k.Logs, k.New, k.Edit, k.OpenConfig, k.Quit, k.ScrollUp, k.ScrollDown)
}

func errIfUnexpectedStat(err error) error {
	if err != nil {
		return fmt.Errorf("stat config: %w", err)
	}
	return nil
}

func missingKeys(content string, cfg Config) bool {
	keys := []string{
		"up", "down", "tab_next", "tab_prev", "start", "stop", "remove",
		"inspect", "logs", "new", "edit", "open_config", "quit", "scroll_up", "scroll_down",
	}
	for _, key := range keys {
		if !strings.Contains(content, key+" =") {
			return true
		}
	}
	return cfg.Keys.OpenConfig == ""
}
