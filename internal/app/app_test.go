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
	view := m.View().Content
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

func TestHintsWrapInsteadOfDisappearing(t *testing.T) {
	m := model{cfg: config.Default(), width: 24, tab: int(runtime.Containers)}
	hints := m.renderHints()
	if hints == "" {
		t.Fatal("hints are empty at a narrow terminal width")
	}
	if lines := strings.Count(hints, "\n") + 1; lines > 2 {
		t.Fatalf("hint lines = %d, want at most 2", lines)
	}
}

func TestViewKeepsHintsInsideFooter(t *testing.T) {
	m := model{
		client: runtime.Client{Kind: runtime.Docker, Bin: "docker"},
		cfg:    config.Default(),
		width:  80,
		height: 20,
	}
	content := m.View().Content
	if !strings.Contains(content, "config") || !strings.Contains(content, "quit") {
		t.Fatalf("footer hints are missing: %q", content)
	}
	if got := lipgloss.Height(content); got > m.height {
		t.Fatalf("view height = %d, terminal height = %d", got, m.height)
	}
}

func TestMatchesFilterByFieldAndText(t *testing.T) {
	item := runtime.Item{Status: "running", Name: "chillblast-db", Image: "postgres:16", Ports: "5432/tcp"}
	if !matchesFilter(item, "status:running image:postgres") {
		t.Fatal("expected field filter to match")
	}
	if matchesFilter(item, "status:exited") {
		t.Fatal("unexpected status filter match")
	}
	if !matchesFilter(item, "chillblast") {
		t.Fatal("expected free-text filter to match")
	}
}
