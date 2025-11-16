package server

import (
	"context"
	"log/slog"
	"strings"
	"text/template"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const tmpl = `yaml:
receivers:
  otlp:
    protocols:
      grpc:
      http:
exporters:
  mem: {}
service:
  telemetry:
    metrics:
      level: none
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [mem]
    metrics:
      receivers: [otlp]
      exporters: [mem]
    logs:
      receivers: [otlp]
      exporters: [mem]`

func Start(ctx context.Context, cancel context.CancelFunc) {
	var configs strings.Builder
	err := template.Must(template.New("config").Parse(tmpl)).Execute(
		&configs,
		map[string]string{},
	)
	if err != nil {
		slog.ErrorContext(ctx, "could not create internal collector config", "err", err)
		cancel()
		return
	}

	settings := otelcol.CollectorSettings{
		Factories: factories,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs:              []string{configs.String()},
				ProviderFactories: []confmap.ProviderFactory{yamlprovider.NewFactory()},
			},
		},
		LoggingOptions: []zap.Option{
			zap.WrapCore(func(zapcore.Core) zapcore.Core {
				return &slogCore{}
			}),
		},
	}

	col, err := otelcol.NewCollector(settings)
	if err != nil {
		slog.ErrorContext(ctx, "could not initiate internal collector: %s", "err", err)
		cancel()
		return
	}
	go func() {
		err := col.Run(ctx)
		cancel()
		if err != nil {
			slog.ErrorContext(ctx, "could not run internal collector: %s", "err", err)
		}
	}()
}

func factories() (f otelcol.Factories, err error) {
	f.Receivers, err = otelcol.MakeFactoryMap(
		otlpreceiver.NewFactory(),
	)
	if err != nil {
		return
	}
	f.Exporters, err = otelcol.MakeFactoryMap(newMemExporter())
	if err != nil {
		return
	}
	return
}
