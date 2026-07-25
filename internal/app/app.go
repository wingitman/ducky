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
	"github.com/charmbracelet/x/ansi"
	"github.com/wingitman/ducky/internal/config"
	"github.com/wingitman/ducky/internal/projectfiles"
	"github.com/wingitman/ducky/internal/runtime"
	"github.com/wingitman/ducky/internal/ui"
	"github.com/wingitman/ducky/internal/update"
	"github.com/wingitman/ducky/internal/version"
)

const tabCount = 7

var tabNames = []string{"Containers", "Images", "Volumes", "Networks", "Compose", "Kubernetes", "Files"}

type model struct {
	client         runtime.Client
	cfg            config.Config
	tab            int
	cursor         int
	items          []runtime.Item
	allItems       []runtime.Item
	filter         string
	filterInput    textinput.Model
	searching      bool
	searchBefore   string
	output         string
	previewFocus   bool
	previewKind    string
	status         string
	loading        bool
	width          int
	height         int
	outputViewport viewport.Model
	prompt         *textinput.Model
	updateChecking bool
	runCh          <-chan runtime.RunEvent
	runCancel      context.CancelFunc
	running        bool
}

type loadedMsg struct {
	items []runtime.Item
	err   error
}

type actionMsg struct {
	output       string
	err          error
	kind         string
	reload       bool
	focusPreview bool
}

type runEventMsg struct {
	event runtime.RunEvent
}

type configReloadedMsg struct {
	cfg config.Config
	err error
}

type updateCheckedMsg struct {
	result update.Result
}

type createMsg struct {
	output string
	err    error
}

// New creates the root application model.
func New(client runtime.Client, cfg config.Config) tea.Model {
	filterInput := textinput.New()
	filterInput.Placeholder = "filter: status:running image:postgres name:db"
	filterInput.CharLimit = 256
	return model{client: client, cfg: cfg, loading: true, updateChecking: true, outputViewport: viewport.New(), filterInput: filterInput}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.load(), m.checkUpdates())
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
	case loadedMsg:
		m.loading = false
		m.allItems = msg.items
		m.applyFilter()
		m.cursor = clamp(m.cursor, 0, len(m.items)-1)
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.status = fmt.Sprintf("%d %s", len(m.items), strings.ToLower(tabNames[m.tab]))
		}
	case updateCheckedMsg:
		m.updateChecking = false
		if msg.result.Available {
			m.status = "update available: " + msg.result.Latest[:min(len(msg.result.Latest), 7)]
		}
	case actionMsg:
		m.loading = false
		if msg.err != nil {
			m.setOutput(msg.err.Error(), msg.kind)
			m.status = "command failed"
		} else {
			output := strings.TrimSpace(msg.output)
			if output == "" {
				output = "command completed successfully"
			}
			m.setOutput(output, msg.kind)
			m.status = "command completed"
		}
		if msg.focusPreview {
			m.previewFocus = true
		}
		if msg.reload && msg.err == nil {
			return m, m.load()
		}
		return m, nil
	case runEventMsg:
		if msg.event.Command != "" {
			m.setOutput(msg.event.Command, "text")
		}
		if msg.event.Output != "" {
			m.appendOutput(msg.event.Output)
		}
		if msg.event.Done {
			m.running = false
			m.loading = false
			m.runCh = nil
			m.runCancel = nil
			if msg.event.Err != nil {
				m.appendOutput(msg.event.Err.Error())
				m.status = "command failed"
				return m, nil
			}
			m.status = "command completed"
			return m, m.load()
		}
		return m, waitForRunEvent(m.runCh)
	case createMsg:
		m.loading = false
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.output = strings.TrimSpace(msg.output)
			m.previewFocus = true
			m.previewKind = "text"
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
		if m.searching {
			return m.updateSearch(msg)
		}
		if m.previewFocus && m.isPreviewScrollKey(msg) {
			m.scrollPreview(msg)
			return m, nil
		}
		return m.updateKey(msg)
	}
	return m, nil
}

