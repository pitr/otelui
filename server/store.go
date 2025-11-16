package server

import (
	"math"
	"sync"
	"time"

	"github.com/zhangyunhao116/skipmap"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type Log struct {
	Received     time.Time
	Log          *plog.LogRecord
	ResourceLogs *plog.ResourceLogs
	ScopeLogs    *plog.ScopeLogs
}

var storage struct {
	sync.RWMutex

	logsReceived    int
	tracesReceived  int
	metricsReceived int

	logs *skipmap.Uint64Map[*Log]

	payloads []any
}

var Send func(msg any)

func init() {
	storage.logs = skipmap.NewUint64[*Log]()
}

func consumeLogs(p plog.Logs) error {
	storage.Lock()
	storage.logsReceived += p.LogRecordCount()
	storage.payloads = append(storage.payloads, p)
	Send(ConsumeEvent{
		Payloads: len(storage.payloads),
		Logs:     storage.logsReceived,
		Traces:   storage.tracesReceived,
		Metrics:  storage.metricsReceived,
	})
	storage.Unlock()

	now := time.Now().UTC()

	for _, rl := range p.ResourceLogs().All() {
		for _, sl := range rl.ScopeLogs().All() {
			for _, l := range sl.LogRecords().All() {
				storage.logs.Store(uint64(math.MaxUint64-l.Timestamp()), &Log{
					Log:          &l,
					ResourceLogs: &rl,
					ScopeLogs:    &sl,
					Received:     now,
				})
			}
		}
	}
	return nil
}

func consumeTraces(p ptrace.Traces) error {
	storage.Lock()
	storage.tracesReceived += p.SpanCount()
	storage.payloads = append(storage.payloads, p)
	Send(ConsumeEvent{
		Payloads: len(storage.payloads),
		Logs:     storage.logsReceived,
		Traces:   storage.tracesReceived,
		Metrics:  storage.metricsReceived,
	})
	storage.Unlock()
	return nil
}

func consumeMetrics(p pmetric.Metrics) error {
	storage.Lock()
	storage.metricsReceived += p.MetricCount()
	storage.payloads = append(storage.payloads, p)
	Send(ConsumeEvent{
		Payloads: len(storage.payloads),
		Logs:     storage.logsReceived,
		Traces:   storage.tracesReceived,
		Metrics:  storage.metricsReceived,
	})
	storage.Unlock()
	return nil
}

type ConsumeEvent struct {
	Payloads int
	Logs     int
	Traces   int
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
