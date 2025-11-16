package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/pitr/otelui/server"
	"github.com/pitr/otelui/ui"
)

func main() {
	f, err := os.OpenFile("debug.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		fmt.Printf("error opening file for logging: %s", err)
		return
	}
	defer f.Close()

	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	log.Default()

	ctx, cancel := context.WithCancel(context.Background())
	server.Start(ctx, cancel)
	go ui.Run(ctx, cancel)

	<-ctx.Done()
}
