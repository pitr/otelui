package ui

import tea "github.com/charmbracelet/bubbletea"

type metricsModel struct{}

func newMetricsModel() tea.Model {
	return &metricsModel{}
}

func (m *metricsModel) Init() tea.Cmd {
	return nil
}

func (m *metricsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *metricsModel) View() string {
	return "WIP"
}