func (m model) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.running {
		if matches(msg, m.cfg.Keys.Back) {
			if m.runCancel != nil {
				m.runCancel()
			}
			m.status = "cancelling command"
		}
		return m, nil
	}
	switch {
	case matches(msg, m.cfg.Keys.Back) && m.previewFocus:
		m.previewFocus = false
		return m, nil
	case matches(msg, m.cfg.Keys.Quit):
		return m, tea.Quit
	case matches(msg, m.cfg.Keys.TabNext):
		m.tab = (m.tab + 1) % tabCount
		m.cursor, m.output, m.loading, m.previewFocus = 0, "", true, false
		return m, m.load()
	case matches(msg, m.cfg.Keys.TabPrev):
		m.tab = (m.tab + tabCount - 1) % tabCount
		m.cursor, m.output, m.loading, m.previewFocus = 0, "", true, false
		return m, m.load()
	case matches(msg, m.cfg.Keys.Down):
		m.cursor = clamp(m.cursor+1, 0, len(m.items)-1)
	case matches(msg, m.cfg.Keys.Up):
		m.cursor = clamp(m.cursor-1, 0, len(m.items)-1)
	case matches(msg, m.cfg.Keys.Search):
		m.searching = true
		m.searchBefore = m.filter
		m.previewFocus = false
		m.filterInput.SetValue(m.filter)
		return m, m.filterInput.Focus()
	case matches(msg, m.cfg.Keys.Refresh):
		m.loading = true
		return m, m.load()
	case matches(msg, m.cfg.Keys.Back) && m.previewFocus:
		m.previewFocus = false
		return m, nil
	case matches(msg, m.cfg.Keys.Start):
		return m, m.action("start")
	case matches(msg, m.cfg.Keys.Stop):
		return m, m.action("stop")
	case matches(msg, m.cfg.Keys.Remove):
		return m, m.action("remove")
	case matches(msg, m.cfg.Keys.Inspect):
		if runtime.Resource(m.tab) == runtime.Files {
			return m, m.previewFile()
		}
		return m, m.action("inspect")
	case matches(msg, m.cfg.Keys.Logs):
		return m, m.logs()
	case matches(msg, m.cfg.Keys.New):
		return m, m.beginPrompt()
	case matches(msg, m.cfg.Keys.OpenConfig):
		return m, openConfig()
	case matches(msg, m.cfg.Keys.OpenPreview) && m.output != "" && runtime.Resource(m.tab) != runtime.Files:
		return m, openPreview(m.output, m.previewKind)
	case matches(msg, m.cfg.Keys.OpenPreview) && runtime.Resource(m.tab) == runtime.Files:
		return m, m.openSelectedFile()
	case matches(msg, m.cfg.Keys.Setup):
		return m, openSetup()
	case matches(msg, m.cfg.Keys.Run) && runtime.Resource(m.tab) == runtime.Files:
		return m.startFileRun()
	case matches(msg, m.cfg.Keys.Edit) && runtime.Resource(m.tab) == runtime.Compose:
		return m, openComposeFile()
	}
	return m, nil
}

func (m model) load() tea.Cmd {
	return func() tea.Msg {
		if runtime.Resource(m.tab) == runtime.Files {
			root, err := os.Getwd()
			if err != nil {
				return loadedMsg{err: err}
			}
			items, err := projectfiles.Discover(root)
			return loadedMsg{items: items, err: err}
		}
		items, err := m.client.List(context.Background(), runtime.Resource(m.tab))
		return loadedMsg{items: items, err: err}
	}
}

func (m model) checkUpdates() tea.Cmd {
	return func() tea.Msg {
		return updateCheckedMsg{result: update.Check(context.Background(), m.cfg, version.Current())}
	}
}

func (m model) action(action string) tea.Cmd {
	if len(m.items) == 0 || m.cursor >= len(m.items) {
		return nil
	}
	name := m.items[m.cursor].Name
	return func() tea.Msg {
		output, err := m.client.Action(context.Background(), runtime.Resource(m.tab), action, name)
		kind := "text"
		if action == "inspect" {
			kind = "json"
		}
		return actionMsg{output: output, err: err, kind: kind, reload: true, focusPreview: action == "inspect"}
	}
}

func (m model) logs() tea.Cmd {
	if len(m.items) == 0 || m.cursor >= len(m.items) {
		return nil
	}
	name := m.items[m.cursor].Name
	return func() tea.Msg {
		output, err := m.client.Logs(context.Background(), runtime.Resource(m.tab), name)
		return actionMsg{output: output, err: err, kind: "log", reload: true}
	}
}

func (m model) selectedFile() (runtime.Item, bool) {
	if runtime.Resource(m.tab) != runtime.Files || m.cursor < 0 || m.cursor >= len(m.items) {
		return runtime.Item{}, false
	}
	return m.items[m.cursor], true
}

