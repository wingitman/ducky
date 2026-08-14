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
	Up          string `toml:"up"`
	Down        string `toml:"down"`
	TabNext     string `toml:"tab_next"`
	TabPrev     string `toml:"tab_prev"`
	Start       string `toml:"start"`
	Stop        string `toml:"stop"`
	Remove      string `toml:"remove"`
	Inspect     string `toml:"inspect"`
	Logs        string `toml:"logs"`
	New         string `toml:"new"`
	Edit        string `toml:"edit"`
	OpenConfig  string `toml:"open_config"`
	Quit        string `toml:"quit"`
	ScrollUp    string `toml:"scroll_up"`
	ScrollDown  string `toml:"scroll_down"`
	Search      string `toml:"search"`
	ClearSearch string `toml:"clear_search"`
	Confirm     string `toml:"confirm"`
	Back        string `toml:"back"`
	PageUp      string `toml:"page_up"`
	PageDown    string `toml:"page_down"`
	Refresh     string `toml:"refresh"`
	OpenPreview string `toml:"open_preview"`
	Setup       string `toml:"setup"`
	Run         string `toml:"run"`
	Theme       string `toml:"theme"`
}

// Config is ducky's persisted configuration.
type Config struct {
	Keys    Keys    `toml:"keys"`
	Updates Updates `toml:"updates"`
	Themes  Themes  `toml:"themes"`
}

// Updates controls the asynchronous startup update check.
type Updates struct {
	DisableChecks bool   `toml:"disable_checks"`
	CurrentCommit string `toml:"current_commit"`
	Repository    string `toml:"repository"`
}

// Default returns ducky's default configuration.
func Default() Config {
	return Config{Keys: Keys{
		Up: "up", Down: "down", TabNext: "tab", TabPrev: "shift+tab",
		Start: "s", Stop: "x", Remove: "d", Inspect: "enter", Logs: "g",
		New: "n", Edit: "e", OpenConfig: "o", Quit: "q",
		ScrollUp: "ctrl+u", ScrollDown: "ctrl+d",
		Search: "/", ClearSearch: "esc", Confirm: "enter", Back: "esc",
		PageUp: "pgup", PageDown: "pgdown", Refresh: "r", OpenPreview: "E", Setup: "S",
		Run: "R", Theme: "T",
	}, Updates: Updates{Repository: "https://github.com/wingitman/ducky"}, Themes: Themes{
		ThemeName: "terminal",
		ThemeFile: defaultThemeFile(),
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
		_ = EnsureThemesFile(cfg)
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
	_ = EnsureThemesFile(cfg)
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
	if cfg.Keys.Search == "" {
		cfg.Keys.Search = d.Search
	}
	if cfg.Keys.ClearSearch == "" {
		cfg.Keys.ClearSearch = d.ClearSearch
	}
	if cfg.Keys.Confirm == "" {
		cfg.Keys.Confirm = d.Confirm
	}
	if cfg.Keys.Back == "" {
		cfg.Keys.Back = d.Back
	}
	if cfg.Keys.PageUp == "" {
		cfg.Keys.PageUp = d.PageUp
	}
	if cfg.Keys.PageDown == "" {
		cfg.Keys.PageDown = d.PageDown
	}
	if cfg.Keys.Refresh == "" {
		cfg.Keys.Refresh = d.Refresh
	}
	if cfg.Keys.OpenPreview == "" {
		cfg.Keys.OpenPreview = d.OpenPreview
	}
	if cfg.Keys.Setup == "" {
		cfg.Keys.Setup = d.Setup
	}
	if cfg.Keys.Run == "" {
		cfg.Keys.Run = d.Run
	}
	if cfg.Keys.Theme == "" {
		cfg.Keys.Theme = d.Theme
	}
	if cfg.Updates.Repository == "" {
		cfg.Updates.Repository = Default().Updates.Repository
	}
	if cfg.Themes.ThemeName == "" {
		cfg.Themes.ThemeName = Default().Themes.ThemeName
	}
	if cfg.Themes.ThemeFile == "" {
		cfg.Themes.ThemeFile = Default().Themes.ThemeFile
	}
}

func render(cfg Config) string {
	k := cfg.Keys
	u := cfg.Updates
	return "# ducky configuration file\n# Key values use Bubble Tea names such as up, enter, ctrl+u, or single letters.\n\n[keys]\n" +
		fmt.Sprintf("up = %q\ndown = %q\ntab_next = %q\ntab_prev = %q\nstart = %q\nstop = %q\nremove = %q\ninspect = %q\nlogs = %q\nnew = %q\nedit = %q\nopen_config = %q\nquit = %q\nscroll_up = %q\nscroll_down = %q\nsearch = %q\nclear_search = %q\nconfirm = %q\nback = %q\npage_up = %q\npage_down = %q\nrefresh = %q\nopen_preview = %q\nsetup = %q\nrun = %q\ntheme = %q\n\n[updates]\ndisable_checks = %t\ncurrent_commit = %q\nrepository = %q\n\n[themes]\ntheme_name = %q\ntheme_file = %q\n# Optional overrides applied after the selected theme.\n# primary = \"#7C9EF0\"\n# accent = \"#F0A47C\"\n# muted = \"#666688\"\n# error = \"#F07C7C\"\n# success = \"#7CF09C\"\n# border = \"#444466\"\n# selected_background = \"#2A2A4A\"\n# selected_foreground = \"#EEEEFF\"\n", k.Up, k.Down, k.TabNext, k.TabPrev, k.Start, k.Stop, k.Remove, k.Inspect, k.Logs, k.New, k.Edit, k.OpenConfig, k.Quit, k.ScrollUp, k.ScrollDown, k.Search, k.ClearSearch, k.Confirm, k.Back, k.PageUp, k.PageDown, k.Refresh, k.OpenPreview, k.Setup, k.Run, k.Theme, u.DisableChecks, u.CurrentCommit, u.Repository, cfg.Themes.ThemeName, cfg.Themes.ThemeFile)
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
		"inspect", "logs", "new", "edit", "open_config", "quit", "scroll_up", "scroll_down", "search", "clear_search", "confirm", "back", "page_up", "page_down", "refresh", "open_preview", "setup", "run", "theme",
	}
	for _, key := range []string{"disable_checks", "current_commit", "repository", "theme_name", "theme_file"} {
		if !strings.Contains(content, key+" =") {
			return true
		}
	}
	for _, key := range keys {
		if !strings.Contains(content, key+" =") {
			return true
		}
	}
	return cfg.Keys.OpenConfig == ""
}
