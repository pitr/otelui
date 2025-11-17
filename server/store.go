package server

import (
	"math"
	"sync"
	"time"

	"github.com/zhangyunhao116/skipmap"
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

	logs *skipmap.Uint64Map[*Log]
}

var Send func(msg any)

func init() {
	storage.logs = skipmap.NewUint64[*Log]()
}

func consumeLogs(p []*logs.ResourceLogs) {
	if p == nil {
		return
	}

	logsReceived := 0
	now := time.Now().UTC()

	for _, rl := range p {
		for _, sl := range rl.ScopeLogs {
			logsReceived += len(sl.LogRecords)

			for _, l := range sl.LogRecords {
				storage.logs.Store(uint64(math.MaxUint64-l.TimeUnixNano), &Log{
					Log:          l,
					ResourceLogs: rl,
					ScopeLogs:    sl,
					Received:     now,
				})
			}
		}
	}

	storage.Lock()
	defer storage.Unlock()
	storage.logsReceived += logsReceived
	storage.payloadsReceived++
	Send(ConsumeEvent{
		Payloads: storage.payloadsReceived,
		Logs:     storage.logsReceived,
		Spans:    storage.spansReceived,
		Metrics:  storage.metricsReceived,
	})
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
	Send(ConsumeEvent{
		Payloads: storage.payloadsReceived,
		Logs:     storage.logsReceived,
		Spans:    storage.spansReceived,
		Metrics:  storage.metricsReceived,
	})
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
	Send(ConsumeEvent{
		Payloads: storage.payloadsReceived,
		Logs:     storage.logsReceived,
		Spans:    storage.spansReceived,
		Metrics:  storage.metricsReceived,
	})
}

type ConsumeEvent struct {
	Payloads int
	Logs     int
	Spans    int
	Metrics  int
}

func QueryLogs(n int) []*Log {
	res := []*Log{}
	storage.logs.Range(func(key uint64, value *Log) bool {
		res = append(res, value)
		n--
		return n != 0
	})
	return res
}
