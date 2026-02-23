package ui

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"pitr.ca/otelui/server"
	"pitr.ca/otelui/ui/components"
)

type mRoot uint

const (
	mRootLogs mRoot = iota
	mRootTraces
	mRootMetrics
	mRootPayloads

	mRootTopOffset = 1
)

type keyMapRoot struct {
	Next  key.Binding
	Prev  key.Binding
	Quit  key.Binding
	TZ    key.Binding
	Reset key.Binding
}

type model struct {
	keyMap keyMapRoot
	help   help.Model

	mode   mRoot
	w      int
	models map[mRoot]tea.Model
}

func newRootModel() tea.Model {
	names := []string{"Logs", "Traces", "Metrics", "Payloads"}
	return &model{
		keyMap: keyMapRoot{
			Next:  key.NewBinding(key.WithKeys("]"), key.WithHelp("[ ]", "switch mode")),
			Prev:  key.NewBinding(key.WithKeys("[")),
			Quit:  key.NewBinding(key.WithKeys("ctrl+c", "q")),
			TZ:    key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "UTC/local")),
			Reset: key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "reset")),
		},
		help: help.New(),
		models: map[mRoot]tea.Model{
			mRootLogs:     newLogsModel(rootTabTitle(names, mRootLogs)),
			mRootTraces:   newTracesModel(rootTabTitle(names, mRootTraces)),
			mRootMetrics:  newMetricsModel(rootTabTitle(names, mRootMetrics)),
			mRootPayloads: newPayloadsModel(rootTabTitle(names, mRootPayloads)),
		},
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	default:
		defer func(start time.Time) {
			slog.Debug(fmt.Sprintf("Update(%T) %s - %#v", msg, time.Since(start), msg))
		}(time.Now())
	}

	switch msg := msg.(type) {
	case server.ConsumeEvent:
		for k, v := range m.models {
			m.models[k], cmd = v.Update(msg)
			cmds = append(cmds, cmd)
		}
		cmd = tea.Batch(cmds...)
	case tea.WindowSizeMsg:
		m.w = msg.Width
		msg.Height -= mRootTopOffset
		for k, v := range m.models {
			m.models[k], cmd = v.Update(msg)
			cmds = append(cmds, cmd)
		}
		cmd = tea.Batch(cmds...)
	case tea.MouseMsg:
		if msg.Y >= mRootTopOffset {
			msg.Y -= mRootTopOffset
			m.models[m.mode], cmd = m.models[m.mode].Update(msg)
		}
	case tea.KeyMsg:
		capturing := false
		if ic, ok := m.models[m.mode].(components.InputCapture); ok {
			capturing = ic.IsCapturingInput()
		}
		switch {
		case key.Matches(msg, m.keyMap.Quit) && (!capturing || msg.String() == "ctrl+c"):
			return m, tea.Quit
		case key.Matches(msg, m.keyMap.Next) && !capturing:
			m.mode = (m.mode + 1) % mRoot(len(m.models))
		case key.Matches(msg, m.keyMap.Prev) && !capturing:
			m.mode = (m.mode - 1) % mRoot(len(m.models))
		case key.Matches(msg, m.keyMap.TZ) && !capturing:
			tzUTC = !tzUTC
			components.TZUTC = tzUTC
			for k, v := range m.models {
				m.models[k], cmd = v.Update(refreshMsg{})
				cmds = append(cmds, cmd)
			}
			cmd = tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Reset):
			server.Reset()
			for k, v := range m.models {
				m.models[k], cmd = v.Update(refreshMsg{reset: true})
				cmds = append(cmds, cmd)
			}
			cmd = tea.Batch(cmds...)
		default:
			m.models[m.mode], cmd = m.models[m.mode].Update(msg)
		}
	default:
		m.models[m.mode], cmd = m.models[m.mode].Update(msg)
	}

	return m, cmd
}

func (m model) View() string {
	defer func(start time.Time) { slog.Debug(fmt.Sprintf("View() %s", time.Since(start))) }(time.Now())

	keys := []key.Binding{m.keyMap.Next, m.keyMap.Reset, m.keyMap.TZ}
	if h, ok := m.models[m.mode].(components.Helpful); ok {
		keys = append(keys, h.Help()...)
	}
	m.help.Width = m.w
	return m.models[m.mode].View() + "\n " + m.help.ShortHelpView(keys)
}

func rootTabTitle(names []string, m mRoot) string {
	s := lipgloss.NewStyle().Foreground(components.AccentColor)
	parts := make([]string, len(names))
	for i, name := range names {
		if mRoot(i) == m {
			parts[i] = s.Render(name)
		} else {
			parts[i] = "\033[0m" + name
		}
	}
	return strings.Join(parts, " ")
}
