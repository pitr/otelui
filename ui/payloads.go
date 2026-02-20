package ui

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/tree"
	logs "go.opentelemetry.io/proto/otlp/logs/v1"
	metrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	traces "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/pitr/otelui/server"
	"github.com/pitr/otelui/ui/components"
	"github.com/pitr/otelui/utils"
)

type payloadsModel struct {
	view components.Splitview[*components.Viewport, *components.Viewport]

	lastPayloads int
}

func newPayloadsModel(title string) tea.Model {
	m := payloadsModel{}
	m.view = components.NewSplitview(
		components.NewViewport(title, m.updateDetailsContent).WithSearch(),
		components.NewViewport("Details", nil),
	)
	return m
}

func (m payloadsModel) Init() tea.Cmd          { return nil }
func (m payloadsModel) Help() []key.Binding    { return m.view.Help() }
func (m payloadsModel) View() string           { return m.view.View() }
func (m payloadsModel) IsCapturingInput() bool { return m.view.Top().IsCapturingInput() }

func (m payloadsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case refreshMsg:
		if msg.reset {
			m.lastPayloads = 0
		}
		m.updateMainContent()
	case server.ConsumeEvent:
		if m.lastPayloads != msg.Payloads {
			m.lastPayloads = msg.Payloads
			m.updateMainContent()
		}
	default:
		m.view, cmd = m.view.Update(msg)
	}
	return m, cmd
}

func (m *payloadsModel) updateMainContent() {
	payloads := []components.ViewRow{}
	for _, p := range server.GetPayloads() {
		t := "unknown"
		var search strings.Builder
		switch pp := p.Payload.(type) {
		case []*logs.ResourceLogs:
			t = "logs"
			for _, rl := range pp {
				search.WriteString(attrsSearch(rl.Resource.Attributes))
				for _, sl := range rl.ScopeLogs {
					search.WriteString(attrsSearch(sl.Scope.Attributes))
					for _, lr := range sl.LogRecords {
						search.WriteString(utils.AnyToString(lr.Body))
						search.WriteByte(' ')
						search.WriteString(attrsSearch(lr.Attributes))
					}
				}
			}
		case []*traces.ResourceSpans:
			t = "spans"
			for _, rs := range pp {
				search.WriteString(attrsSearch(rs.Resource.Attributes))
				for _, ss := range rs.ScopeSpans {
					search.WriteString(attrsSearch(ss.Scope.Attributes))
					for _, s := range ss.Spans {
						search.WriteString(s.Name)
						search.WriteByte(' ')
						search.WriteString(attrsSearch(s.Attributes))
					}
				}
			}
		case []*metrics.ResourceMetrics:
			t = "metrics"
			for _, rm := range pp {
				search.WriteString(attrsSearch(rm.Resource.Attributes))
				for _, sm := range rm.ScopeMetrics {
					search.WriteString(attrsSearch(sm.Scope.Attributes))
					for _, metric := range sm.Metrics {
						search.WriteString(metric.Name)
						search.WriteByte(' ')
					}
				}
			}
		}
		s := fmt.Sprintf("%s %3d %s", nanoToString(uint64(p.Received.UnixNano())), p.Num, t)
		payloads = append(payloads, components.ViewRow{Str: s, Yank: s, Search: search.String(), Raw: p})
	}
	m.view.Top().SetContent(payloads)
}

