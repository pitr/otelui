package ui

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pitr/otelui/server"
	"github.com/pitr/otelui/ui/components"
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

	mode      mRoot
	w         int
	models    map[mRoot]tea.Model
	_selected lipgloss.Style
	ce        server.ConsumeEvent
	statuses  []any
}

func newRootModel() tea.Model {
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
			mRootLogs:     newLogsModel(),
			mRootTraces:   newTracesModel(),
			mRootMetrics:  newMetricsModel(),
			mRootPayloads: newPayloadsModel(),
		},
		_selected: lipgloss.NewStyle().Background(components.SelectionColor).Bold(true),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

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
		payloadsStyle := lipgloss.NewStyle()
		logsStyle := lipgloss.NewStyle()
		spansStyle := lipgloss.NewStyle()
		metricsStyle := lipgloss.NewStyle()
		if msg.Payloads != m.ce.Payloads {
			payloadsStyle = payloadsStyle.Background(components.SelectionColor)
		}
		if msg.Logs != m.ce.Logs {
			logsStyle = logsStyle.Background(components.SelectionColor)
		}
		if msg.Spans != m.ce.Spans {
			spansStyle = spansStyle.Background(components.SelectionColor)
		}
		if msg.Metrics != m.ce.Metrics {
			metricsStyle = metricsStyle.Background(components.SelectionColor)
		}
		m.statuses = []any{
			logsStyle.Render(strconv.Itoa(msg.Logs)),
			spansStyle.Render(strconv.Itoa(msg.Spans)),
			metricsStyle.Render(strconv.Itoa(msg.Metrics)),
			payloadsStyle.Render(strconv.Itoa(msg.Payloads)),
		}
		m.ce = msg
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
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keyMap.Next):
			m.mode = (m.mode + 1) % mRoot(len(m.models))
		case key.Matches(msg, m.keyMap.Prev):
			m.mode = (m.mode - 1) % mRoot(len(m.models))
		case key.Matches(msg, m.keyMap.TZ):
			tzUTC = !tzUTC
			components.TZUTC = tzUTC
			for k, v := range m.models {
				m.models[k], cmd = v.Update(refreshMsg{})
				cmds = append(cmds, cmd)
			}
			cmd = tea.Batch(cmds...)
		case key.Matches(msg, m.keyMap.Reset):
			server.Reset()
			m.ce = server.ConsumeEvent{}
			m.statuses = nil
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

	status := "waiting for data..."
	if len(m.statuses) > 0 {
		statusfmt := []string{"logs", "spans", "metrics", "payloads"}
		for i, s := range statusfmt {
			if i == int(m.mode) {
				s = lipgloss.NewStyle().Bold(true).Render(strings.ToUpper(s))
			}
			statusfmt[i] = s + "=%s"
		}
		status = fmt.Sprintf(strings.Join(statusfmt, " "), m.statuses...)
	}
	keys := []key.Binding{m.keyMap.TZ, m.keyMap.Reset, m.keyMap.Next}
	if m, ok := m.models[m.mode].(components.Helpful); ok {
		keys = append(m.Help(), keys...)
	}

	m.help.Width = m.w - lipgloss.Width(status) - 3
	help := m.help.ShortHelpView(keys)

	gap := strings.Repeat(" ", max(0, m.w-lipgloss.Width(help+status)))
	header := status + gap + help

	return header + "\n" + m.models[m.mode].View()
}
