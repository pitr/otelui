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
	v1 "go.opentelemetry.io/proto/otlp/trace/v1"

	"pitr.ca/otelui/server"
	"pitr.ca/otelui/ui/components"
)

const (
	traceMinPane  = 6
	tracePaneStep = 2
)

type keyMapTraces struct {
	Increase key.Binding
	Decrease key.Binding
	Next     key.Binding
	Prev     key.Binding
	Enter    key.Binding
	Esc      key.Binding
	GoToLogs key.Binding
}

type tracesModel struct {
	views     [3]*components.Viewport
	focus     int
	w         int
	h         [3]int
	lastSpans int
	keyMap    keyMapTraces
	selected  *server.Trace
}

func newTracesModel(title string) tea.Model {
	m := &tracesModel{
		keyMap: keyMapTraces{
			Increase: key.NewBinding(key.WithKeys("="), key.WithHelp("- =", "resize")),
			Decrease: key.NewBinding(key.WithKeys("-")),
			Next:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
			Prev:     key.NewBinding(key.WithKeys("shift+tab")),
			Enter:    key.NewBinding(key.WithKeys("enter")),
			Esc:      key.NewBinding(key.WithKeys("esc")),
			GoToLogs: key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "jump to logs")),
		},
	}
	m.views = [3]*components.Viewport{
		components.NewViewport(title).WithSelectFunc(m.updateSpanTree),
		components.NewViewport("Spans").WithSelectFunc(m.updateSpanDetails),
		components.NewViewport("Details"),
	}
	m.views[0].SetFocus(true)
	return m
}

func (m *tracesModel) Init() tea.Cmd          { return nil }
func (m *tracesModel) IsCapturingInput() bool { return m.views[m.focus].IsCapturingInput() }

func (m *tracesModel) Help() []key.Binding {
	if m.IsCapturingInput() {
		return append([]key.Binding{m.keyMap.Next}, m.views[m.focus].Help()...)
	}
	bindings := []key.Binding{m.keyMap.Next, m.keyMap.Increase}
	bindings = append(bindings, m.views[m.focus].Help()...)
	if m.selected != nil {
		bindings = append(bindings, m.keyMap.GoToLogs)
	}
	return bindings
}

func (m *tracesModel) View() string {
	return lipgloss.JoinVertical(lipgloss.Left, m.views[0].View(), m.views[1].View(), m.views[2].View())
}

func (m *tracesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case refreshMsg:
		if msg.reset {
			m.lastSpans = 0
		}
		m.updateTraceList()
	case server.ConsumeEvent:
		if m.lastSpans != msg.Spans {
			m.lastSpans = msg.Spans
			m.updateTraceList()
		}
	case navigateMsg:
		return m, m.views[0].SetSearch(msg.filter)
	case tea.WindowSizeMsg:
		m.w = msg.Width
		total := msg.Height
		m.h[0] = total / 4
		m.h[1] = total / 4
		m.h[2] = total - m.h[0] - m.h[1]
		m.resizeViewports()
	case tea.KeyMsg:
		capturing := m.views[m.focus].IsCapturingInput()
		switch {
		case key.Matches(msg, m.keyMap.Next):
			m.setFocus((m.focus + 1) % 3)
		case key.Matches(msg, m.keyMap.Prev):
			m.setFocus((m.focus + 2) % 3)
		case key.Matches(msg, m.keyMap.Enter) && m.focus < 2:
			m.setFocus(m.focus + 1)
		case key.Matches(msg, m.keyMap.Esc) && m.focus > 0 && !capturing:
			m.setFocus(m.focus - 1)
		case key.Matches(msg, m.keyMap.Increase) && !capturing:
			other := (m.focus + 1) % 3
			if m.h[other] >= traceMinPane+tracePaneStep {
				m.h[other] -= tracePaneStep
				m.h[m.focus] += tracePaneStep
				m.resizeViewports()
			}
		case key.Matches(msg, m.keyMap.Decrease) && !capturing:
			other := (m.focus + 1) % 3
			if m.h[m.focus] >= traceMinPane+tracePaneStep {
				m.h[m.focus] -= tracePaneStep
				m.h[other] += tracePaneStep
				m.resizeViewports()
			}
		case key.Matches(msg, m.keyMap.GoToLogs) && !capturing:
			if m.selected != nil {
				filter := m.selected.TraceID[:6]
				return m, func() tea.Msg { return navigateMsg{mRootLogs, filter} }
			}
		default:
			m.viewAt(m.focus).Update(msg)
		}
	}
	return m, nil
}