func (m model) previewFile() tea.Cmd {
	item, ok := m.selectedFile()
	if !ok {
		return nil
	}
	return func() tea.Msg {
		content, err := os.ReadFile(item.Path)
		return actionMsg{output: string(content), err: err, kind: "text", focusPreview: true}
	}
}

func (m model) openSelectedFile() tea.Cmd {
	item, ok := m.selectedFile()
	if !ok {
		return nil
	}
	return openFileInEditor(item.Path)
}

func (m model) startFileRun() (tea.Model, tea.Cmd) {
	item, ok := m.selectedFile()
	if !ok {
		return m, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := m.client.StartFile(ctx, item)
	if err != nil {
		cancel()
		return m, func() tea.Msg { return actionMsg{err: err, kind: "text"} }
	}
	m.runCh = ch
	m.runCancel = cancel
	m.running = true
	m.loading = true
	m.status = "running selected file"
	m.output = ""
	m.previewFocus = true
	return m, waitForRunEvent(ch)
}

func waitForRunEvent(events <-chan runtime.RunEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return runEventMsg{event: runtime.RunEvent{Done: true}}
		}
		return runEventMsg{event: event}
	}
}

func (m model) updateSearch(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case matches(msg, m.cfg.Keys.Confirm):
		m.filterInput.Blur()
		m.filter = m.filterInput.Value()
		m.searching = false
		m.applyFilter()
		return m, nil
	case matches(msg, m.cfg.Keys.ClearSearch) || matches(msg, m.cfg.Keys.Back):
		m.filterInput.Blur()
		m.filter = m.searchBefore
		m.filterInput.SetValue(m.filter)
		m.searching = false
		m.applyFilter()
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filter = m.filterInput.Value()
	m.applyFilter()
	return m, cmd
}

func (m *model) applyFilter() {
	query := strings.TrimSpace(strings.ToLower(m.filter))
	if query == "" {
		m.items = append([]runtime.Item(nil), m.allItems...)
		m.cursor = clamp(m.cursor, 0, len(m.items)-1)
		return
	}
	var filtered []runtime.Item
	for _, item := range m.allItems {
		if matchesFilter(item, query) {
			filtered = append(filtered, item)
		}
	}
	m.items = filtered
	m.cursor = clamp(m.cursor, 0, len(m.items)-1)
}

