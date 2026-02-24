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
	logs "go.opentelemetry.io/proto/otlp/logs/v1"

	"pitr.ca/otelui/server"
	"pitr.ca/otelui/ui/components"
	"pitr.ca/otelui/utils"
)

type logsModel struct {
	view     components.Splitview[*components.Viewport, *components.Viewport]
	lastLogs int
}

func newLogsModel(title string) tea.Model {
	m := logsModel{}
	m.view = components.NewSplitview(
		components.NewViewport(title).WithSelectFunc(m.updateDetailsContent),
		components.NewViewport("Details"),
	)
	return m
}

func (m logsModel) Init() tea.Cmd          { return nil }
func (m logsModel) Help() []key.Binding    { return m.view.Help() }
func (m logsModel) View() string           { return m.view.View() }
func (m logsModel) IsCapturingInput() bool { return m.view.Top().IsCapturingInput() }

func (m logsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case refreshMsg:
		if msg.reset {
			m.lastLogs = 0
		}
		m.updateMainContent()
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

func (m *logsModel) updateMainContent() {
	var buf strings.Builder

	lines := []components.ViewRow{}
	for _, l := range server.GetLogs() {
		var col lipgloss.TerminalColor = lipgloss.NoColor{}
		switch {
		case l.Log.SeverityNumber >= logs.SeverityNumber_SEVERITY_NUMBER_ERROR:
			col = components.ErrorColor
		case l.Log.SeverityNumber >= logs.SeverityNumber_SEVERITY_NUMBER_WARN:
			col = components.WarnColor
		case l.Log.SeverityNumber >= logs.SeverityNumber_SEVERITY_NUMBER_INFO:
			col = components.InfoColor
		case l.Log.SeverityNumber >= logs.SeverityNumber_SEVERITY_NUMBER_DEBUG:
			col = components.DebugColor
		}

		buf.WriteString(nanoToString(l.Log.TimeUnixNano))
		buf.WriteByte(' ')
		buf.WriteString(resourceToServiceName(l.ResourceLogs.Resource))
		buf.WriteByte(' ')
		buf.WriteString(renderForeground(col, lipgloss.PlaceHorizontal(3, lipgloss.Left, l.Log.SeverityText)))
		buf.WriteByte(' ')
		buf.WriteString(utils.AnyToString(l.Log.Body))
		str := buf.String()
		search := attrsSearch(l.Log.Attributes, l.ScopeLogs.Scope.Attributes, l.ResourceLogs.Resource.Attributes)
		lines = append(lines, components.ViewRow{Str: str, Raw: l, Search: search})
		buf.Reset()
	}
	m.view.Top().SetContent(lines)
}

func (m *logsModel) updateDetailsContent(selected components.ViewRow) {
	selectedLog, _ := selected.Raw.(*server.Log)
	if selectedLog == nil {
		m.view.Bot().SetContent([]components.ViewRow{})
		return
	}

	ts := nanoToString(selectedLog.Log.TimeUnixNano)
	tsobserved := nanoToString(selectedLog.Log.ObservedTimeUnixNano)

	t := tree.Root("Body: " + utils.AnyToString(selectedLog.Log.Body)).
		Child("Time: " + ts).
		Child(fmt.Sprintf("Time (Observed): %s (%s later)", tsobserved, time.Duration(selectedLog.Log.ObservedTimeUnixNano-selectedLog.Log.TimeUnixNano))).
		Child(fmt.Sprintf("Time (Arrived): %s (%s later)", nanoToString(uint64(selectedLog.Received.UnixNano())), time.Duration(selectedLog.Received.UnixNano()-int64(selectedLog.Log.TimeUnixNano))))
	if attrs, set := attrsToTree("Attributes", selectedLog.Log.Attributes); set {
		t.Child(attrs)
	}
	if sattrs, set := attrsToTree("Attributes", selectedLog.ScopeLogs.Scope.Attributes); set {
		t.Child(tree.Root(fmt.Sprintf("Scope for %s (%s)", selectedLog.ScopeLogs.Scope.Name, selectedLog.ScopeLogs.Scope.Version)).Child(sattrs))
	}
	if rattrs, set := attrsToTree("Resource Attributes", selectedLog.ResourceLogs.Resource.Attributes); set {
		t.Child(rattrs)
	}
	if selectedLog.Log.EventName != "" {
		t.Child("Event Name: " + selectedLog.Log.EventName)
	}
	if len(selectedLog.Log.TraceId) != 0 {
		t.Child("TraceID: " + hex.EncodeToString(selectedLog.Log.TraceId))
	}
	if len(selectedLog.Log.SpanId) != 0 {
		t.Child("SpanID: " + hex.EncodeToString(selectedLog.Log.SpanId))
	}
	lines := []components.ViewRow{}
	for l := range strings.SplitSeq(t.String(), "\n") {
		lines = append(lines, components.ViewRow{Str: l})
	}
	m.view.Bot().SetContent(lines)
}
