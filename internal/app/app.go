// Package app implements ducky's Bubble Tea application model.
package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/wingitman/ducky/internal/config"
	"github.com/wingitman/ducky/internal/runtime"
	"github.com/wingitman/ducky/internal/ui"
)

const tabCount = 6

var tabNames = []string{"Containers", "Images", "Volumes", "Networks", "Compose", "Kubernetes"}

type model struct {
	client         runtime.Client
	cfg            config.Config
	tab            int
	cursor         int
	items          []runtime.Item
	output         string
	status         string
	loading        bool
	width          int
	height         int
	outputViewport viewport.Model
	prompt         *textinput.Model
}

type loadedMsg struct {
	items []runtime.Item
	err   error
}

type actionMsg struct {
	output string
	err    error
}

type configReloadedMsg struct {
	cfg config.Config
	err error
}

type createMsg struct {
	output string
	err    error
}

// New creates the root application model.
func New(client runtime.Client, cfg config.Config) tea.Model {
	return model{client: client, cfg: cfg, loading: true, outputViewport: viewport.New()}
}

func (m model) Init() tea.Cmd {
	return m.load()
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
	case loadedMsg:
		m.loading = false
		m.items = msg.items
		m.cursor = clamp(m.cursor, 0, len(m.items)-1)
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.status = fmt.Sprintf("%d %s", len(m.items), strings.ToLower(tabNames[m.tab]))
		}
	case actionMsg:
		m.loading = false
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.output = strings.TrimSpace(msg.output)
			m.outputViewport.SetContent(m.output)
			m.outputViewport.GotoTop()
			m.resize()
			m.status = "command completed"
		}
		return m, m.load()
	case createMsg:
		m.loading = false
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.output = strings.TrimSpace(msg.output)
			m.outputViewport.SetContent(m.output)
			m.outputViewport.GotoTop()
			m.status = "resource created"
			m.resize()
		}
		return m, m.load()
	case configReloadedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.cfg = msg.cfg
			m.status = "config reloaded"
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.prompt != nil {
			return m.updatePrompt(msg)
		}
		if m.output != "" {
			var cmd tea.Cmd
			m.outputViewport, cmd = m.outputViewport.Update(message)
			if m.isScrollKey(msg) {
				return m, cmd
			}
		}
		return m.updateKey(msg)
	}
	return m, nil
}

func (m model) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case matches(msg, m.cfg.Keys.Quit) || msg.String() == "ctrl+c" || msg.String() == "esc":
		return m, tea.Quit
	case matches(msg, m.cfg.Keys.TabNext):
		m.tab = (m.tab + 1) % tabCount
		m.cursor, m.output, m.loading = 0, "", true
		return m, m.load()
	case matches(msg, m.cfg.Keys.TabPrev):
		m.tab = (m.tab + tabCount - 1) % tabCount
		m.cursor, m.output, m.loading = 0, "", true
		return m, m.load()
	case matches(msg, m.cfg.Keys.Down):
		m.cursor = clamp(m.cursor+1, 0, len(m.items)-1)
	case matches(msg, m.cfg.Keys.Up):
		m.cursor = clamp(m.cursor-1, 0, len(m.items)-1)
	case msg.String() == "r":
		m.loading = true
		return m, m.load()
	case matches(msg, m.cfg.Keys.Start):
		return m, m.action("start")
	case matches(msg, m.cfg.Keys.Stop):
		return m, m.action("stop")
	case matches(msg, m.cfg.Keys.Remove):
		return m, m.action("remove")
	case matches(msg, m.cfg.Keys.Inspect):
		return m, m.action("inspect")
	case matches(msg, m.cfg.Keys.Logs):
		return m, m.logs()
	case matches(msg, m.cfg.Keys.New):
		return m, m.beginPrompt()
	case matches(msg, m.cfg.Keys.OpenConfig):
		return m, openConfig()
	case matches(msg, m.cfg.Keys.Edit) && runtime.Resource(m.tab) == runtime.Compose:
		return m, openComposeFile()
	}
	return m, nil
}

func (m model) load() tea.Cmd {
	return func() tea.Msg {
		items, err := m.client.List(context.Background(), runtime.Resource(m.tab))
		return loadedMsg{items: items, err: err}
	}
}

func (m model) action(action string) tea.Cmd {
	if len(m.items) == 0 || m.cursor >= len(m.items) {
		return nil
	}
	name := m.items[m.cursor].Name
	return func() tea.Msg {
		output, err := m.client.Action(context.Background(), runtime.Resource(m.tab), action, name)
		return actionMsg{output: output, err: err}
	}
}

