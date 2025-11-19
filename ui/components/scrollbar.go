package components

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type ScrollbarPosition int

const (
	ScrollbarHorizontal ScrollbarPosition = iota
	ScrollbarVertical

	ScrollbarSize = 1

	scrollbarVThumb = "▐"
	scrollbarHThumb = "━"
)

func Scrollbar(style lipgloss.Style, pos ScrollbarPosition, size, total, visible, offset int) string {
	b, _, _, _, _ := style.GetBorder()
	c := style.GetBorderRightForeground()
	if total <= visible {
		switch pos {
		case ScrollbarVertical:
			return lipgloss.NewStyle().Foreground(c).Render(b.TopRight + "\n" + strings.Repeat(b.Right+"\n", size) + b.BottomRight)
		case ScrollbarHorizontal:
			return lipgloss.NewStyle().Foreground(c).Render(b.BottomLeft + strings.Repeat(b.Bottom, size))
		}
	}
	ratio := float64(size) / float64(total)
	thumbSize := max(1, int(math.Round(float64(visible)*ratio)))
	thumbOffset := max(0, min(size-thumbSize, int(math.Round(float64(offset)*ratio))))

	if pos == ScrollbarVertical {
		return lipgloss.NewStyle().Foreground(c).Render(b.TopRight + "\n" +
			strings.Repeat(b.Right+"\n", thumbOffset) +
			strings.Repeat(scrollbarVThumb+"\n", thumbSize) +
			strings.Repeat(b.Right+"\n", max(0, size-thumbOffset-thumbSize)) +
			b.BottomRight)
	} else {
		return lipgloss.NewStyle().Foreground(c).Render(b.BottomLeft +
			strings.Repeat(b.Bottom, thumbOffset) +
			strings.Repeat(scrollbarHThumb, thumbSize) +
			strings.Repeat(b.Bottom, max(0, size-thumbOffset-thumbSize)))
	}
}
