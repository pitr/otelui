package ui

import tea "github.com/charmbracelet/bubbletea"

type tracesModel struct{}

func newTracesModel() tea.Model {
	return &tracesModel{}
}

func (m *tracesModel) Init() tea.Cmd {
	return nil
}

func (m *tracesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *tracesModel) View() string {
	return "WIP"
}
