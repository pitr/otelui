package server

func GetPayloads() []*Payload {
	Storage.RLock()
	defer Storage.RUnlock()
	res := make([]*Payload, len(Storage.payloads))
	copy(res, Storage.payloads)
	return res
}

func GetLogs() []*Log {
	Storage.RLock()
	defer Storage.RUnlock()
	res := make([]*Log, len(Storage.logs))
	copy(res, Storage.logs)
	return res
}

func GetTraces() []*Trace {
	Storage.RLock()
	defer Storage.RUnlock()
	res := make([]*Trace, len(Storage.traceOrder))
	for i, id := range Storage.traceOrder {
		orig := Storage.traces[id]
		t := &Trace{TraceID: orig.TraceID, Spans: make([]*Span, len(orig.Spans))}
		copy(t.Spans, orig.Spans)
		res[i] = t
	}
	return res
}

func GetMetrics() []string {
	Storage.RLock()
	defer Storage.RUnlock()
	res := []string{}
	for k := range Storage.metrics {
		res = append(res, k)
	}
	return res
}

func GetDatapoints(name string) *Datapoints {
	Storage.RLock()
	defer Storage.RUnlock()

	m, ok := Storage.metrics[name]
	if !ok {
		return nil
	}

	res := &Datapoints{
		Times:  make([]uint64, len(m.Times)),
		Values: make([]float64, len(m.Values)),
	}
	copy(res.Times, m.Times)
	copy(res.Values, m.Values)
	return res
}