func (m *tracesModel) setFocus(pane int) {
	m.focus = pane
	for i := range m.views {
		m.views[i].SetFocus(i == pane)
	}
}

func (m *tracesModel) viewAt(pane int) *components.Viewport {
	return m.views[pane]
}

func (m *tracesModel) resizeViewports() {
	for i := range m.views {
		m.views[i].Update(tea.WindowSizeMsg{Width: m.w, Height: m.h[i]})
	}
}

func (m *tracesModel) updateTraceList() {
	traces := server.GetTraces()
	rows := make([]components.ViewRow, len(traces))
	for i, t := range traces {
		var root *server.Span
		var minStart, maxEnd uint64
		for _, s := range t.Spans {
			if len(s.Span.ParentSpanId) == 0 {
				root = s
			}
			if minStart == 0 {
				minStart = s.Span.StartTimeUnixNano
			}
			minStart = min(minStart, s.Span.StartTimeUnixNano)
			maxEnd = max(maxEnd, s.Span.EndTimeUnixNano)
		}
		svc, name, ts := "", "(no root span)", ""
		if root != nil {
			svc = resourceToServiceName(root.Resource)
			name = root.Span.Name
			ts = nanoToString(root.Span.StartTimeUnixNano)
		}
		dur := time.Duration(maxEnd - minStart)
		str := fmt.Sprintf("%s %s svc=%s name=%s dur=%s (%d spans)", ts, t.TraceID[:6], svc, name, dur, len(t.Spans))
		rows[i] = components.ViewRow{Str: str, Raw: t}
	}
	m.views[0].SetContent(rows)
}

func (m *tracesModel) updateSpanTree(selected components.ViewRow) {
	trace, _ := selected.Raw.(*server.Trace)
	m.selected = trace
	if trace == nil {
		m.views[1].SetContent([]components.ViewRow{})
		m.views[2].SetContent([]components.ViewRow{})
		return
	}

	var (
		traceStart, traceEnd uint64
		spanByID             = map[string]*server.Span{}
		children             = map[string][]*server.Span{}
		roots                []*server.Span
	)
	for _, s := range trace.Spans {
		if traceStart == 0 || s.Span.StartTimeUnixNano < traceStart {
			traceStart = s.Span.StartTimeUnixNano
		}
		if s.Span.EndTimeUnixNano > traceEnd {
			traceEnd = s.Span.EndTimeUnixNano
		}
		spanByID[hex.EncodeToString(s.Span.SpanId)] = s
	}
	for _, s := range trace.Spans {
		pid := hex.EncodeToString(s.Span.ParentSpanId)
		if len(s.Span.ParentSpanId) == 0 || spanByID[pid] == nil {
			roots = append(roots, s)
		} else {
			children[pid] = append(children[pid], s)
		}
	}

	var spanOrder []*server.Span

	var buildNode func(s *server.Span) *tree.Tree
	buildNode = func(s *server.Span) *tree.Tree {
		spanOrder = append(spanOrder, s)
		dur := time.Duration(s.Span.EndTimeUnixNano - s.Span.StartTimeUnixNano)
		status := ""
		if s.Span.Status != nil {
			switch s.Span.Status.Code {
			case v1.Status_STATUS_CODE_UNSET:
			case v1.Status_STATUS_CODE_OK:
				status = " " + renderForeground(components.InfoColor, s.Span.Status.Code.String())
			case v1.Status_STATUS_CODE_ERROR:
				status = " " + renderForeground(components.ErrorColor, s.Span.Status.Code.String())
			}
		}
		node := tree.Root(fmt.Sprintf("%s %s %s%s", resourceToServiceName(s.Resource), s.Span.Name, dur, status))
		for _, child := range children[hex.EncodeToString(s.Span.SpanId)] {
			node.Child(buildNode(child))
		}
		return node
	}

	trees := make([]string, len(roots))
	for i, s := range roots {
		trees[i] = buildNode(s).String()
	}

	treeLines := strings.Split(strings.Join(trees, "\n"), "\n")
	maxW := 0
	for _, l := range treeLines {
		maxW = max(maxW, lipgloss.Width(l))
	}
	barW := max(20, m.w-maxW-3) // border + space
	rows := make([]components.ViewRow, len(treeLines))
	for i, line := range treeLines {
		s := spanOrder[i].Span
		pad := strings.Repeat(" ", maxW-lipgloss.Width(line)+1) // space
		str := line + pad + ganttBar(s.StartTimeUnixNano, s.EndTimeUnixNano, traceStart, traceEnd, barW)
		rows[i] = components.ViewRow{Str: str, Raw: spanOrder[i]}
	}
	m.views[1].SetContent(rows)
}

