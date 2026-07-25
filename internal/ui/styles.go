// Package ui contains ducky's shared visual language.
package ui

import "charm.land/lipgloss/v2"

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
)
