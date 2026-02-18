package server

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	v1 "go.opentelemetry.io/proto/otlp/common/v1"
	logs "go.opentelemetry.io/proto/otlp/logs/v1"
	metrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	resource "go.opentelemetry.io/proto/otlp/resource/v1"
	traces "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/pitr/otelui/utils"
)

type Payload struct {
	Received time.Time
	Num      int
	Payload  any
}

type Log struct {
	Received     time.Time
	Log          *logs.LogRecord
	ResourceLogs *logs.ResourceLogs
	ScopeLogs    *logs.ScopeLogs
}

type Datapoints struct {
	Times  []uint64
	Values []float64
}

type Span struct {
	Span     *traces.Span
	Resource *resource.Resource
	Scope    *v1.InstrumentationScope
}

type Trace struct {
	TraceID string
	Spans   []*Span
}

var Storage struct {
	sync.RWMutex

	spansReceived   int
	metricsReceived int

	payloads   []*Payload
	logs       []*Log
	metrics    map[string]*Datapoints
	traces     map[string]*Trace
	traceOrder []string
}

type ConsumeEvent struct {
	Payloads int
	Logs     int
	Spans    int
	Metrics  int
}

var Send func(msg any)

func Reset() {
	Storage.Lock()
	defer Storage.Unlock()
	Storage.logs = []*Log{}
	Storage.payloads = []*Payload{}
	Storage.metrics = map[string]*Datapoints{}
	Storage.traces = map[string]*Trace{}
	Storage.traceOrder = []string{}
	Storage.spansReceived = 0
	Storage.metricsReceived = 0
}

func setupStorage() {
	Storage.logs = []*Log{}
	Storage.payloads = []*Payload{}
	Storage.metrics = map[string]*Datapoints{}
	Storage.traces = map[string]*Trace{}
	Storage.traceOrder = []string{}

	go func() {
		for range time.Tick(time.Second) {
			Storage.RLock()
			e := ConsumeEvent{
				Payloads: len(Storage.payloads),
				Logs:     len(Storage.logs),
				Spans:    Storage.spansReceived,
				Metrics:  Storage.metricsReceived,
			}
			Storage.RUnlock()
			Send(e)
		}
	}()
}

func consumeLogs(p []*logs.ResourceLogs) {
	if p == nil {
		return
	}

	newLogs := []*Log{}
	now := time.Now().UTC()

	for _, rl := range p {
		for _, sl := range rl.ScopeLogs {
			for _, l := range sl.LogRecords {
				newLogs = append(newLogs, &Log{
					Log:          l,
					ResourceLogs: rl,
					ScopeLogs:    sl,
					Received:     now,
				})
			}
		}
	}

	Storage.Lock()
	defer Storage.Unlock()

	Storage.payloads = append(Storage.payloads, &Payload{Received: now, Num: len(newLogs), Payload: p})
	for _, log := range newLogs {
		i := sort.Search(len(Storage.logs), func(i int) bool { return Storage.logs[i].Log.TimeUnixNano > log.Log.TimeUnixNano })
		Storage.logs = append(Storage.logs, nil)
		copy(Storage.logs[i+1:], Storage.logs[i:])
		Storage.logs[i] = log
	}
}

func consumeTraces(p []*traces.ResourceSpans) {
	if p == nil {
		return
	}

	now := time.Now().UTC()
	spansReceived := 0
	byTrace := map[string][]*Span{}

	for _, rs := range p {
		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				spansReceived++
				tid := hex.EncodeToString(s.TraceId)
				byTrace[tid] = append(byTrace[tid], &Span{Span: s, Resource: rs.Resource, Scope: ss.Scope})
			}
		}
	}

	Storage.Lock()
	defer Storage.Unlock()

	Storage.payloads = append(Storage.payloads, &Payload{Received: now, Num: spansReceived, Payload: p})
	for tid, spans := range byTrace {
		if t, ok := Storage.traces[tid]; ok {
			t.Spans = append(t.Spans, spans...)
		} else {
			Storage.traces[tid] = &Trace{TraceID: tid, Spans: spans}
			Storage.traceOrder = append(Storage.traceOrder, tid)
		}
	}
	Storage.spansReceived += spansReceived
}

func consumeMetrics(p []*metrics.ResourceMetrics) {
	if p == nil {
		return
	}

	now := time.Now().UTC()
	metricsReceived := 0

	Storage.Lock()
	defer Storage.Unlock()

	for _, rm := range p {
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				switch d := m.Data.(type) {
				case *metrics.Metric_Gauge:
					for _, dp := range d.Gauge.DataPoints {
						metricsReceived++
						attrs := serializeAttributes(m.Name, dp.Attributes, sm.Scope.Attributes, rm.Resource.Attributes)
						if Storage.metrics[attrs] == nil {
							Storage.metrics[attrs] = &Datapoints{}
						}
						Storage.metrics[attrs].Times = append(Storage.metrics[attrs].Times, dp.TimeUnixNano)

						switch v := dp.Value.(type) {
						case *metrics.NumberDataPoint_AsInt:
							Storage.metrics[attrs].Values = append(Storage.metrics[attrs].Values, float64(v.AsInt))
						case *metrics.NumberDataPoint_AsDouble:
							Storage.metrics[attrs].Values = append(Storage.metrics[attrs].Values, v.AsDouble)
						}
					}
				case *metrics.Metric_Sum:
					for _, dp := range d.Sum.DataPoints {
						metricsReceived++
						attrs := serializeAttributes(m.Name, dp.Attributes, sm.Scope.Attributes, rm.Resource.Attributes)
						if Storage.metrics[attrs] == nil {
							Storage.metrics[attrs] = &Datapoints{}
						}
						Storage.metrics[attrs].Times = append(Storage.metrics[attrs].Times, dp.TimeUnixNano)

						switch v := dp.Value.(type) {
						case *metrics.NumberDataPoint_AsInt:
							Storage.metrics[attrs].Values = append(Storage.metrics[attrs].Values, float64(v.AsInt))
						case *metrics.NumberDataPoint_AsDouble:
							Storage.metrics[attrs].Values = append(Storage.metrics[attrs].Values, v.AsDouble)
						}
					}
				case *metrics.Metric_Summary:
					for _, dp := range d.Summary.DataPoints {
						metricsReceived++
						attrs := serializeAttributes(m.Name, dp.Attributes, sm.Scope.Attributes, rm.Resource.Attributes)
						if Storage.metrics[attrs] == nil {
							Storage.metrics[attrs] = &Datapoints{}
						}
						Storage.metrics[attrs].Times = append(Storage.metrics[attrs].Times, dp.TimeUnixNano)
						Storage.metrics[attrs].Values = append(Storage.metrics[attrs].Values, dp.Sum)
					}
				case *metrics.Metric_Histogram:
				case *metrics.Metric_ExponentialHistogram:
				}
			}
		}
	}

	Storage.payloads = append(Storage.payloads, &Payload{Received: now, Num: metricsReceived, Payload: p})
	Storage.metricsReceived += metricsReceived
}

func serializeAttributes(name string, attrs ...[]*v1.KeyValue) string {
	hashes := []string{}
	for _, attr := range attrs {
		for _, a := range attr {
			hashes = append(hashes, fmt.Sprintf("%s=%q", a.Key, utils.AnyToString(a.Value)))
		}
	}
	sort.Strings(hashes)
	return fmt.Sprintf("%s{%s}", name, strings.Join(hashes, ","))
}
