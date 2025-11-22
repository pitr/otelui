package server

import (
	"sync"
	"time"

	logs "go.opentelemetry.io/proto/otlp/logs/v1"
	metrics "go.opentelemetry.io/proto/otlp/metrics/v1"
	traces "go.opentelemetry.io/proto/otlp/trace/v1"
)

type Log struct {
	Received     time.Time
	Log          *logs.LogRecord
	ResourceLogs *logs.ResourceLogs
	ScopeLogs    *logs.ScopeLogs
}

var storage struct {
	sync.RWMutex

	payloadsReceived int
	logsReceived     int
	spansReceived    int
	metricsReceived  int

	logs []*Log
}

var Send func(msg any)

func init() {
	storage.logs = []*Log{}

	go func() {
		for range time.Tick(time.Second) {
			storage.RLock()
			e := ConsumeEvent{
				Payloads: storage.payloadsReceived,
				Logs:     storage.logsReceived,
				Spans:    storage.spansReceived,
				Metrics:  storage.metricsReceived,
			}
			storage.RUnlock()
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

	storage.Lock()

	storage.payloadsReceived++
	storage.logsReceived += len(newLogs)
	storage.logs = append(storage.logs, newLogs...)

	storage.Unlock()

	Send(NewLogsEvent{NewLogs: newLogs})
}

func consumeTraces(p []*traces.ResourceSpans) {
	if p == nil {
		return
	}

	spansReceived := 0

	for _, rs := range p {
		for _, ss := range rs.ScopeSpans {
			spansReceived += len(ss.Spans)
		}
	}

	storage.Lock()
	defer storage.Unlock()
	storage.spansReceived += spansReceived
	storage.payloadsReceived++
}

func consumeMetrics(p []*metrics.ResourceMetrics) {
	if p == nil {
		return
	}

	metricsReceived := 0

	for _, rm := range p {
		for _, sm := range rm.ScopeMetrics {
			metricsReceived += len(sm.Metrics)
		}
	}

	storage.Lock()
	defer storage.Unlock()
	storage.metricsReceived += metricsReceived
	storage.payloadsReceived++
}

type ServerEvent interface{ secret() }

type ConsumeEvent struct {
	Payloads int
	Logs     int
	Spans    int
	Metrics  int
}

type QueriedLogsEvent struct{ Logs []*Log }
type NewLogsEvent struct{ NewLogs []*Log }

func (ConsumeEvent) secret()     {}
func (QueriedLogsEvent) secret() {}
func (NewLogsEvent) secret()     {}

var _ ServerEvent = ConsumeEvent{}
var _ ServerEvent = QueriedLogsEvent{}
var _ ServerEvent = NewLogsEvent{}

func QueryLogs() QueriedLogsEvent {
	storage.RLock()
	defer storage.RUnlock()
	return QueriedLogsEvent{Logs: storage.logs}
}
