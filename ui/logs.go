package ui

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	"github.com/pitr/otelui/server"
	"github.com/pitr/otelui/ui/components"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	plog "go.opentelemetry.io/proto/otlp/logs/v1"
)

type keyMapLogs struct {
	Up   key.Binding
	Down key.Binding
}

func (k keyMapLogs) Help() []key.Binding {
	return []key.Binding{k.Up}
}

type QueriedLogs []*server.Log

type logsModel struct {
	lastLogs    int
	keyMap      keyMapLogs
	logs        []*server.Log
	selectedLog *server.Log
	_selected   lipgloss.Style
	view        components.Splitview[*components.Viewport]
}

func newLogsModel() tea.Model {
	return logsModel{
		keyMap: keyMapLogs{
			Up:   key.NewBinding(key.WithKeys("up"), key.WithHelp("↑/↓", "select")),
			Down: key.NewBinding(key.WithKeys("down")),
		},
		logs:      []*server.Log{},
		_selected: lipgloss.NewStyle().Background(components.FadedColor),
		view: components.NewSplitview(
			components.NewViewport(true),
			components.NewViewport(false),
		),
	}
}

func (m logsModel) Init() tea.Cmd {
	return nil
}

func (m logsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case server.ConsumeEvent:
		if msg.Logs != m.lastLogs {
			m.lastLogs = msg.Logs
			return m, func() tea.Msg { return QueriedLogs(server.QueryLogs(100)) }
		}
	case QueriedLogs:
		m.logs = msg
		m.renderMain()
	default:
		m.view, cmd = m.view.Update(msg)
		return m, cmd
	}
	return m, cmd
}

func (m logsModel) View() string {
	return m.view.View()
}

// func (m *logsModel) selectLog(l *server.Log) {
// 	m.selectedLog = l
// 	if l == nil {
// 		m.mode = mLogsFocusMain
// 	}
// 	m.renderMain()
// 	m.renderDetails()
// }

func (m *logsModel) renderMain() {
	var buf strings.Builder
	for i, l := range m.logs {
		if i > 0 {
			buf.WriteByte('\n')
		}
		s := lipgloss.NewStyle()
		switch {
		case l.Log.SeverityNumber >= plog.SeverityNumber_SEVERITY_NUMBER_ERROR:
			s = s.Foreground(lipgloss.Color("9"))
		case l.Log.SeverityNumber >= plog.SeverityNumber_SEVERITY_NUMBER_WARN:
			s = s.Foreground(lipgloss.Color("11"))
		case l.Log.SeverityNumber >= plog.SeverityNumber_SEVERITY_NUMBER_INFO:
			s = s.Foreground(lipgloss.Color("14"))
		case l.Log.SeverityNumber >= plog.SeverityNumber_SEVERITY_NUMBER_DEBUG:
			s = s.Foreground(lipgloss.Color("15"))
		}
		svc := "-"
		for _, attr := range l.ResourceLogs.Resource.Attributes {
			if attr.Key == string(semconv.ServiceNameKey) {
				svc = attr.Value.GetStringValue()
				break
			}
		}
		str := strings.Join([]string{
			time.Unix(0, int64(l.Log.TimeUnixNano)).UTC().Format(time.RFC3339),
			svc,
			s.Render(lipgloss.PlaceHorizontal(3, lipgloss.Left, l.Log.SeverityText)),
		}, " ")
		if l == m.selectedLog {
			buf.WriteString(m._selected.Render(str))
			buf.WriteString(m._selected.Render(" "))
			buf.WriteString(m._selected.Render(l.Log.Body.GetStringValue()))
		} else {
			buf.WriteString(str)
			buf.WriteByte(' ')
			buf.WriteString(l.Log.Body.GetStringValue())
		}
	}
	m.view.Get(0).SetContent(buf.String())
}

func (m *logsModel) renderDetails() {
	if m.selectedLog == nil {
		m.view.Get(1).SetContent("")
		return
	}

	attrs := tree.New().Root("Attributes")
	for _, a := range m.selectedLog.Log.Attributes {
		attrs = attrs.Child(fmt.Sprintf("%s: %s", a.Key, a.Value.GetStringValue()))
	}
	sattrs := tree.New().Root("Attributes")
	for _, a := range m.selectedLog.ScopeLogs.Scope.Attributes {
		sattrs = sattrs.Child(fmt.Sprintf("%s: %s", a.Key, a.Value.GetStringValue()))
	}
	rattrs := tree.New().Root("Attributes")
	for _, a := range m.selectedLog.ResourceLogs.Resource.Attributes {
		rattrs = rattrs.Child(fmt.Sprintf("%s: %s", a.Key, a.Value.GetStringValue()))
	}
	ts := time.Unix(0, int64(m.selectedLog.Log.TimeUnixNano))
	tsobserved := time.Unix(0, int64(m.selectedLog.Log.ObservedTimeUnixNano))

	t := tree.Root("Body: " + m.selectedLog.Log.Body.GetStringValue()).
		Child("Time: " + ts.Format(time.RFC3339)).
		Child(fmt.Sprintf("Time (Observed): %s (%s)", tsobserved.Format(time.RFC3339), tsobserved.Sub(ts))).
		Child(fmt.Sprintf("Time (Arrived): %s (%s)", m.selectedLog.Received.Format(time.RFC3339), m.selectedLog.Received.Sub(ts))).
		Child(fmt.Sprintf("Severity: %s (%d)", m.selectedLog.Log.SeverityText, m.selectedLog.Log.SeverityNumber)).
		Child("Event Name: " + m.selectedLog.Log.EventName).
		Child(attrs).
		Child(tree.New().Root("Scope").
			Child("Schema URL: " + m.selectedLog.ScopeLogs.SchemaUrl).
			Child("Scope Name: " + m.selectedLog.ScopeLogs.Scope.Name).
			Child("Scope Version: " + m.selectedLog.ScopeLogs.Scope.Version).
			Child(sattrs)).
		Child(tree.New().Root("Resource").
			Child("Schema URL: " + m.selectedLog.ResourceLogs.SchemaUrl).
			Child(rattrs))
	if len(m.selectedLog.Log.TraceId) != 0 {
		t.Child("TraceID: " + hex.EncodeToString(m.selectedLog.Log.TraceId))
	}
	if len(m.selectedLog.Log.SpanId) != 0 {
		t.Child("SpanID: " + hex.EncodeToString(m.selectedLog.Log.SpanId))
	}
	m.view.Get(1).SetContent(t.String())
}

func (m logsModel) Help() []key.Binding {
	return append(m.view.Help(), m.keyMap.Help()...)
}
