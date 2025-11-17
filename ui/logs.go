package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	"github.com/pitr/otelui/server"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

type mLogs uint

const (
	mLogsFocusMain mLogs = iota
	mLogsFocusDetails
)

type keyMapLogs struct {
	Increase key.Binding
	Decrease key.Binding
	Up       key.Binding
	Down     key.Binding
	Next     key.Binding
	Prev     key.Binding
}

func (k keyMapLogs) Help() []key.Binding {
	return []key.Binding{k.Increase, k.Up, k.Next}
}

type QueriedLogs []*server.Log

type logsModel struct {
	mode        mLogs
	hm, hd      int
	w           int
	keyMap      keyMapLogs
	logs        []*server.Log
	selectedLog *server.Log
	_selected   lipgloss.Style
	_focused    lipgloss.Style
	_unfocused  lipgloss.Style
	main        viewport.Model
	details     viewport.Model
}

func newLogsModel() tea.Model {
	faded := lipgloss.AdaptiveColor{Light: "#B2B2B2", Dark: "#4A4A4A"}
	return &logsModel{
		keyMap: keyMapLogs{
			Increase: key.NewBinding(key.WithKeys("+"), key.WithHelp("+/-", "resize")),
			Decrease: key.NewBinding(key.WithKeys("-")),
			Up:       key.NewBinding(key.WithKeys("up"), key.WithHelp("↑/↓", "select")),
			Down:     key.NewBinding(key.WithKeys("down")),
			Next:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
			Prev:     key.NewBinding(key.WithKeys("shift+tab")),
		},
		logs:       []*server.Log{},
		_selected:  lipgloss.NewStyle().Background(faded),
		_focused:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()),
		_unfocused: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(faded),
		main:       viewport.New(10, 10),
		details:    viewport.New(10, 10),
	}
}

func (m *logsModel) Init() tea.Cmd {
	return nil
}

func (m *logsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.main.Width = msg.Width
		m.details.Width = msg.Width
		m.hm = msg.Height / 3 * 2
		m.hd = msg.Height - m.hm
		m.main.Height = m.hm
		m.details.Height = m.hd
		m.main.SetHorizontalStep(1)
		m.main.MouseWheelDelta = 1
		m.details.SetHorizontalStep(1)
		m.details.MouseWheelDelta = 1
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Increase):
			if m.hd > 6 {
				m.hd -= 2
				m.hm += 2
				m.main.Height = m.hm
				m.details.Height = m.hd
			}
		case key.Matches(msg, m.keyMap.Decrease):
			if m.hm > 6 {
				m.hd += 2
				m.hm -= 2
				m.main.Height = m.hm
				m.details.Height = m.hd
			}
		case key.Matches(msg, m.keyMap.Next):
			m.mode = (m.mode + 1) % 2
		case key.Matches(msg, m.keyMap.Prev):
			m.mode = (m.mode - 1) % 2
		case key.Matches(msg, m.keyMap.Up):
			m.selectUp()
		case key.Matches(msg, m.keyMap.Down):
			m.selectDown()
		}
	case tea.MouseMsg:
		if msg.Y >= m.hm && m.selectedLog != nil {
			m.details, cmd = m.details.Update(msg)
			return m, cmd
		} else if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionRelease {
			offset := m.main.YOffset - m.main.Style.GetBorderTopSize()
			if len(m.logs) > msg.Y+offset {
				m.selectLog(m.logs[msg.Y+offset])
			}
		} else {
			m.main, cmd = m.main.Update(msg)
			return m, cmd
		}
	case server.ConsumeEvent:
		return m, func() tea.Msg { return QueriedLogs(server.QueryLogs(100)) }
	case QueriedLogs:
		m.logs = msg
		m.renderMain()
	}
	return m, nil
}

func (m *logsModel) View() string {
	switch m.mode {
	case mLogsFocusMain:
		m.main.Style = m._focused
		m.details.Style = m._unfocused
	case mLogsFocusDetails:
		m.main.Style = m._unfocused
		m.details.Style = m._focused
	}
	return m.main.View() + "\n" + m.details.View()
}

func (m *logsModel) selectLog(l *server.Log) {
	m.selectedLog = l
	if l == nil {
		m.mode = mLogsFocusMain
	}
	m.renderMain()
	m.renderDetails()
}

