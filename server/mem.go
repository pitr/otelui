package server

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type memExporter struct{}

func newMemExporter() exporter.Factory {
	mem := &memExporter{}
	stability := component.StabilityLevelStable

	return exporter.NewFactory(
		component.MustNewType("mem"),
		func() component.Config { return nil },
		exporter.WithLogs(mem.CreateLogs, stability),
		exporter.WithMetrics(mem.CreateMetrics, stability),
		exporter.WithTraces(mem.CreateTraces, stability),
	)
}

func (m *memExporter) CreateTraces(_ context.Context, _ exporter.Settings, _ component.Config) (exporter.Traces, error) {
	return m, nil
}
func (m *memExporter) CreateMetrics(_ context.Context, _ exporter.Settings, _ component.Config) (exporter.Metrics, error) {
	return m, nil
}
func (m *memExporter) CreateLogs(_ context.Context, _ exporter.Settings, _ component.Config) (exporter.Logs, error) {
	return m, nil
}

func (*memExporter) Start(_ context.Context, _ component.Host) error { return nil }
func (*memExporter) Shutdown(_ context.Context) error                { return nil }
func (*memExporter) Capabilities() consumer.Capabilities             { return consumer.Capabilities{} }

func (*memExporter) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	return consumeTraces(td)
}
func (*memExporter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	return consumeMetrics(md)
}
func (*memExporter) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	return consumeLogs(ld)
}
