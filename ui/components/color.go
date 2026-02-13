package components

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	SelectionColor = lipgloss.AdaptiveColor{Light: "7", Dark: "236"}
	AccentColor    = lipgloss.AdaptiveColor{Light: "6", Dark: "14"}
	ErrorColor     = lipgloss.Color("1")
	WarnColor      = lipgloss.Color("3")
	InfoColor      = lipgloss.Color("4")
	DebugColor     = lipgloss.Color("8")
)