func (m *logsModel) renderMain() {
	var buf strings.Builder
	for i, l := range m.logs {
		if i > 0 {
			buf.WriteByte('\n')
		}
		s := lipgloss.NewStyle()
		switch {
		case l.Log.SeverityNumber() >= plog.SeverityNumberError:
			s = s.Foreground(lipgloss.Color("9"))
		case l.Log.SeverityNumber() >= plog.SeverityNumberWarn:
			s = s.Foreground(lipgloss.Color("11"))
		case l.Log.SeverityNumber() >= plog.SeverityNumberInfo:
			s = s.Foreground(lipgloss.Color("14"))
		case l.Log.SeverityNumber() >= plog.SeverityNumberDebug:
			s = s.Foreground(lipgloss.Color("15"))
		}
		svc := "-"
		if val, ok := l.ResourceLogs.Resource().Attributes().Get(string(semconv.ServiceNameKey)); ok && val.Type() == pcommon.ValueTypeStr {
			svc = val.AsString()
		}
		str := strings.Join([]string{
			l.Log.Timestamp().AsTime().UTC().Format(time.RFC3339),
			svc,
			s.Render(lipgloss.PlaceHorizontal(3, lipgloss.Left, l.Log.SeverityText())),
		}, " ")
		if l == m.selectedLog {
			buf.WriteString(m._selected.Render(str))
			buf.WriteString(m._selected.Render(" "))
			buf.WriteString(m._selected.Render(l.Log.Body().AsString()))
		} else {
			buf.WriteString(str)
			buf.WriteByte(' ')
			buf.WriteString(l.Log.Body().AsString())
		}
	}
	m.main.SetContent(buf.String())
}

func (m *logsModel) renderDetails() {
	if m.selectedLog == nil {
		m.details.SetContent("")
		return
	}

	attrs := tree.New().Root("Attributes")
	for k, v := range m.selectedLog.Log.Attributes().All() {
		attrs = attrs.Child(fmt.Sprintf("%s: %s", k, v.AsString()))
	}
	sattrs := tree.New().Root("Attributes")
	for k, v := range m.selectedLog.ScopeLogs.Scope().Attributes().All() {
		sattrs = sattrs.Child(fmt.Sprintf("%s: %s", k, v.AsString()))
	}
	rattrs := tree.New().Root("Attributes")
	for k, v := range m.selectedLog.ResourceLogs.Resource().Attributes().All() {
		rattrs = rattrs.Child(fmt.Sprintf("%s: %s", k, v.AsString()))
	}
	ts := m.selectedLog.Log.Timestamp().AsTime()
	tsobserved := m.selectedLog.Log.ObservedTimestamp().AsTime()

	t := tree.Root("Body: " + m.selectedLog.Log.Body().Str()).
		Child("Time: " + ts.Format(time.RFC3339)).
		Child(fmt.Sprintf("Time (Observed): %s (%s)", tsobserved.Format(time.RFC3339), tsobserved.Sub(ts))).
		Child(fmt.Sprintf("Time (Arrived): %s (%s)", m.selectedLog.Received.Format(time.RFC3339), m.selectedLog.Received.Sub(ts))).
		Child(fmt.Sprintf("Severity: %s (%d)", m.selectedLog.Log.SeverityText(), m.selectedLog.Log.SeverityNumber())).
		Child("Event Name: " + m.selectedLog.Log.EventName()).
		Child(attrs).
		Child(tree.New().Root("Scope").
			Child("Schema URL: " + m.selectedLog.ScopeLogs.SchemaUrl()).
			Child("Scope Name: " + m.selectedLog.ScopeLogs.Scope().Name()).
			Child("Scope Version: " + m.selectedLog.ScopeLogs.Scope().Version()).
			Child(sattrs)).
		Child(tree.New().Root("Resource").
			Child("Schema URL: " + m.selectedLog.ResourceLogs.SchemaUrl()).
			Child(rattrs))
	if !m.selectedLog.Log.TraceID().IsEmpty() {
		t.Child("TraceID: " + m.selectedLog.Log.TraceID().String())
	}
	if !m.selectedLog.Log.SpanID().IsEmpty() {
		t.Child("SpanID: " + m.selectedLog.Log.SpanID().String())
	}
	m.details.SetContent(t.String())
}

func (m *logsModel) selectUp() {
	if m.selectedLog == nil {
		return
	}
	var prev *server.Log
	for _, l := range m.logs {
		if l == m.selectedLog && prev != nil {
			m.selectLog(prev)
			break
		} else {
			prev = l
		}
	}
}

func (m *logsModel) selectDown() {
	if m.selectedLog == nil && len(m.logs) > 0 {
		m.selectLog(m.logs[0])
	} else if m.selectedLog != nil {
		next := false
		for _, l := range m.logs {
			if l == m.selectedLog {
				next = true
			} else if next {
				m.selectLog(l)
				break
			}
		}
	}
}

func (m *logsModel) Help() []key.Binding {
	return m.keyMap.Help()
}
