package components

import "github.com/charmbracelet/lipgloss"

var (
	FadedColor     = lipgloss.AdaptiveColor{Light: "#E6E6E6", Dark: "#383838"}
	HighlightColor = lipgloss.AdaptiveColor{Light: "#CBB1FD", Dark: "#7D56F4"}
	ErrorColor     = lipgloss.Color("#FF6B6B")
	WarnColor      = lipgloss.Color("#FFD93D")
	InfoColor      = lipgloss.Color("#0F93FC")
	DebugColor     = lipgloss.Color("#999999")
)
