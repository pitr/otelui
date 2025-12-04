package components

import (
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type ViewRow struct {
	Str  string
	Yank string
	Raw  any
}

type keysViewport struct {
	Yank   key.Binding
	Pgdown key.Binding
	Pgup   key.Binding
	Down   key.Binding
	Up     key.Binding
	Left   key.Binding
	Right  key.Binding
}

type Viewport struct {
	isFocused bool
	title     string

	onSelect func(ViewRow)

	w, h       int
	_border    lipgloss.Border
	_focused   lipgloss.Style
	_unfocused lipgloss.Style
	_selected  lipgloss.Style

	keyMap keysViewport

	selected         int
	xOffset          int
	yOffset          int
	lines            []ViewRow
	longestLineWidth int
}

func NewViewport(title string, onselect func(ViewRow)) *Viewport {
	return &Viewport{
		title:    title,
		onSelect: onselect,
		selected: -1,
		keyMap: keysViewport{
			Yank:   key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy")),
			Pgup:   key.NewBinding(key.WithKeys("pgup")),
			Pgdown: key.NewBinding(key.WithKeys("pgdown", " ")),
			Up:     key.NewBinding(key.WithKeys("up")),
			Down:   key.NewBinding(key.WithKeys("down")),
			Left:   key.NewBinding(key.WithKeys("left")),
			Right:  key.NewBinding(key.WithKeys("right")),
		},
		_border:    lipgloss.RoundedBorder(),
		_focused:   lipgloss.NewStyle().BorderForeground(HighlightColor),
		_unfocused: lipgloss.NewStyle(),
		_selected:  lipgloss.NewStyle().Background(FadedColor),
	}
}

func (v Viewport) Help() []key.Binding {
	return []key.Binding{v.keyMap.Yank}
}

func (v Viewport) Init() tea.Cmd {
	return nil
}

func (v *Viewport) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.w = msg.Width - 2
		v.h = msg.Height - 2
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, v.keyMap.Pgdown):
			v.scrollTo(v.selected + v.h)
		case key.Matches(msg, v.keyMap.Pgup):
			v.scrollTo(v.selected - v.h)
		case key.Matches(msg, v.keyMap.Down):
			v.scrollTo(v.selected + 1)
		case key.Matches(msg, v.keyMap.Up):
			v.scrollTo(v.selected - 1)
		case key.Matches(msg, v.keyMap.Left):
			v.xOffset = max(v.xOffset-1, 0)
		case key.Matches(msg, v.keyMap.Right):
			v.xOffset = max(0, min(v.xOffset+1, v.longestLineWidth-v.w))
		case key.Matches(msg, v.keyMap.Yank):
			if len(v.lines) > 0 {
				clipboard.WriteAll(v.lines[v.selected].Yank)
			}
		}
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			break
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			v.yOffset = max(0, v.yOffset-1)
		case tea.MouseButtonWheelDown:
			v.yOffset = max(0, min(len(v.lines)-v.h, v.yOffset+1))
		case tea.MouseButtonWheelLeft:
			v.xOffset = max(v.xOffset-1, 0)
		case tea.MouseButtonWheelRight:
			v.xOffset = max(0, min(v.xOffset+1, v.longestLineWidth-v.w))
		case tea.MouseButtonLeft:
			v.scrollTo(msg.Y + v.yOffset)
		}
	}

	return cmd
}

func (v *Viewport) View() string {
	s := v._unfocused
	if v.isFocused {
		s = v._focused
	}
	s = s.Border(v._border, true, false, false, true)
	lines := v.visibleLines()
	return strings.Replace(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.JoinVertical(
				lipgloss.Left,
				s.Render(lipgloss.NewStyle().
					Width(v.w).MaxWidth(v.w).
					Height(v.h).MaxHeight(v.h).
					Render(strings.Join(lines, "\n"))),
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
				len(lines),
				v.yOffset,
			),
		),
		strings.Repeat(v._border.Top, lipgloss.Width(v.title)+3),
		v._border.Top+" "+v.title+" ",
		1,
	)
}

func (v *Viewport) SetContent(lines []ViewRow) {
	v.lines = lines
	v.longestLineWidth = v.findLongestLineWidth(lines)
	v.xOffset = max(0, min(v.xOffset, v.longestLineWidth-v.w))

	if v.selected >= len(v.lines) {
		v.scrollTo(len(v.lines) - 1)
	} else {
		v.scrollTo(v.selected)
	}
}

func (v *Viewport) AddContent(lines []ViewRow) {
	v.lines = append(v.lines, lines...)
	v.longestLineWidth = max(v.longestLineWidth, v.findLongestLineWidth(lines))
	v.scrollTo(v.selected)
}

func (v Viewport) IsFocused() bool {
	return v.isFocused
}

func (v *Viewport) SetFocus(b bool) {
	v.isFocused = b
}

func (v *Viewport) scrollTo(s int) {
	s = max(0, min(s, len(v.lines)-1))
	if v.selected == s {
		return
	}
	v.selected = s
	v.yOffset = max(0, v.selected-v.h/2)
	if v.yOffset+v.h > len(v.lines) {
		v.yOffset = max(0, len(v.lines)-v.h)
	}
	if len(v.lines) > 0 && v.onSelect != nil {
		v.onSelect(v.lines[v.selected])
	}
}

func (v Viewport) visibleLines() (lines []string) {
	top := v.yOffset
	bottom := min(top+v.h, len(v.lines))
	for i, l := range v.lines[top:bottom] {
		lines = append(lines, ansi.Cut(l.Str, v.xOffset, v.xOffset+v.w))
		if i+top == v.selected && v.isFocused {
			lines[i] = v._selected.Render(lipgloss.PlaceHorizontal(v.w, lipgloss.Left, lines[i]))
		}
	}
	return lines
}

func (v *Viewport) findLongestLineWidth(lines []ViewRow) int {
	w := 0
	for _, l := range lines {
		if ww := ansi.StringWidth(l.Str); ww > w {
			w = ww
		}
	}
	return w
}
