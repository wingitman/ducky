package app

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/wingitman/ducky/internal/config"
	"github.com/wingitman/ducky/internal/runtime"
)

func TestRenderStaysWithinTerminalBounds(t *testing.T) {
	m := model{
		client:         runtime.Client{Kind: runtime.Docker, Bin: "docker"},
		cfg:            config.Default(),
		width:          48,
		height:         14,
		output:         strings.Repeat("inspect output\n", 100),
		outputViewport: viewport.New(),
	}
	m.outputViewport.SetContent(m.output)
	m.resize()
	view := m.render()
	if got := lipgloss.Width(view); got > m.width {
		t.Fatalf("render width = %d, terminal width = %d", got, m.width)
	}
	if got := lipgloss.Height(view); got > m.height {
		t.Fatalf("render height = %d, terminal height = %d", got, m.height)
	}
}

func TestHintsUseConfiguredKeys(t *testing.T) {
	cfg := config.Default()
	cfg.Keys.OpenConfig = "z"
	m := model{cfg: cfg, width: 160, tab: int(runtime.Containers)}
	hints := m.renderHints()
	if !strings.Contains(hints, "z") {
		t.Fatalf("hints do not contain configured config key: %q", hints)
	}
}
