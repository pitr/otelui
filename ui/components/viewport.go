package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type Viewport struct {
	IsFocused bool

	w, h       int
	_border    lipgloss.Border
	_focused   lipgloss.Style
	_unfocused lipgloss.Style

	yOffset          int
	xOffset          int
	yPosition        int
	lines            []string
	longestLineWidth int
}

func NewViewport(focused bool) Viewport {
	return Viewport{
		IsFocused:  focused,
		_border:    lipgloss.RoundedBorder(),
		_focused:   lipgloss.NewStyle(),
		_unfocused: lipgloss.NewStyle().BorderForeground(FadedColor),
	}
}

func (v Viewport) Init() tea.Cmd {
	return nil
}

func (v Viewport) Update(msg tea.Msg) (Viewport, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.w = msg.Width - 2
		v.h = msg.Height - 2
	case tea.KeyMsg:
		switch msg.String() {
		case "pgdown", " ":
			v.ScrollDown(v.h)
		case "pgup":
			v.ScrollUp(v.h)
		case "down":
			v.ScrollDown(1)
		case "up":
			v.ScrollUp(1)
		case "left":
			v.ScrollLeft(1)
		case "right":
			v.ScrollRight(1)
		}

	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			break
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if msg.Shift {
				v.ScrollLeft(1)
			} else {
				v.ScrollUp(1)
			}

		case tea.MouseButtonWheelDown:
			if msg.Shift {
				v.ScrollRight(1)
			} else {
				v.ScrollDown(1)
			}
		case tea.MouseButtonWheelLeft:
			v.ScrollLeft(1)
		case tea.MouseButtonWheelRight:
			v.ScrollRight(1)
		}
	}

	return v, cmd
}

func (v Viewport) View() string {
	s := v._unfocused
	if v.IsFocused {
		s = v._focused
	}
	s = s.Border(v._border, true, false, false, true)
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.JoinVertical(
			lipgloss.Left,
			s.Render(lipgloss.NewStyle().
				Width(v.w).MaxWidth(v.w).
				Height(v.h).MaxHeight(v.h).
				Render(strings.Join(v.visibleLines(), "\n"))),
			Scrollbar(
				s,
				ScrollbarHorizontal,
				v.w,
				v.longestLineWidth,
				v.w,
				v.xOffset,
			),
		),
		Scrollbar(
			s,
			ScrollbarVertical,
			v.h,
			len(v.lines),
			len(v.visibleLines()),
			v.yOffset,
		))
}

func (v Viewport) AtTop() bool {
	return v.yOffset <= 0
}

func (v Viewport) AtBottom() bool {
	return v.yOffset >= v.maxYOffset()
}

func (v *Viewport) SetYOffset(n int) {
	v.yOffset = clamp(n, 0, v.maxYOffset())
}

func (v *Viewport) ScrollDown(n int) {
	if v.AtBottom() || n == 0 || len(v.lines) == 0 {
		return
	}

	v.SetYOffset(v.yOffset + n)
}

func (v *Viewport) ScrollUp(n int) {
	if v.AtTop() || n == 0 || len(v.lines) == 0 {
		return
	}

	v.SetYOffset(v.yOffset - n)
}

func (v *Viewport) SetXOffset(n int) {
	v.xOffset = clamp(n, 0, v.longestLineWidth-v.w)
}

func (v *Viewport) ScrollLeft(n int) {
	v.SetXOffset(v.xOffset - n)
}

func (v *Viewport) ScrollRight(n int) {
	v.SetXOffset(v.xOffset + n)
}

func (v Viewport) maxYOffset() int {
	return max(0, len(v.lines)-v.h)
}

func (v *Viewport) SetContent(s string) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	v.lines = strings.Split(s, "\n")
	v.longestLineWidth = v.findLongestLineWidth(v.lines)

	if v.yOffset > len(v.lines)-1 {
		v.SetYOffset(v.maxYOffset())
	}
}

func (v Viewport) visibleLines() (lines []string) {
	if len(v.lines) > 0 {
		top := max(0, v.yOffset)
		bottom := clamp(v.yOffset+v.h, top, len(v.lines))
		lines = v.lines[top:bottom]
	}

	if (v.xOffset == 0 && v.longestLineWidth <= v.w) || v.w == 0 {
		return lines
	}

	cutLines := make([]string, len(lines))
	for i := range lines {
		cutLines[i] = ansi.Cut(lines[i], v.xOffset, v.xOffset+v.w)
	}
	return cutLines
}

func (v *Viewport) findLongestLineWidth(lines []string) int {
	w := 0
	for _, l := range lines {
		if ww := ansi.StringWidth(l); ww > w {
			w = ww
		}
	}
	return w
}

func clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}
