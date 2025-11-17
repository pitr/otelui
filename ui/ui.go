package ui

import (
	"context"
	"log/slog"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pitr/otelui/server"
)

var (
	fadedColor     = lipgloss.AdaptiveColor{Light: "#B2B2B2", Dark: "#4A4A4A"}
	highlightColor = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
)

type Helpful interface {
	Help() []key.Binding
}

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
