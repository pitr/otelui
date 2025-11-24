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
	"github.com/charmbracelet/x/ansi"
	"github.com/pitr/otelui/server"
	"github.com/pitr/otelui/ui/components"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	plog "go.opentelemetry.io/proto/otlp/logs/v1"
)

type logsModel struct {
	view components.Splitview[*components.Viewport]
}

func newLogsModel() tea.Model {
	m := logsModel{}
	m.view = components.NewSplitview(
		components.NewViewport(m.updateDetailsContent),
		components.NewViewport(nil),
	)
	return m
}

func (m logsModel) Init() tea.Cmd {
	return func() tea.Msg { return server.QueryLogs() }
}

func (m logsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case server.NewLogsEvent:
		m.updateMainContent(msg.NewLogs)
	case server.QueriedLogsEvent:
		m.updateMainContent(msg.Logs)
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
// 	selectedLog = l
// 	if l == nil {
// 		m.mode = mLogsFocusMain
// 	}
// 	m.renderMain()
// 	m.renderDetails()
// }

func (*logsModel) renderLog(l *server.Log) (str, yank string) {
	var buf strings.Builder
	s := lipgloss.NewStyle()
	switch {
	case l.Log.SeverityNumber >= plog.SeverityNumber_SEVERITY_NUMBER_ERROR:
		s = s.Foreground(lipgloss.Color("#FF6B6B"))
	case l.Log.SeverityNumber >= plog.SeverityNumber_SEVERITY_NUMBER_WARN:
		s = s.Foreground(lipgloss.Color("#FFD93D"))
	case l.Log.SeverityNumber >= plog.SeverityNumber_SEVERITY_NUMBER_INFO:
		s = s.Foreground(lipgloss.Color("#0f93fc"))
	case l.Log.SeverityNumber >= plog.SeverityNumber_SEVERITY_NUMBER_DEBUG:
		s = s.Foreground(lipgloss.Color("15"))
	}
	svc := "-"
	for _, attr := range l.ResourceLogs.Resource.Attributes {
		if attr.Key == string(semconv.ServiceNameKey) {
			svc = AnyToString(attr.Value)
			break
		}
	}
	buf.WriteString(time.Unix(0, int64(l.Log.TimeUnixNano)).UTC().Format(time.RFC3339))
	buf.WriteByte(' ')
	buf.WriteString(svc)
	buf.WriteByte(' ')
	buf.WriteString(strings.ReplaceAll(s.Render(lipgloss.PlaceHorizontal(3, lipgloss.Left, l.Log.SeverityText)), "\x1b[0m", "\x1b[39m"))
	// if l == selectedLog {
	// 	buf.WriteString(m._selected.Render(str))
	// 	buf.WriteString(m._selected.Render(" "))
	// 	buf.WriteString(m._selected.Render(AnyToString(l.Log.Body)))
	buf.WriteByte(' ')
	buf.WriteString(AnyToString(l.Log.Body))
	return buf.String(), ansi.Strip(buf.String())
}

func (m *logsModel) updateMainContent(logs []*server.Log) {
	lines := []components.ViewRow{}
	for _, l := range logs {
		s, y := m.renderLog(l)
		lines = append(lines, components.ViewRow{Str: s, Yank: y, Raw: l})
	}
	m.view.Get(0).AddContent(lines)
}

func (m *logsModel) updateDetailsContent(selected components.ViewRow) {
	selectedLog := selected.Raw.(*server.Log)
	if selectedLog == nil {
		m.view.Get(1).SetContent([]components.ViewRow{})
		return
	}

	attrs := tree.New().Root("Attributes")
	for _, a := range selectedLog.Log.Attributes {
		attrs = attrs.Child(fmt.Sprintf("%s: %s", a.Key, AnyToString(a.Value)))
	}
	sattrs := tree.New().Root("Attributes")
	for _, a := range selectedLog.ScopeLogs.Scope.Attributes {
		sattrs = sattrs.Child(fmt.Sprintf("%s: %s", a.Key, AnyToString(a.Value)))
	}
	rattrs := tree.New().Root("Attributes")
	for _, a := range selectedLog.ResourceLogs.Resource.Attributes {
		rattrs = rattrs.Child(fmt.Sprintf("%s: %s", a.Key, AnyToString(a.Value)))
	}
	ts := time.Unix(0, int64(selectedLog.Log.TimeUnixNano))
	tsobserved := time.Unix(0, int64(selectedLog.Log.ObservedTimeUnixNano))

	t := tree.Root("Body: " + AnyToString(selectedLog.Log.Body)).
		Child("Time: " + ts.Format(time.RFC3339)).
		Child(fmt.Sprintf("Time (Observed): %s (%s)", tsobserved.Format(time.RFC3339), tsobserved.Sub(ts))).
		Child(fmt.Sprintf("Time (Arrived): %s (%s)", selectedLog.Received.Format(time.RFC3339), selectedLog.Received.Sub(ts))).
		Child(fmt.Sprintf("Severity: %s (%d)", selectedLog.Log.SeverityText, selectedLog.Log.SeverityNumber)).
		Child("Event Name: " + selectedLog.Log.EventName).
		Child(attrs).
		Child(tree.New().Root("Scope").
			Child("Schema URL: " + selectedLog.ScopeLogs.SchemaUrl).
			Child("Scope Name: " + selectedLog.ScopeLogs.Scope.Name).
			Child("Scope Version: " + selectedLog.ScopeLogs.Scope.Version).
			Child(sattrs)).
		Child(tree.New().Root("Resource").
			Child("Schema URL: " + selectedLog.ResourceLogs.SchemaUrl).
			Child(rattrs))
	if len(selectedLog.Log.TraceId) != 0 {
		t.Child("TraceID: " + hex.EncodeToString(selectedLog.Log.TraceId))
	}
	if len(selectedLog.Log.SpanId) != 0 {
		t.Child("SpanID: " + hex.EncodeToString(selectedLog.Log.SpanId))
	}
	lines := []components.ViewRow{}
	for l := range strings.SplitSeq(t.String(), "\n") {
		lines = append(lines, components.ViewRow{Str: l, Yank: l, Raw: l})
	}
	m.view.Get(1).SetContent(lines)
}

func (m logsModel) Help() []key.Binding {
	return m.view.Help()
}
