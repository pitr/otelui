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
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	logs "go.opentelemetry.io/proto/otlp/logs/v1"

	"github.com/pitr/otelui/server"
	"github.com/pitr/otelui/ui/components"
)

type logsModel struct {
	view     components.Splitview[*components.Viewport]
	lastLogs int
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
	return nil
}

func (m logsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case server.ConsumeEvent:
		if m.lastLogs != msg.Logs {
			m.lastLogs = msg.Logs
			m.updateMainContent()
		}
	default:
		m.view, cmd = m.view.Update(msg)
		return m, cmd
	}
	return m, cmd
}

func (m logsModel) View() string {
	return m.view.View()
}

func (m *logsModel) updateMainContent() {
	var buf strings.Builder

	lines := []components.ViewRow{}
	for _, l := range server.Storage.Logs {
		s := lipgloss.NewStyle()
		switch {
		case l.Log.SeverityNumber >= logs.SeverityNumber_SEVERITY_NUMBER_ERROR:
			s = s.Foreground(lipgloss.Color("#FF6B6B"))
		case l.Log.SeverityNumber >= logs.SeverityNumber_SEVERITY_NUMBER_WARN:
			s = s.Foreground(lipgloss.Color("#FFD93D"))
		case l.Log.SeverityNumber >= logs.SeverityNumber_SEVERITY_NUMBER_INFO:
			s = s.Foreground(lipgloss.Color("#0f93fc"))
		case l.Log.SeverityNumber >= logs.SeverityNumber_SEVERITY_NUMBER_DEBUG:
			s = s.Foreground(lipgloss.Color("15"))
		}
		svc := "-"
		for _, attr := range l.ResourceLogs.Resource.Attributes {
			if attr.Key == string(semconv.ServiceNameKey) {
				svc = AnyToString(attr.Value)
				break
			}
		}
		buf.WriteString(nanoToString(l.Log.TimeUnixNano))
		buf.WriteByte(' ')
		buf.WriteString(svc)
		buf.WriteByte(' ')
		// only reset foreground (so row select works correctly)
		buf.WriteString(strings.ReplaceAll(s.Render(lipgloss.PlaceHorizontal(3, lipgloss.Left, l.Log.SeverityText)), "\x1b[0m", "\x1b[39m"))
		buf.WriteByte(' ')
		buf.WriteString(AnyToString(l.Log.Body))
		str := buf.String()
		lines = append(lines, components.ViewRow{Str: str, Yank: ansi.Strip(str), Raw: l})
		buf.Reset()
	}
	m.view.Get(0).AddContent(lines)
}

func (m *logsModel) updateDetailsContent(selected components.ViewRow) {
	selectedLog := selected.Raw.(*server.Log)
	if selectedLog == nil {
		m.view.Get(1).SetContent([]components.ViewRow{})
		return
	}

	attrs := attrsToTree("Attributes", selectedLog.Log.Attributes)
	sattrs := attrsToTree("Attributes", selectedLog.ScopeLogs.Scope.Attributes)
	rattrs := attrsToTree("Attributes", selectedLog.ResourceLogs.Resource.Attributes)
	ts := nanoToString(selectedLog.Log.TimeUnixNano)
	tsobserved := nanoToString(selectedLog.Log.ObservedTimeUnixNano)

	t := tree.Root("Body: " + AnyToString(selectedLog.Log.Body)).
		Child("Time: " + ts).
		Child(fmt.Sprintf("Time (Observed): %s (%s)", tsobserved, time.Duration(selectedLog.Log.ObservedTimeUnixNano-selectedLog.Log.TimeUnixNano))).
		Child(fmt.Sprintf("Time (Arrived): %s (%s)", nanoToString(uint64(selectedLog.Received.UnixNano())), selectedLog.Received.Add(time.Duration(selectedLog.Log.TimeUnixNano)))).
		Child(fmt.Sprintf("Severity: %s (%d)", selectedLog.Log.SeverityText, selectedLog.Log.SeverityNumber)).
		Child("Event Name: " + selectedLog.Log.EventName).
		Child(attrs).
		Child(tree.Root("Scope").
			Child("Schema URL: " + selectedLog.ScopeLogs.SchemaUrl).
			Child("Scope Name: " + selectedLog.ScopeLogs.Scope.Name).
			Child("Scope Version: " + selectedLog.ScopeLogs.Scope.Version).
			Child(sattrs)).
		Child(tree.Root("Resource").
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
		lines = append(lines, components.ViewRow{Str: l, Yank: treeTrim(l)})
	}
	m.view.Get(1).SetContent(lines)
}

func (m logsModel) Help() []key.Binding {
	return m.view.Help()
}