func matchesFilter(item runtime.Item, query string) bool {
	text := strings.ToLower(strings.Join([]string{item.Status, item.Name, item.Image, item.Ports, item.Extra, item.Details, item.Type, item.Path}, " "))
	for _, token := range strings.Fields(query) {
		key, value, ok := strings.Cut(token, ":")
		if !ok {
			if !strings.Contains(text, token) {
				return false
			}
			continue
		}
		field := strings.ToLower(value)
		var source string
		switch key {
		case "status":
			source = item.Status
		case "name":
			source = item.Name
		case "image":
			source = item.Image
		case "port", "ports":
			source = item.Ports
		default:
			source = text
		}
		if !strings.Contains(strings.ToLower(source), field) {
			return false
		}
	}
	return true
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
	if matches(msg, m.cfg.Keys.Back) {
		m.prompt = nil
		return m, nil
	}
	if matches(msg, m.cfg.Keys.Confirm) {
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
	view = clampView(view, m.width, m.height)
	result := tea.NewView(view)
	result.AltScreen = true
	result.MouseMode = tea.MouseModeCellMotion
	return result
}

func clampView(view string, width, height int) string {
	lines := strings.Split(view, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, max(1, width-1), "")
	}
	return strings.Join(lines, "\n")
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

	count := fmt.Sprintf("%d", len(m.items))
	if m.filter != "" {
		count += "/" + fmt.Sprintf("%d", len(m.allItems))
	}
	rows := []string{ui.Header.Render(fit(fmt.Sprintf("%s  %s %s", tabNames[m.tab], count, m.status), max(1, m.width-2)))}
	if m.loading {
		rows = append(rows, ui.Muted.Render("Loading..."))
	} else if len(m.items) == 0 {
		rows = append(rows, ui.Muted.Render("No resources found."))
	} else {
		if runtime.Resource(m.tab) == runtime.Containers {
			rows = append(rows, ui.Muted.Render(fit("STATUS       NAME                 IMAGE              PORT(S)              OTHER", max(1, m.width-6))))
		} else if runtime.Resource(m.tab) == runtime.Files {
			rows = append(rows, ui.Muted.Render(fit("TYPE         PATH", max(1, m.width-6))))
		}
		for i, item := range m.items {
			line := m.renderItemLine(item)
			if i == m.cursor {
				line = ui.Selected.Render(line)
			}
			rows = append(rows, line)
		}
	}
	listBoxHeight, outputBoxHeight, promptBoxHeight := m.layoutHeights()
	searchView := ""
	if m.searching {
		searchView = ui.FocusedPanel.Width(max(1, m.width-1)).Height(1).Render(ui.Key.Render(m.cfg.Keys.Search) + " " + m.filterInput.View())
	}
	promptView := ""
	if m.prompt != nil {
		promptView = ui.FocusedPanel.Width(max(1, m.width-1)).Height(max(1, promptBoxHeight-2)).Render(m.prompt.View())
	}
	output := ""
	if m.output != "" {
		output = ui.FocusedPanel.Width(max(1, m.width-1)).Height(max(1, outputBoxHeight-2)).Render(m.outputViewport.View())
	}
	if limit := max(1, listBoxHeight-2); len(rows) > limit {
		rows = append(rows[:limit-1], ui.Muted.Render("... more items; scroll is available in a later view"))
	}
	list := ui.Panel.Width(max(1, m.width-1)).Height(max(1, listBoxHeight-2)).Render(strings.Join(rows, "\n"))
	hints := m.renderHints()
	divider := ui.Muted.Render(strings.Repeat("─", max(1, m.width)))
	sections := []string{brand, tabBar}
	if searchView != "" {
		sections = append(sections, searchView)
	}
	sections = append(sections, list)
	if output != "" {
		sections = append(sections, output)
	}
	if promptView != "" {
		sections = append(sections, promptView)
	}
	sections = append(sections, divider, hints)
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m model) renderItemLine(item runtime.Item) string {
	if runtime.Resource(m.tab) == runtime.Files {
		return fit(fmt.Sprintf("%-12s %s", item.Type, item.Name), max(1, m.width-6))
	}
	if runtime.Resource(m.tab) == runtime.Containers {
		return fit(fmt.Sprintf("%-12s %-20s %-18s %-20s %s", "["+item.Status+"]", item.Name, item.Image, item.Ports, item.Extra), max(1, m.width-6))
	}
	return fit(fmt.Sprintf("%-32s %s", item.Name, item.Details), max(1, m.width-6))
}

func (m *model) resize() {
	if m.width < 1 || m.height < 1 {
		return
	}
	_, outputBoxHeight, _ := m.layoutHeights()
	m.outputViewport.SetWidth(max(1, m.width-6))
	m.outputViewport.SetHeight(max(1, outputBoxHeight-2))
}

func (m *model) setOutput(output, kind string) {
	m.output = strings.TrimSpace(output)
	m.previewKind = kind
	m.outputViewport.SetContent(m.output)
	m.outputViewport.GotoTop()
	m.resize()
}

func (m *model) appendOutput(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	if m.output == "" {
		m.setOutput(line, "text")
		return
	}
	m.output += "\n" + line
	m.outputViewport.SetContent(m.output)
	m.outputViewport.GotoBottom()
	m.resize()
}

func (m model) layoutHeights() (listBoxHeight, outputBoxHeight, promptBoxHeight int) {
	const (
		headerHeight  = 1
		tabsHeight    = 1
		dividerHeight = 1
		hintsHeight   = 2
	)
	searchHeight := 2
	if m.prompt != nil {
		promptBoxHeight = 3
	}
	if !m.searching {
		searchHeight = 0
	}
	available := max(1, m.height-headerHeight-tabsHeight-dividerHeight-hintsHeight-promptBoxHeight-searchHeight)
	listBoxHeight = available
	if m.output != "" && available >= 8 {
		outputBoxHeight = max(4, available/3)
		listBoxHeight = max(3, available-outputBoxHeight)
	}
	return listBoxHeight, outputBoxHeight, promptBoxHeight
}

func (m model) isPreviewScrollKey(msg tea.KeyPressMsg) bool {
	return matches(msg, m.cfg.Keys.Up) || matches(msg, m.cfg.Keys.Down) || matches(msg, m.cfg.Keys.ScrollUp) || matches(msg, m.cfg.Keys.ScrollDown) || matches(msg, m.cfg.Keys.PageUp) || matches(msg, m.cfg.Keys.PageDown)
}