func (m model) logs() tea.Cmd {
	if len(m.items) == 0 || m.cursor >= len(m.items) {
		return nil
	}
	name := m.items[m.cursor].Name
	return func() tea.Msg {
		output, err := m.client.Logs(context.Background(), runtime.Resource(m.tab), name)
		return actionMsg{output: output, err: err}
	}
}

func (m model) beginPrompt() tea.Cmd {
	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = m.promptPlaceholder()
	cmd := input.Focus()
	m.prompt = &input
	return cmd
}

func (m model) updatePrompt(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.prompt = nil
		return m, nil
	}
	if msg.String() == "enter" {
		value := m.prompt.Value()
		m.prompt = nil
		return m, func() tea.Msg {
			output, err := m.client.Create(context.Background(), runtime.Resource(m.tab), value)
			return createMsg{output: output, err: err}
		}
	}
	updated, cmd := m.prompt.Update(msg)
	m.prompt = &updated
	return m, cmd
}

func (m model) promptPlaceholder() string {
	switch runtime.Resource(m.tab) {
	case runtime.Containers:
		return "image[:tag], e.g. nginx:alpine"
	case runtime.Images:
		return "image[:tag], e.g. redis:7"
	case runtime.Volumes, runtime.Networks:
		return "new resource name"
	case runtime.Kubernetes:
		return "manifest path, e.g. deploy.yaml"
	default:
		return "value"
	}
}

func (m model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("Loading ducky...")
	}
	view := m.render()
	result := tea.NewView(view)
	result.AltScreen = true
	result.MouseMode = tea.MouseModeCellMotion
	return result
}

func (m model) render() string {
	brand := ui.BrandDelby.Render("delby") + ui.BrandSoft.Render("soft") + ui.Muted.Render(" / ducky / "+string(m.client.Kind))
	if lipgloss.Width(brand) > m.width {
		brand = ui.Header.Render("ducky")
	}
	tabs := make([]string, 0, len(tabNames))
	usedTabsWidth := 0
	for i, name := range tabNames {
		label := " " + name + " "
		separator := 0
		if usedTabsWidth > 0 {
			separator = 1
		}
		if usedTabsWidth+separator+lipgloss.Width(label) > m.width && i != m.tab {
			continue
		}
		if usedTabsWidth+separator+lipgloss.Width(label) > m.width {
			label = " " + fit(name, max(1, m.width-usedTabsWidth-separator-2)) + " "
		}
		style := ui.Muted
		if i == m.tab {
			style = ui.Selected
		}
		tabs = append(tabs, style.Render(label))
		usedTabsWidth += separator + lipgloss.Width(label)
	}
	tabBar := strings.Join(tabs, " ")

	rows := []string{ui.Header.Render(fit(fmt.Sprintf("%s  %s", tabNames[m.tab], m.status), max(1, m.width-2)))}
	if m.loading {
		rows = append(rows, ui.Muted.Render("Loading..."))
	} else if len(m.items) == 0 {
		rows = append(rows, ui.Muted.Render("No resources found."))
	} else {
		for i, item := range m.items {
			line := fit(fmt.Sprintf("%-32s %s", item.Name, item.Details), max(1, m.width-6))
			if i == m.cursor {
				line = ui.Selected.Render(line)
			}
			rows = append(rows, line)
		}
	}
	contentHeight := max(4, m.height-5)
	if m.prompt != nil {
		contentHeight = max(3, contentHeight-2)
	}
	promptView := ""
	if m.prompt != nil {
		promptView = ui.FocusedPanel.Width(max(1, m.width)).Render(m.prompt.View())
	}
	listHeight := contentHeight
	output := ""
	if m.output != "" {
		outputHeight := max(4, contentHeight/3)
		listHeight = max(3, contentHeight-outputHeight)
		output = ui.FocusedPanel.Width(max(1, m.width)).Height(outputHeight).Render(m.outputViewport.View())
	}
	if limit := max(1, listHeight-2); len(rows) > limit {
		rows = append(rows[:limit-1], ui.Muted.Render("... more items; scroll is available in a later view"))
	}
	list := ui.Panel.Width(max(1, m.width)).Height(listHeight).Render(strings.Join(rows, "\n"))
	hints := m.renderHints()
	divider := ui.Muted.Render(strings.Repeat("─", max(1, m.width)))
	return lipgloss.JoinVertical(lipgloss.Left, brand, tabBar, list, output, promptView, divider, hints)
}

func (m *model) resize() {
	if m.width < 1 || m.height < 1 {
		return
	}
	contentHeight := max(4, m.height-5)
	if m.prompt != nil {
		contentHeight = max(3, contentHeight-2)
	}
	outputHeight := 0
	if m.output != "" {
		outputHeight = max(4, contentHeight/3)
	}
	m.outputViewport.SetWidth(max(1, m.width-6))
	m.outputViewport.SetHeight(max(1, outputHeight-2))
}

