// Package ui contains ducky's shared visual language.
package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

var (
	ColorPrimary = lipgloss.Color("#7C9EF0")
	ColorAccent  = lipgloss.Color("#F0A47C")
	ColorGreen   = lipgloss.Color("#7CF09C")
	ColorRed     = lipgloss.Color("#F07C7C")
	ColorMuted   = lipgloss.Color("#666688")
	ColorSubtle  = lipgloss.Color("#444466")
	ColorHeader  = lipgloss.Color("#EEEEFF")
	ColorSelect  = lipgloss.Color("#2A2A4A")

	Header       = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	Muted        = lipgloss.NewStyle().Foreground(ColorMuted)
	Error        = lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
	Success      = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	Selected     = lipgloss.NewStyle().Background(ColorSelect).Foreground(ColorHeader).Bold(true)
	Panel        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorSubtle)
	FocusedPanel = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorPrimary)
	Key          = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	BrandDelby   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	BrandSoft    = lipgloss.NewStyle().Foreground(lipgloss.Color("#5865F2")).Bold(true)
	Selector     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
)

// ConfigureTheme applies a complete semantic palette. Terminal mode omits
// explicit colors so the terminal's normal foreground and background inherit.
func ConfigureTheme(colors map[string]string, terminal bool) {
	ColorPrimary = themedColor(colors, terminal, "primary", "#7C9EF0")
	ColorAccent = themedColor(colors, terminal, "accent", "#F0A47C")
	ColorGreen = themedColor(colors, terminal, "success", "#7CF09C")
	ColorRed = themedColor(colors, terminal, "error", "#F07C7C")
	ColorMuted = themedColor(colors, terminal, "muted", "#666688")
	ColorSubtle = themedColor(colors, terminal, "border", "#444466")
	ColorHeader = themedColor(colors, terminal, "selected_foreground", "#EEEEFF")
	ColorSelect = themedColor(colors, terminal, "selected_background", "#2A2A4A")

	Header = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "primary", "#7C9EF0")
	Muted = themedStyle(lipgloss.NewStyle(), colors, terminal, "muted", "#666688")
	Error = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "error", "#F07C7C")
	Success = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "success", "#7CF09C")
	Selected = themedBackground(themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "selected_foreground", "#EEEEFF"), colors, terminal, "selected_background", "#2A2A4A")
	Panel = themedBorder(lipgloss.NewStyle().Border(lipgloss.RoundedBorder()), colors, terminal, "border", "#444466")
	FocusedPanel = themedBorder(lipgloss.NewStyle().Border(lipgloss.RoundedBorder()), colors, terminal, "primary", "#7C9EF0")
	Key = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "accent", "#F0A47C")
	BrandDelby = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "brand_primary", "#FFFFFF")
	BrandSoft = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "brand_secondary", "#5865F2")
	Selector = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "selector", "#FFFFFF")
}

func themedColor(colors map[string]string, terminal bool, key, fallback string) color.Color {
	if value, ok := themedValue(colors, terminal, key, fallback); ok {
		return lipgloss.Color(value)
	}
	return nil
}

func themedStyle(style lipgloss.Style, colors map[string]string, terminal bool, key, fallback string) lipgloss.Style {
	if value, ok := themedValue(colors, terminal, key, fallback); ok {
		return style.Foreground(lipgloss.Color(value))
	}
	return style
}

func themedBackground(style lipgloss.Style, colors map[string]string, terminal bool, key, fallback string) lipgloss.Style {
	if value, ok := themedValue(colors, terminal, key, fallback); ok {
		return style.Background(lipgloss.Color(value))
	}
	return style
}

func themedBorder(style lipgloss.Style, colors map[string]string, terminal bool, key, fallback string) lipgloss.Style {
	if value, ok := themedValue(colors, terminal, key, fallback); ok {
		return style.BorderForeground(lipgloss.Color(value))
	}
	return style
}

func themedValue(colors map[string]string, terminal bool, key, fallback string) (string, bool) {
	if value := colors[key]; value != "" {
		return value, true
	}
	if terminal {
		return "", false
	}
	return fallback, fallback != ""
}