func (m *model) scrollPreview(msg tea.KeyPressMsg) {
	switch {
	case matches(msg, m.cfg.Keys.Up) || matches(msg, m.cfg.Keys.ScrollUp):
		m.outputViewport.ScrollUp(1)
	case matches(msg, m.cfg.Keys.Down) || matches(msg, m.cfg.Keys.ScrollDown):
		m.outputViewport.ScrollDown(1)
	case matches(msg, m.cfg.Keys.PageUp):
		m.outputViewport.PageUp()
	case matches(msg, m.cfg.Keys.PageDown):
		m.outputViewport.PageDown()
	}
}

func (m model) renderHints() string {
	hint := func(key, description string) string {
		if key == "" {
			return ""
		}
		return key + ":" + description + "  "
	}
	common := hint(m.cfg.Keys.TabNext, "next") + hint(m.cfg.Keys.TabPrev, "previous") + hint(m.cfg.Keys.Up, "up") + hint(m.cfg.Keys.Down, "down") + hint(m.cfg.Keys.Search, "filter")
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
	case runtime.Files:
		actions = hint(m.cfg.Keys.Inspect, "preview") + hint(m.cfg.Keys.OpenPreview, "edit") + hint(m.cfg.Keys.Run, "run")
	}
	tail := hint(m.cfg.Keys.Setup, "setup") + hint(m.cfg.Keys.OpenConfig, "config") + hint(m.cfg.Keys.Quit, "quit")
	if m.output != "" {
		tail = hint(m.cfg.Keys.OpenPreview, "editor") + tail
	}
	parts := strings.Fields(common + actions + tail)
	if m.output != "" {
		parts = append(parts, "scroll:"+m.cfg.Keys.ScrollUp+"/"+m.cfg.Keys.ScrollDown)
	}
	lines := make([]string, 0, 2)
	line := ""
	for _, part := range parts {
		candidate := part
		if line != "" {
			candidate = line + "  " + part
		}
		if line != "" && lipgloss.Width(candidate) > m.width {
			lines = append(lines, line)
			line = part
			continue
		}
		line = candidate
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) > 2 {
		lines = lines[:2]
	}
	var rendered strings.Builder
	for lineIndex, hintLine := range lines {
		if lineIndex > 0 {
			rendered.WriteByte('\n')
		}
		for _, part := range strings.Fields(hintLine) {
			pieces := strings.SplitN(part, ":", 2)
			if len(pieces) == 2 {
				rendered.WriteString(ui.Key.Render(pieces[0]))
				rendered.WriteString(ui.Muted.Render(":" + pieces[1] + "  "))
			}
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

func openPreview(content, kind string) tea.Cmd {
	suffix := ".txt"
	if kind == "json" {
		suffix = ".json"
	}
	if kind == "log" {
		suffix = ".log"
	}
	tmp, err := os.CreateTemp("", "ducky-preview-*"+suffix)
	if err != nil {
		return func() tea.Msg { return actionMsg{err: err} }
	}
	path := tmp.Name()
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		_ = os.Remove(path)
		return func() tea.Msg { return actionMsg{err: err} }
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(path)
		return func() tea.Msg { return actionMsg{err: err} }
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "nano"
	}
	return tea.ExecProcess(exec.Command(editor, path), func(err error) tea.Msg {
		_ = os.Remove(path)
		return actionMsg{err: err, output: content, kind: kind}
	})
}

func openFileInEditor(path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "nano"
	}
	return tea.ExecProcess(exec.Command(editor, path), func(err error) tea.Msg {
		return actionMsg{err: err, output: "editor closed", kind: "text"}
	})
}

func openSetup() tea.Cmd {
	return tea.ExecProcess(exec.Command(os.Args[0], "setup"), func(err error) tea.Msg {
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{output: "setup complete", kind: "text"}
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
	if binding == "" {
		return false
	}
	actual := msg.String()
	if actual == binding {
		return true
	}
	// Bubble Tea may report an uppercase printable binding as shift+<letter>
	// depending on the terminal keyboard protocol in use.
	if len([]rune(binding)) == 1 && binding >= "A" && binding <= "Z" {
		return actual == "shift+"+strings.ToLower(binding)
	}
	return false
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