func (m model) isScrollKey(msg tea.KeyPressMsg) bool {
	return matches(msg, m.cfg.Keys.ScrollUp) || matches(msg, m.cfg.Keys.ScrollDown) || msg.String() == "pgup" || msg.String() == "pgdown"
}

func (m model) renderHints() string {
	hint := func(key, description string) string {
		if key == "" {
			return ""
		}
		return key + ":" + description + "  "
	}
	common := hint(m.cfg.Keys.TabNext, "next") + hint(m.cfg.Keys.TabPrev, "previous") + hint(m.cfg.Keys.Up, "up") + hint(m.cfg.Keys.Down, "down")
	var actions string
	switch runtime.Resource(m.tab) {
	case runtime.Containers:
		actions = hint(m.cfg.Keys.New, "run") + hint(m.cfg.Keys.Start, "start") + hint(m.cfg.Keys.Stop, "stop") + hint(m.cfg.Keys.Remove, "remove") + hint(m.cfg.Keys.Inspect, "inspect") + hint(m.cfg.Keys.Logs, "logs")
	case runtime.Images:
		actions = hint(m.cfg.Keys.New, "pull/build") + hint(m.cfg.Keys.Remove, "remove") + hint(m.cfg.Keys.Inspect, "inspect")
	case runtime.Volumes, runtime.Networks:
		actions = hint(m.cfg.Keys.New, "create") + hint(m.cfg.Keys.Remove, "remove") + hint(m.cfg.Keys.Inspect, "inspect")
	case runtime.Compose:
		actions = hint(m.cfg.Keys.Start, "up") + hint(m.cfg.Keys.Stop, "stop") + hint(m.cfg.Keys.Remove, "down") + hint(m.cfg.Keys.Inspect, "config") + hint(m.cfg.Keys.Edit, "edit")
	case runtime.Kubernetes:
		actions = hint(m.cfg.Keys.New, "apply") + hint(m.cfg.Keys.Remove, "delete") + hint(m.cfg.Keys.Inspect, "inspect") + hint(m.cfg.Keys.Logs, "logs")
	}
	parts := strings.Fields(common + actions + hint(m.cfg.Keys.OpenConfig, "config") + hint(m.cfg.Keys.Quit, "quit"))
	line := ""
	for _, part := range parts {
		candidate := part
		if line != "" {
			candidate = line + "  " + part
		}
		if lipgloss.Width(candidate) > m.width {
			break
		}
		line = candidate
	}
	var rendered strings.Builder
	for _, part := range strings.Fields(line) {
		pieces := strings.SplitN(part, ":", 2)
		if len(pieces) == 2 {
			rendered.WriteString(ui.Key.Render(pieces[0]))
			rendered.WriteString(ui.Muted.Render(":" + pieces[1] + "  "))
		}
	}
	if m.output != "" {
		scrollHint := "  scroll output with " + m.cfg.Keys.ScrollUp + "/" + m.cfg.Keys.ScrollDown
		if lipgloss.Width(rendered.String())+lipgloss.Width(scrollHint) <= m.width {
			rendered.WriteString(ui.Muted.Render(scrollHint))
		}
	}
	return rendered.String()
}

func openConfig() tea.Cmd {
	path, err := config.Path()
	if err != nil {
		return func() tea.Msg { return configReloadedMsg{err: err} }
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "nano"
	}
	return tea.ExecProcess(exec.Command(editor, path), func(err error) tea.Msg {
		cfg, loadErr := config.Load()
		if err != nil && loadErr == nil {
			loadErr = err
		}
		return configReloadedMsg{cfg: cfg, err: loadErr}
	})
}

func openComposeFile() tea.Cmd {
	path := "compose.yaml"
	for _, candidate := range []string{"compose.yaml", "compose.yml", "docker-compose.yml", "docker-compose.yaml"} {
		if _, err := os.Stat(candidate); err == nil {
			path = candidate
			break
		}
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "nano"
	}
	return tea.ExecProcess(exec.Command(editor, path), func(err error) tea.Msg {
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{output: "Compose file updated"}
	})
}

func matches(msg tea.KeyPressMsg, binding string) bool {
	return binding != "" && msg.String() == binding
}

func fit(text string, width int) string {
	if width <= 0 || lipgloss.Width(text) <= width {
		return text
	}
	if width <= 3 {
		return text[:width]
	}
	return text[:width-3] + "..."
}

func clamp(value, low, high int) int {
	if high < low {
		return low
	}
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
