package main

import (
	"context"
	"errors"
	"fmt"
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
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
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

	id := 0

	for {
		select {
		case <-ticker1.C:
			ctx, span := tracer.Start(context.Background(), "i-am-root")
			span.SetStatus(codes.Ok, "i-am-ok")
			span.SetAttributes(attribute.String("i-am-attr", "i-am-val"))
			time.Sleep(2 * time.Nanosecond)
			span.AddEvent("i-am-event")
			time.Sleep(2 * time.Nanosecond)
			_, child1 := tracer.Start(ctx, "i-am-child")
			ctx, child2 := tracer.Start(ctx, "i-am-child")
			_, sub1 := tracer.Start(ctx, "i-am-seq")
			time.Sleep(1 * time.Nanosecond)
			sub1.End()
			_, sub2 := tracer.Start(ctx, "i-am-seq")
			time.Sleep(1 * time.Nanosecond)
			sub2.End()
			_, sub3 := tracer.Start(ctx, "i-am-seq")
			time.Sleep(1 * time.Nanosecond)
			sub3.End()
			child2.End()
			child1.End()
			span.End()
		case <-ticker2.C:
			ctx, span := tracer.Start(context.Background(), "i-am-root")
			span.SetStatus(codes.Error, "i-am-error")
			slog.ErrorContext(ctx, "i-am-error", "i-am-attr", "i-am-val")
			slog.WarnContext(ctx, fmt.Sprintf("oops %d", id))
			slog.DebugContext(ctx, fmt.Sprintf("ok %d", id), "i-am-int", 42)
			id++
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

	ctx := context.Background()
	res, _ := resource.Merge(
		resource.Default(),
		resource.NewSchemaless(semconv.ServiceName("generator-app")),
	)

	// traces
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	traceExporter, err := otlptracehttp.New(ctx, otlptracehttp.WithInsecure())
	if err != nil {
		return shutdown, err
	}
	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// metrics
	metricExporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithInsecure())
	if err != nil {
		return shutdown, err
	}
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(
			metricExporter,
			metric.WithProducer(runtime.NewProducer()),
			metric.WithInterval(5*time.Second),
		)),
		metric.WithResource(res),
	)
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	if err := runtime.Start(); err != nil {
		panic(err)
	}

	// logs
	logExporter, err := otlploghttp.New(ctx, otlploghttp.WithInsecure())
	if err != nil {
		return shutdown, err
	}
	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
		log.WithResource(res),
	)
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)
	slog.SetDefault(otelslog.NewLogger("generator"))

	return shutdown, err
}
