package ui

import (
	"context"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"

	"pitr.ca/otelui/server"
)

type refreshMsg struct{ reset bool }

func Run(ctx context.Context, cancel context.CancelFunc) {
	prog := tea.NewProgram(
		newRootModel(),
		tea.WithAltScreen(),
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
