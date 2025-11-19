package ui

import tea "github.com/charmbracelet/bubbletea"

type payloadsModel struct{}

func newPayloadsModel() tea.Model {
	return payloadsModel{}
}

func (m payloadsModel) Init() tea.Cmd {
	return nil
}

func (m payloadsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m payloadsModel) View() string {
	return "WIP"
}
