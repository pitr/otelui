package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"

	"pitr.ca/otelui/server"
	"pitr.ca/otelui/ui"
)

func main() {
	logs := io.Discard

	if os.Getenv("DEBUG") != "" {
		f, err := os.OpenFile("debug.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
		if err != nil {
			fmt.Printf("error opening file for logging: %s", err)
			return
		}
		logs = f
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	log.Default()

	ctx, cancel := context.WithCancel(context.Background())
	server.Start(ctx, cancel)
	go ui.Run(ctx, cancel)

	<-ctx.Done()
}
