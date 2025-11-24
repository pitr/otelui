package server

import (
	"sync"
	"time"

	logs "go.opentelemetry.io/proto/otlp/logs/v1"
	metrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	traces "go.opentelemetry.io/proto/otlp/trace/v1"
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

var Storage struct {
	sync.RWMutex

	spansReceived   int
	metricsReceived int

	Payloads []*Payload
	Logs     []*Log
}

var Send func(msg any)

func init() {
	Storage.Logs = []*Log{}
	Storage.Payloads = []*Payload{}

	go func() {
		for range time.Tick(time.Second) {
			Storage.RLock()
			e := ConsumeEvent{
				Payloads: len(Storage.Payloads),
				Logs:     len(Storage.Logs),
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

	Storage.Payloads = append(Storage.Payloads, &Payload{Received: now, Num: len(newLogs), Payload: p})
	Storage.Logs = append(Storage.Logs, newLogs...)

}

func consumeTraces(p []*traces.ResourceSpans) {
	if p == nil {
		return
	}

	now := time.Now().UTC()
	spansReceived := 0

	for _, rs := range p {
		for _, ss := range rs.ScopeSpans {
			spansReceived += len(ss.Spans)
		}
	}

	Storage.Lock()
	defer Storage.Unlock()

	Storage.Payloads = append(Storage.Payloads, &Payload{Received: now, Num: spansReceived, Payload: p})
	Storage.spansReceived += spansReceived
}

func consumeMetrics(p []*metrics.ResourceMetrics) {
	if p == nil {
		return
	}

	now := time.Now().UTC()
	metricsReceived := 0

	for _, rm := range p {
		for _, sm := range rm.ScopeMetrics {
			metricsReceived += len(sm.Metrics)
		}
	}

	Storage.Lock()
	defer Storage.Unlock()

	Storage.Payloads = append(Storage.Payloads, &Payload{Received: now, Num: metricsReceived, Payload: p})
	Storage.metricsReceived += metricsReceived
}

type ConsumeEvent struct {
	Payloads int
	Logs     int
	Spans    int
	Metrics  int
}
