# otelui

Terminal UI for OpenTelemetry data. Receives OTLP over gRPC (`:4317`) and HTTP (`:4318`), stores in memory, and renders logs, traces, and metrics in real-time.

![demo](demo.gif)

## Install

Download a binary from [GitHub Releases](https://github.com/pitr/otelui/releases), or:

```sh
go install pitr.ca/otelui@latest
```

## Usage

Point your OTEL SDK or collector at `localhost:4317` (gRPC) or `localhost:4318` (HTTP). The run:

```sh
otelui
```
