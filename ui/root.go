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
)

type keyMapRoot struct {
	Next key.Binding
	Prev key.Binding
	Quit key.Binding
}

func (k keyMapRoot) Help() []key.Binding {
	return []key.Binding{k.Next}
}

type model struct {
	keyMap keyMapRoot
	help   help.Model

	mode      mRoot
	w         int
	topoffset int
	models    map[mRoot]tea.Model
	menus     []string
	_selected lipgloss.Style
	ce        server.ConsumeEvent
	status    string
}

func newRootModel() tea.Model {
	return &model{
		keyMap: keyMapRoot{
			Next: key.NewBinding(key.WithKeys("]"), key.WithHelp("[ ]", "switch mode")),
			Prev: key.NewBinding(key.WithKeys("[")),
			Quit: key.NewBinding(key.WithKeys("ctrl+c", "q")),
		},
		help:      help.New(),
		topoffset: 2,
		models: map[mRoot]tea.Model{
			mRootLogs:     newLogsModel(),
			mRootTraces:   newTracesModel(),
			mRootMetrics:  newMetricsModel(),
			mRootPayloads: newPayloadsModel(),
		},
		menus: []string{
			"   Logs   ",
			"  Traces  ",
			" Metrics ",
			" Payloads ",
		},
		status:    "waiting for data...",
		_selected: lipgloss.NewStyle().Background(components.FadedColor).Bold(true),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case QueriedLogs:
	default:
		start := time.Now()
		defer func() { slog.Debug(fmt.Sprintf("%s to process %#v", time.Since(start), msg)) }()
	}

	switch msg := msg.(type) {
	case server.ConsumeEvent:
		payloadsStyle := lipgloss.NewStyle()
		logsStyle := lipgloss.NewStyle()
		spansStyle := lipgloss.NewStyle()
		metricsStyle := lipgloss.NewStyle()
		if msg.Payloads != m.ce.Payloads {
			payloadsStyle = payloadsStyle.Background(components.HighlightColor)
		}
		if msg.Logs != m.ce.Logs {
			logsStyle = logsStyle.Background(components.HighlightColor)
		}
		if msg.Spans != m.ce.Spans {
			spansStyle = spansStyle.Background(components.HighlightColor)
		}
		if msg.Metrics != m.ce.Metrics {
			metricsStyle = metricsStyle.Background(components.HighlightColor)
		}
		m.status = fmt.Sprintf("payloads=%s logs=%s spans=%s metrics=%s", payloadsStyle.Render(strconv.Itoa(msg.Payloads)), logsStyle.Render(strconv.Itoa(msg.Logs)), spansStyle.Render(strconv.Itoa(msg.Spans)), metricsStyle.Render(strconv.Itoa(msg.Metrics)))
		m.ce = msg
		m.models[m.mode], cmd = m.models[m.mode].Update(msg)
	case tea.WindowSizeMsg:
		m.w = msg.Width
		msg.Height -= m.topoffset
		var cmd2 tea.Cmd
		for k, v := range m.models {
			m.models[k], cmd2 = v.Update(msg)
			cmd = tea.Batch(cmd, cmd2)
		}
	case tea.MouseMsg:
		if msg.Y >= m.topoffset {
			msg.Y -= m.topoffset
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
		default:
			m.models[m.mode], cmd = m.models[m.mode].Update(msg)
		}
	default:
		m.models[m.mode], cmd = m.models[m.mode].Update(msg)
	}

	return m, cmd
}

func (m model) View() string {
	m.help.Width = m.w - lipgloss.Width(m.status) - 3

	keys := m.keyMap.Help()
	if m, ok := m.models[m.mode].(components.Helpful); ok {
		keys = append(m.Help(), keys...)
	}
	help := m.help.ShortHelpView(keys)

	gap := strings.Repeat(" ", max(0, m.w-lipgloss.Width(help+m.status)))
	header := m.status + gap + help

	menu := ""
	for i, mode := range m.menus {
		if m.mode == mRoot(i) {
			menu += m._selected.Render(mode)
		} else {
			menu += mode
		}
	}
	menu = lipgloss.PlaceHorizontal(m.w, lipgloss.Center, menu)
	view := m.models[m.mode].View()

	return lipgloss.JoinVertical(lipgloss.Left, header, menu, view)
}
