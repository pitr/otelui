package ui

import (
	"sort"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"pitr.ca/otelui/server"
	"pitr.ca/otelui/ui/components"
)

type metricsModel struct {
	view        components.Splitview[*components.Viewport, *components.Timeseries]
	lastMetrics int
}

func newMetricsModel(title string) tea.Model {
	m := metricsModel{lastMetrics: -1}
	m.view = components.NewSplitview(
		components.NewViewport(title).WithSelectFunc(m.updateDetailsContent),
		components.NewTimeseries("Details"),
	)
	return m
}

func (m metricsModel) Init() tea.Cmd          { return nil }
func (m metricsModel) View() string           { return m.view.View() }
func (m metricsModel) Help() []key.Binding    { return m.view.Help() }
func (m metricsModel) IsCapturingInput() bool { return m.view.Top().IsCapturingInput() }

func (m metricsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case refreshMsg:
		if msg.reset {
			m.lastMetrics = -1
		}
		m.updateMainContent()
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

func (m *metricsModel) updateMainContent() {
	lines := []components.ViewRow{}
	ms := server.GetMetrics()
	sort.Strings(ms)
	for _, m := range ms {
		lines = append(lines, components.ViewRow{Str: m, Raw: m})
	}
	m.view.Top().SetContent(lines)
}

func (m *metricsModel) updateDetailsContent(selected components.ViewRow) {
	metric, _ := selected.Raw.(string)
	m.view.Bot().SetContent(metric)
}
