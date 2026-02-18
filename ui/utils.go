package ui

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
	metrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	resource "go.opentelemetry.io/proto/otlp/resource/v1"

	"github.com/pitr/otelui/utils"
)

func AnyToType(v *v1.AnyValue) string {
	if v == nil {
		return "(null)"
	}
	switch v := v.Value.(type) {
	case *v1.AnyValue_StringValue:
		return "string"
	case *v1.AnyValue_BoolValue:
		return "bool"
	case *v1.AnyValue_IntValue:
		return "int"
	case *v1.AnyValue_DoubleValue:
		return "double"
	case *v1.AnyValue_ArrayValue:
		return "array"
	case *v1.AnyValue_KvlistValue:
		return "kvlist"
	case *v1.AnyValue_BytesValue:
		return "bytes"
	default:
		return fmt.Sprintf("%#v", v)
	}
}

var tzUTC bool

func nanoToString(nsec uint64) string {
	t := time.Unix(0, int64(nsec))
	if tzUTC {
		t = t.UTC()
	}
	return t.Format("2006-01-02T15:04:05.000000000Z07:00")
}

// treeTrim removes leading tree structure produced by lipgloss/tree, used for yanking
func treeTrim(s string) string {
	return strings.TrimLeft(s, " └├─│")
}

func attrsToTree(name string, kvs []*v1.KeyValue) (t *tree.Tree, set bool) {
	attrs := tree.Root(fmt.Sprintf("%s (%d):", name, len(kvs)))
	for _, a := range kvs {
		attrs = attrs.Child(fmt.Sprintf("%s: %s (%s)", a.Key, utils.AnyToString(a.Value), AnyToType(a.Value)))
	}
	return attrs, len(kvs) > 0
}

func scopeToTree(scope *v1.InstrumentationScope) *tree.Tree {
	sattrs, _ := attrsToTree("Attributes", scope.Attributes)
	return tree.Root("Scope").
		Child("Scope.Name: " + scope.Name).
		Child("Scope.Version: " + scope.Version).
		Child(sattrs).
		Child("DroppedAttributesCount: " + fmt.Sprint(scope.DroppedAttributesCount))
}

func exemplarsToTree(es []*metrics.Exemplar) *tree.Tree {
	t := tree.Root("Exemplars")
	for _, e := range es {
		value := "unknown"
		switch v := e.Value.(type) {
		case *metrics.Exemplar_AsInt:
			value = fmt.Sprintf("%d (int)", v.AsInt)
		case *metrics.Exemplar_AsDouble:
			value = fmt.Sprintf("%f (double)", v.AsDouble)
		}
		attrs, _ := attrsToTree("FilteredAttributes", e.FilteredAttributes)
		t.Child(tree.Root("Exemplar").
			Child(fmt.Sprintf("TimeUnixNano: %s (raw=%d)", nanoToString(e.TimeUnixNano), e.TimeUnixNano)).
			Child("Value: " + value).
			Child(attrs).
			Child("TraceId: " + hex.EncodeToString(e.TraceId)).
			Child("SpanId: " + hex.EncodeToString(e.SpanId)))
	}
	return t
}

func numberDataPointToTree(dp *metrics.NumberDataPoint) *tree.Tree {
	value := "unknown"
	switch v := dp.Value.(type) {
	case *metrics.NumberDataPoint_AsInt:
		value = fmt.Sprintf("%d (int)", v.AsInt)
	case *metrics.NumberDataPoint_AsDouble:
		value = fmt.Sprintf("%f (double)", v.AsDouble)
	}
	attrs, _ := attrsToTree("Attributes", dp.Attributes)
	return tree.Root("NumberDataPoint").
		Child(attrs).
		Child(fmt.Sprintf("StartTimeUnixNano: %s (raw=%d)", nanoToString(dp.StartTimeUnixNano), dp.StartTimeUnixNano)).
		Child(fmt.Sprintf("TimeUnixNano: %s (raw=%d)", nanoToString(dp.TimeUnixNano), dp.TimeUnixNano)).
		Child("Value: " + value).
		Child(exemplarsToTree(dp.Exemplars)).
		Child(fmt.Sprintf("Flags: %d", dp.Flags))
}

func exponentialBucketsToTree(name string, eb *metrics.ExponentialHistogramDataPoint_Buckets) *tree.Tree {
	return tree.Root(fmt.Sprintf("%s:", name)).
		Child(fmt.Sprintf("Offset: %d", eb.Offset)).
		Child(fmt.Sprintf("BucketCounts: %v", eb.BucketCounts))
}

func ptrToString(x *float64) string {
	if x == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%f", *x)
}

func resourceToServiceName(r *resource.Resource) string {
	if r == nil {
		return "-"
	}
	for _, attr := range r.Attributes {
		if attr.Key == string(semconv.ServiceNameKey) {
			return utils.AnyToString(attr.Value)
		}
	}
	return "-"
}

// renderForeground resets only foreground instead of reset all (so row select works correctly)
func renderForeground(c lipgloss.TerminalColor, str string) string {
	return strings.ReplaceAll(lipgloss.NewStyle().Foreground(c).Render(str), "\x1b[0m", "\x1b[39m")
}
