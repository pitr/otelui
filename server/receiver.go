package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"

	logs "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	metrics "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	traces "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// server is an OTLP logs receiver
type logsReceiver struct {
	logs.UnimplementedLogsServiceServer
}
type tracesReceiver struct {
	traces.UnimplementedTraceServiceServer
}
type metricsReceiver struct {
	metrics.UnimplementedMetricsServiceServer
}

// NewReceiver creates a new OTLP receiver
func Start(ctx context.Context, cancel context.CancelFunc) {
	var err error

	grpcListener, err := net.Listen("tcp", ":4317")
	if err != nil {
		slog.ErrorContext(ctx, "failed to listen on gRPC port 4317", "err", err)
		cancel()
		return
	}

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(4*1024*1024), // 4MB max receive message size
		grpc.MaxSendMsgSize(4*1024*1024), // 4MB max send message size
	)

	lr := &logsReceiver{}
	tr := &tracesReceiver{}
	mr := &metricsReceiver{}

	logs.RegisterLogsServiceServer(grpcServer, lr)
	traces.RegisterTraceServiceServer(grpcServer, tr)
	metrics.RegisterMetricsServiceServer(grpcServer, mr)

	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil && err != grpc.ErrServerStopped {
			slog.ErrorContext(ctx, "OTLP gRPC receiver serve error", "err", err)
			cancel()
		}
	}()

	httpListener, err := net.Listen("tcp", ":4318")
	if err != nil {
		slog.ErrorContext(ctx, "failed to listen on HTTP port 4318", "err", err)
		cancel()
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/logs", lr.handle)
	mux.HandleFunc("/v1/traces", tr.handle)
	mux.HandleFunc("/v1/metrics", mr.handle)

	httpServer := &http.Server{Handler: mux}

	go func() {
		if err := httpServer.Serve(httpListener); err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(ctx, "OTLP HTTP receiver serve error", "err", err)
			cancel()
		}
	}()

	go func() {
		<-ctx.Done()

		if grpcServer != nil {
			grpcServer.GracefulStop()
		}
		if httpServer != nil {
			httpServer.Shutdown(context.Background())
		}
	}()
}

// Export implements the OTLP logs service Export method
func (r *logsReceiver) Export(ctx context.Context, req *logs.ExportLogsServiceRequest) (*logs.ExportLogsServiceResponse, error) {
	consumeLogs(req.ResourceLogs)
	return &logs.ExportLogsServiceResponse{}, nil
}

func (r *tracesReceiver) ExportTraces(ctx context.Context, req *traces.ExportTraceServiceRequest) (*traces.ExportTraceServiceResponse, error) {
	consumeTraces(req.ResourceSpans)
	return &traces.ExportTraceServiceResponse{}, nil
}

func (r *metricsReceiver) ExportMetrics(ctx context.Context, req *metrics.ExportMetricsServiceRequest) (*metrics.ExportMetricsServiceResponse, error) {
	consumeMetrics(req.ResourceMetrics)
	return &metrics.ExportMetricsServiceResponse{}, nil
}

func handle(w http.ResponseWriter, req *http.Request, payload, res proto.Message) (success bool) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return false
	}
	defer req.Body.Close()

	switch req.Header.Get("Content-Type") {
	case "application/x-protobuf", "application/protobuf":
		if err := proto.Unmarshal(body, payload); err != nil {
			http.Error(w, "Failed to unmarshal protobuf", http.StatusBadRequest)
			return false
		}
	case "application/json":
		if err := protojson.Unmarshal(body, payload); err != nil {
			http.Error(w, "Failed to unmarshal JSON", http.StatusBadRequest)
			return false
		}
	default:
		if err := proto.Unmarshal(body, payload); err != nil {
			if err := protojson.Unmarshal(body, payload); err != nil {
				http.Error(w, "Failed to unmarshal request", http.StatusBadRequest)
				return false
			}
		}
	}

	switch req.Header.Get("Accept") {
	case "application/json":
		w.Header().Set("Content-Type", "application/json")
		jsonBytes, _ := protojson.Marshal(res)
		w.Write(jsonBytes)
	default:
		w.Header().Set("Content-Type", "application/x-protobuf")
		protoBytes, _ := proto.Marshal(res)
		w.Write(protoBytes)
	}

	return true
}

func (r *logsReceiver) handle(w http.ResponseWriter, req *http.Request) {
	var payload logs.ExportLogsServiceRequest
	if handle(w, req, &payload, &logs.ExportLogsServiceResponse{}) {
		consumeLogs(payload.ResourceLogs)
	}
}

func (r *tracesReceiver) handle(w http.ResponseWriter, req *http.Request) {
	var payload traces.ExportTraceServiceRequest
	if handle(w, req, &payload, &traces.ExportTraceServiceResponse{}) {
		consumeTraces(payload.ResourceSpans)
	}
}

func (r *metricsReceiver) handle(w http.ResponseWriter, req *http.Request) {
	var payload metrics.ExportMetricsServiceRequest
	if handle(w, req, &payload, &metrics.ExportMetricsServiceResponse{}) {
		consumeMetrics(payload.ResourceMetrics)
	}
}