func (m *tracesModel) updateSpanDetails(selected components.ViewRow) {
	s, _ := selected.Raw.(*server.Span)
	if s == nil {
		m.views[2].SetContent([]components.ViewRow{})
		return
	}

	sp := s.Span

	parentID := "(root)"
	if len(sp.ParentSpanId) > 0 {
		parentID = hex.EncodeToString(sp.ParentSpanId)
	}

	t := tree.Root(fmt.Sprintf("%s (%s)", sp.Name, resourceToServiceName(s.Resource))).
		Child("TraceID: " + hex.EncodeToString(sp.TraceId)).
		Child("SpanID: " + hex.EncodeToString(sp.SpanId)).
		Child("Parent: " + parentID).
		Child("Start: " + nanoToString(sp.StartTimeUnixNano)).
		Child("Duration: " + time.Duration(sp.EndTimeUnixNano-sp.StartTimeUnixNano).String()).
		Child("Kind: " + sp.Kind.String())

	if sp.Status != nil {
		statusMsg := sp.Status.Message
		t.Child(fmt.Sprintf("Status: %s %s", sp.Status.Code.String(), statusMsg))
	}

	if attrs, set := attrsToTree("Attributes", sp.Attributes); set {
		t.Child(attrs)
	}
	if s.Resource != nil {
		if rattrs, set := attrsToTree("Resource Attributes", s.Resource.Attributes); set {
			t.Child(rattrs)
		}
	}
	if s.Scope != nil && s.Scope.Name != "" {
		t.Child(fmt.Sprintf("Scope: %s (version=%s)", s.Scope.Name, s.Scope.Version))
	}
	if len(sp.Events) > 0 {
		events := tree.Root(fmt.Sprintf("Events (%d):", len(sp.Events)))
		for _, e := range sp.Events {
			et := tree.Root(fmt.Sprintf("(%s) %s", time.Duration(e.TimeUnixNano-sp.StartTimeUnixNano), e.Name))
			if attrs, set := attrsToTree("Attributes", e.Attributes); set {
				et.Child(attrs)
			}
			events.Child(et)
		}
		t.Child(events)
	}

	lines := []components.ViewRow{}
	for l := range strings.SplitSeq(t.String(), "\n") {
		lines = append(lines, components.ViewRow{Str: l})
	}
	m.views[2].SetContent(lines)
}

func ganttBar(startNano, endNano, traceStart, traceEnd uint64, w int) string {
	if w <= 0 || traceStart == traceEnd {
		return strings.Repeat(" ", max(0, w))
	}
	left := int(float64(startNano-traceStart) / float64(traceEnd-traceStart) * float64(w))
	fill := max(1, int(float64(endNano-startNano)/float64(traceEnd-traceStart)*float64(w)))
	if left+fill > w {
		fill = w - left
	}
	return strings.Repeat(" ", left) + strings.Repeat("▒", fill) + strings.Repeat(" ", w-left-fill)
}