func (m *payloadsModel) updateDetailsContent(selected components.ViewRow) {
	row, _ := selected.Raw.(*server.Payload)
	if row == nil {
		m.view.Bot().SetContent([]components.ViewRow{})
		return
	}
	lines := []components.ViewRow{}
	switch p := row.Payload.(type) {
	case []*logs.ResourceLogs:
		t := tree.Root(fmt.Sprintf("ResourceLogs (%d)", len(p)))
		for _, rl := range p {
			t2 := tree.Root(fmt.Sprintf("ScopeLogs (%d)", len(rl.ScopeLogs)))
			for _, sl := range rl.ScopeLogs {
				t3 := tree.Root(fmt.Sprintf("LogRecords (%d)", len(sl.LogRecords)))
				for _, lr := range sl.LogRecords {
					attrs, _ := attrsToTree("Attributes", lr.Attributes)
					t3 = t3.Child(tree.Root("LogRecord").
						Child(fmt.Sprintf("TimeUnixNano: %s (raw=%d)", nanoToString(lr.TimeUnixNano), lr.TimeUnixNano)).
						Child(fmt.Sprintf("ObservedTimeUnixNano: %s (raw=%d)", nanoToString(lr.ObservedTimeUnixNano), lr.ObservedTimeUnixNano)).
						Child("Body: " + utils.AnyToString(lr.Body)).
						Child(fmt.Sprintf("SeverityNumber: %s (raw=%d)", lr.SeverityNumber.String(), lr.SeverityNumber)).
						Child("SeverityText: " + lr.SeverityText).
						Child(attrs).
						Child(fmt.Sprintf("DroppedAttributesCount: %d", lr.DroppedAttributesCount)).
						Child(fmt.Sprintf("Flags: %d", lr.Flags)).
						Child("TraceId: " + hex.EncodeToString(lr.TraceId)).
						Child("SpanId: " + hex.EncodeToString(lr.SpanId)).
						Child("EventName: " + lr.EventName))
				}
				t2 = t2.Child(tree.Root("ScopeLog").
					Child("SchemaURL: " + sl.SchemaUrl).
					Child(scopeToTree(sl.Scope)).
					Child(t3))
			}
			rattrs, _ := attrsToTree("Attributes", rl.Resource.Attributes)
			t = t.Child(tree.Root("ResourceLog").
				Child("Schema URL: " + rl.SchemaUrl).
				Child(rattrs).
				Child(t2))
		}
		for l := range strings.SplitSeq(t.String(), "\n") {
			lines = append(lines, components.ViewRow{Str: l, Yank: treeTrim(l)})
		}
	case []*traces.ResourceSpans:
		t := tree.Root(fmt.Sprintf("ResourceSpans (%d)", len(p)))
		for _, rs := range p {
			t2 := tree.Root(fmt.Sprintf("ScopeSpans (%d)", len(rs.ScopeSpans)))
			for _, ss := range rs.ScopeSpans {
				t3 := tree.Root(fmt.Sprintf("Spans (%d)", len(ss.Spans)))
				for _, s := range ss.Spans {
					events := tree.Root("Events")
					for _, e := range s.Events {
						eattrs, _ := attrsToTree("Attributes", e.Attributes)
						events = events.Child(tree.Root("Event").
							Child("TimeUnixNano: " + nanoToString(e.TimeUnixNano)).
							Child("Name: " + e.Name).
							Child(eattrs).
							Child(fmt.Sprintf("DroppedAttributesCount: %d", e.DroppedAttributesCount)))
					}
					links := tree.Root("Links")
					for _, l := range s.Links {
						lattrs, _ := attrsToTree("Attributes", l.Attributes)
						links = links.Child(tree.Root("Link").
							Child("TraceId: " + hex.EncodeToString(l.TraceId)).
							Child("SpanId: " + hex.EncodeToString(l.SpanId)).
							Child("TraceState: " + l.TraceState).
							Child(lattrs).
							Child(fmt.Sprintf("Flags: %d", l.Flags)).
							Child(fmt.Sprintf("DroppedAttributesCount: %d", l.DroppedAttributesCount)))
					}
					attrs, _ := attrsToTree("Attributes", s.Attributes)
					t3 = t3.Child(tree.Root("Span").
						Child("TraceId: " + hex.EncodeToString(s.TraceId)).
						Child("SpanId: " + hex.EncodeToString(s.SpanId)).
						Child("ParentSpanId: " + hex.EncodeToString(s.ParentSpanId)).
						Child(fmt.Sprintf("StartTimeUnixNano: %s (raw=%d)", nanoToString(s.StartTimeUnixNano), s.StartTimeUnixNano)).
						Child(fmt.Sprintf("EndTimeUnixNano: %s (raw=%d)", nanoToString(s.EndTimeUnixNano), s.EndTimeUnixNano)).
						Child("Name: " + s.Name).
						Child("Kind: " + s.Kind.String()).
						Child("Status.Message: " + s.Status.Message).
						Child(fmt.Sprintf("Status.Code: %s (raw=%d)", s.Status.Code.String(), s.Status.Code)).
						Child("TraceState: " + s.TraceState).
						Child(fmt.Sprintf("Flags: %d", s.Flags)).
						Child(attrs).
						Child(events).
						Child(links).
						Child(fmt.Sprintf("DroppedAttributesCount: %d", s.DroppedAttributesCount)).
						Child(fmt.Sprintf("DroppedEventsCount: %d", s.DroppedEventsCount)).
						Child(fmt.Sprintf("DroppedLinksCount: %d", s.DroppedLinksCount)))
				}
				t2 = t2.Child(tree.Root("ScopeSpan").
					Child("SchemaURL: " + ss.SchemaUrl).
					Child(scopeToTree(ss.Scope)).
					Child(t3))
			}
			rattrs, _ := attrsToTree("Attributes", rs.Resource.Attributes)
			t = t.Child(tree.Root("ResourceSpan").
				Child("Schema URL: " + rs.SchemaUrl).
				Child(rattrs).
				Child(t2))
		}
		for l := range strings.SplitSeq(t.String(), "\n") {
			lines = append(lines, components.ViewRow{Str: l, Yank: treeTrim(l)})
		}
	case []*metrics.ResourceMetrics:
		t := tree.Root(fmt.Sprintf("ResourceMetrics (%d)", len(p)))
		for _, rm := range p {
			refs := tree.Root("EntityRefs")
			for _, r := range rm.Resource.EntityRefs {
				refs = refs.Child(tree.Root("EntityRef").
					Child("SchemaUrl: " + r.SchemaUrl).
					Child("Type: " + r.Type).
					Child("IdKeys: " + strings.Join(r.IdKeys, ";")).
					Child("DescriptionKeys: " + strings.Join(r.DescriptionKeys, ";")))
			}
			t2 := tree.Root(fmt.Sprintf("ScopeMetrics (%d)", len(rm.ScopeMetrics)))
			for _, sm := range rm.ScopeMetrics {
				t3 := tree.Root(fmt.Sprintf("Metrics (%d)", len(sm.Metrics)))
				for _, m := range sm.Metrics {
					var t4 *tree.Tree
					switch mm := m.Data.(type) {
					case *metrics.Metric_Gauge:
						t4 = tree.Root("Gauge")
						for _, dp := range mm.Gauge.DataPoints {
							t4 = t4.Child(numberDataPointToTree(dp))
						}
					case *metrics.Metric_Sum:
						t4 = tree.Root("Sum").
							Child(fmt.Sprintf("AggregationTemporality: %s (raw=%d)", mm.Sum.AggregationTemporality.String(), mm.Sum.AggregationTemporality)).
							Child(fmt.Sprintf("IsMonotonic: %t", mm.Sum.IsMonotonic))
						for _, dp := range mm.Sum.DataPoints {
							t4 = t4.Child(numberDataPointToTree(dp))
						}
					case *metrics.Metric_Summary:
						t4 = tree.Root("Summary")
						for _, dp := range mm.Summary.DataPoints {
							quantileValues := tree.Root(fmt.Sprintf("QuantileValues (%d)", len(dp.QuantileValues)))
							for _, qv := range dp.QuantileValues {
								quantileValues = quantileValues.Child(fmt.Sprintf("Quantile: %f: Value: %f", qv.Quantile, qv.Value))
							}
							attrs, _ := attrsToTree("Attributes", dp.Attributes)
							t4 = t4.Child(tree.Root("SummaryDataPoint").
								Child(attrs).
								Child(fmt.Sprintf("StartTimeUnixNano: %s (raw=%d)", nanoToString(dp.StartTimeUnixNano), dp.StartTimeUnixNano)).
								Child(fmt.Sprintf("TimeUnixNano: %s (raw=%d)", nanoToString(dp.TimeUnixNano), dp.TimeUnixNano)).
								Child(fmt.Sprintf("Count: %d", dp.Count)).
								Child(fmt.Sprintf("Sum: %f", dp.Sum)).
								Child(quantileValues).
								Child(fmt.Sprintf("Flags: %d", dp.Flags)))
						}
					case *metrics.Metric_Histogram:
						t4 = tree.Root("Histogram").
							Child(fmt.Sprintf("AggregationTemporality: %s (raw=%d)", mm.Histogram.AggregationTemporality.String(), mm.Histogram.AggregationTemporality))
						for _, dp := range mm.Histogram.DataPoints {
							attrs, _ := attrsToTree("Attributes", dp.Attributes)
							t4 = t4.Child(tree.Root("HistogramDataPoint").
								Child(attrs).
								Child(fmt.Sprintf("StartTimeUnixNano: %s (raw=%d)", nanoToString(dp.StartTimeUnixNano), dp.StartTimeUnixNano)).
								Child(fmt.Sprintf("TimeUnixNano: %s (raw=%d)", nanoToString(dp.TimeUnixNano), dp.TimeUnixNano)).
								Child(fmt.Sprintf("Count: %d", dp.Count)).
								Child("Sum: " + ptrToString(dp.Sum)).
								Child(fmt.Sprintf("BucketCounts: %v", dp.BucketCounts)).
								Child(fmt.Sprintf("ExplicitBounds: %v", dp.ExplicitBounds)).
								Child(exemplarsToTree(dp.Exemplars)).
								Child("Min: " + ptrToString(dp.Min)).
								Child("Max: " + ptrToString(dp.Max)).
								Child(fmt.Sprintf("Flags: %d", dp.Flags)))
						}
					case *metrics.Metric_ExponentialHistogram:
						t4 = tree.Root("ExponentialHistogram").
							Child(fmt.Sprintf("AggregationTemporality: %s (raw=%d)", mm.ExponentialHistogram.AggregationTemporality.String(), mm.ExponentialHistogram.AggregationTemporality))
						for _, dp := range mm.ExponentialHistogram.DataPoints {
							attrs, _ := attrsToTree("Attributes", dp.Attributes)
							t4 = t4.Child(tree.Root("HistogramDataPoint").
								Child(attrs).
								Child(fmt.Sprintf("StartTimeUnixNano: %s (raw=%d)", nanoToString(dp.StartTimeUnixNano), dp.StartTimeUnixNano)).
								Child(fmt.Sprintf("TimeUnixNano: %s (raw=%d)", nanoToString(dp.TimeUnixNano), dp.TimeUnixNano)).
								Child(fmt.Sprintf("Count: %d", dp.Count)).
								Child("Sum: " + ptrToString(dp.Sum)).
								Child(fmt.Sprintf("Scale: %d", dp.Scale)).
								Child(fmt.Sprintf("ZeroCount: %d", dp.ZeroCount)).
								Child(exponentialBucketsToTree("Positive", dp.Positive)).
								Child(exponentialBucketsToTree("Negative", dp.Negative)).
								Child(exemplarsToTree(dp.Exemplars)).
								Child("Min: " + ptrToString(dp.Min)).
								Child("Max: " + ptrToString(dp.Max)).
								Child(fmt.Sprintf("ZeroThreshold: %f", dp.ZeroThreshold)).
								Child(fmt.Sprintf("Flags: %d", dp.Flags)))
						}

					}
					mattrs, _ := attrsToTree("Metadata", m.Metadata)
					t3 = t3.Child(tree.Root("Metric").
						Child("Name: " + m.Name).
						Child("Description: " + m.Description).
						Child("Unit: " + m.Unit).
						Child(mattrs).
						Child(t4))
				}
				t2 = t2.Child(tree.Root("ScopeMetric").
					Child("SchemaURL: " + sm.SchemaUrl).
					Child(scopeToTree(sm.Scope)).
					Child(t3))
			}
			rattrs, _ := attrsToTree("Attributes", rm.Resource.Attributes)
			t = t.Child(tree.Root("ResourceMetric").
				Child("Schema URL: " + rm.SchemaUrl).
				Child(rattrs).
				Child(refs).
				Child(t2))
		}
		for l := range strings.SplitSeq(t.String(), "\n") {
			lines = append(lines, components.ViewRow{Str: l, Yank: treeTrim(l)})
		}
	}
	m.view.Bot().SetContent(lines)
}
