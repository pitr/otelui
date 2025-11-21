package ui

import (
	"context"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pitr/otelui/server"
)

func Run(ctx context.Context, cancel context.CancelFunc) {
	prog := tea.NewProgram(
		newRootModel(),
		tea.WithAltScreen(), tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	)
	server.Send = func(msg any) {
		prog.Send(msg)
	}
	if _, err := prog.Run(); err != nil {
		slog.ErrorContext(ctx, "error starting UI", "err", err)
	}
	cancel()
}
