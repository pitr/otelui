package ui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/pitr/otelui/server"
	"github.com/pitr/otelui/ui/components"
)

type metricsModel struct {
	view        components.Splitview[*components.Viewport, *components.Timeseries]
	lastMetrics int
}

func newMetricsModel() tea.Model {
	m := metricsModel{lastMetrics: -1}
	m.view = components.NewSplitview(
		components.NewViewport("Metrics", m.updateDetailsContent),
		components.NewTimeseries("Details"),
	)
	return m
}

func (m metricsModel) Init() tea.Cmd {
	return nil
}

func (m metricsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case server.ConsumeEvent:
		if m.lastMetrics != msg.Metrics {
			m.lastMetrics = msg.Metrics
			m.updateMainContent()
		}
	default:
		m.view, cmd = m.view.Update(msg)
		return m, cmd
	}
	return m, cmd
}

func (m metricsModel) View() string {
	return m.view.View()
}

func (m metricsModel) Help() []key.Binding {
	return m.view.Help()
}

func (m *metricsModel) updateMainContent() {
	lines := []components.ViewRow{}
	for _, m := range server.GetMetrics() {
		lines = append(lines, components.ViewRow{Str: m, Yank: m, Raw: m})
	}
	sort.Slice(lines, func(i, j int) bool { return strings.Compare(lines[i].Str, lines[j].Str) < 0 })
	m.view.Top().SetContent(lines)
}

func (m *metricsModel) updateDetailsContent(selected components.ViewRow) {
	metric := selected.Raw.(string)
	m.view.Bot().SetContent(metric)
}
