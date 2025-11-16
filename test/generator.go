package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	shutdown, err := setupOTelSDK()
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			panic(err)
		}
	}()

	tracer := otel.GetTracerProvider().Tracer("generator")
	ticker1 := time.NewTicker(1 * time.Second)
	ticker2 := time.NewTicker(3 * time.Second)

	println("starting")

	for {
		select {
		case <-ticker1.C:
			ctx, span := tracer.Start(context.Background(), "i-am-root")
			span.SetStatus(codes.Ok, "i-am-ok")
			span.SetAttributes(attribute.String("i-am-attr", "i-am-val"))
			time.Sleep(3 * time.Nanosecond)
			span.AddEvent("i-am-event")
			time.Sleep(3 * time.Nanosecond)
			_, child := tracer.Start(ctx, "i-am-child")
			time.Sleep(3 * time.Nanosecond)
			child.End()
			span.End()
		case <-ticker2.C:
			ctx, span := tracer.Start(context.Background(), "i-am-root")
			span.SetStatus(codes.Error, "i-am-error")
			slog.ErrorContext(ctx, "i-am-error", "i-am-attr", "i-am-val")
			time.Sleep(3 * time.Nanosecond)
			_, child := tracer.Start(ctx, "i-am-child")
			time.Sleep(3 * time.Nanosecond)
			child.End()
			span.End()
		}
	}
}

func setupOTelSDK() (func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error
	var err error

	shutdown := func(ctx context.Context) (err error) {
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		return err
	}

	// traces
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	traceExporter, err := otlptracehttp.New(context.Background())
	if err != nil {
		return shutdown, err
	}
	tracerProvider := trace.NewTracerProvider(trace.WithBatcher(traceExporter))
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// metrics
	metricExporter, err := otlpmetrichttp.New(context.Background())
	if err != nil {
		return shutdown, err
	}
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(
			metricExporter,
			metric.WithProducer(runtime.NewProducer()),
		)),
	)
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	if err := runtime.Start(); err != nil {
		panic(err)
	}

	// logs
	logExporter, err := otlploghttp.New(context.Background())
	if err != nil {
		return shutdown, err
	}
	loggerProvider := log.NewLoggerProvider(log.WithProcessor(log.NewBatchProcessor(logExporter)))
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)
	slog.SetDefault(otelslog.NewLogger("generator"))

	return shutdown, err
}
