package components

import (
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// InputCapture is implemented by models that control input (eg. for search)
type InputCapture interface{ IsCapturingInput() bool }

type ViewRow struct {
	Str    string
	Yank   string
	Raw    any
	Search string
}

type keysViewport struct {
	Yank   key.Binding
	Pgdown key.Binding
	Pgup   key.Binding
	Down   key.Binding
	Up     key.Binding
	Left   key.Binding
	Right  key.Binding
	Search key.Binding
	Esc    key.Binding
}

type Viewport struct {
	isFocused bool
	title     string

	onSelect func(ViewRow)

	w, h      int
	_border   lipgloss.Border
	_focused  lipgloss.TerminalColor
	_selected lipgloss.Style

	keyMap keysViewport

	selected         int
	xOffset          int
	yOffset          int
	lines            []ViewRow
	allLines         []ViewRow
	longestLineWidth int

	searchable   bool
	searching    bool
	searchInput  textinput.Model
	searchFilter string
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
			Search: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
			Esc:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel search")),
		},
		_border:   lipgloss.RoundedBorder(),
		_focused:  AccentColor,
		_selected: lipgloss.NewStyle().Background(SelectionColor),
	}
}

func (v Viewport) Init() tea.Cmd { return nil }

func (v *Viewport) WithSearch() *Viewport {
	v.searchable = true
	ti := textinput.New()
	ti.Prompt = "/"
	ti.CharLimit = 128
	v.searchInput = ti
	return v
}

func (v Viewport) Help() []key.Binding {
	if v.searchable && v.searching {
		return []key.Binding{v.keyMap.Search, v.keyMap.Esc}
	}
	if v.searchable {
		return []key.Binding{v.keyMap.Search, v.keyMap.Yank}
	}
	return []key.Binding{v.keyMap.Yank}
}

func (v *Viewport) SetFocus(b bool)       { v.isFocused = b }
func (v Viewport) IsFocused() bool        { return v.isFocused }
func (v Viewport) IsCapturingInput() bool { return v.searching }

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
		case key.Matches(msg, v.keyMap.Left) && !v.searching:
			v.xOffset = max(v.xOffset-1, 0)
		case key.Matches(msg, v.keyMap.Right) && !v.searching:
			v.xOffset = max(0, min(v.xOffset+1, v.longestLineWidth-v.w))
		case key.Matches(msg, v.keyMap.Yank) && !v.searching:
			if len(v.lines) > 0 {
				clipboard.WriteAll(v.lines[v.selected].Yank)
			}
		case key.Matches(msg, v.keyMap.Esc) && v.searching:
			wasFiltered := v.searchFilter != ""
			v.searching = false
			v.searchInput.Blur()
			v.searchFilter = ""
			v.lines = v.allLines
			v.longestLineWidth = v.findLongestLineWidth(v.lines)
			if wasFiltered {
				v.scrollTo(0)
			}
		case key.Matches(msg, v.keyMap.Search) && v.searchable && !v.searching:
			v.searching = true
			v.searchInput.SetValue("")
			v.searchFilter = ""
			cmd = v.searchInput.Focus()
		case v.searching:
			v.searchInput, cmd = v.searchInput.Update(msg)
			v.searchFilter = strings.ToLower(v.searchInput.Value())
			v.lines = v.filterLines(v.allLines)
			v.longestLineWidth = v.findLongestLineWidth(v.lines)
			v.scrollTo(0)
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
	bs := lipgloss.NewStyle().Border(v._border, false, false, false, true)
	fg := lipgloss.NewStyle()
	if v.isFocused {
		bs = bs.BorderForeground(v._focused)
		fg = fg.Foreground(v._focused)
	}
	lines := v.visibleLines()
	hscroll := Scrollbar(bs, ScrollbarHorizontal, v.w, v.longestLineWidth, v.w, v.xOffset)
	vscroll := Scrollbar(bs, ScrollbarVertical, v.h, len(v.lines), len(lines), v.yOffset)
	content := bs.Render(lipgloss.NewStyle().
		Width(v.w).MaxWidth(v.w).
		Height(v.h).MaxHeight(v.h).
		Render(strings.Join(lines, "\n")))

	var top string
	if v.searching {
		top = fg.Render(v._border.TopLeft+v._border.Top) + v.searchInput.View()
	} else {
		top = fg.Render(v._border.TopLeft+v._border.Top, v.title+" ")
	}
	top += fg.Render(strings.Repeat(v._border.Top, max(0, v.w+1-lipgloss.Width(top))))

	return lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.JoinVertical(lipgloss.Left, top, content, hscroll), vscroll)
}

func (v *Viewport) SetContent(lines []ViewRow) {
	v.allLines = lines
	if v.searching {
		v.lines = v.filterLines(lines)
	} else {
		v.lines = lines
	}
	v.longestLineWidth = v.findLongestLineWidth(v.lines)
	v.xOffset = max(0, min(v.xOffset, v.longestLineWidth-v.w))

	if v.selected >= len(v.lines) {
		v.scrollTo(len(v.lines) - 1)
	} else {
		v.scrollTo(v.selected)
		if v.onSelect == nil {
			return
		}
		if len(v.lines) > 0 && v.selected >= 0 {
			v.onSelect(v.lines[v.selected])
		} else {
			v.onSelect(ViewRow{})
		}
	}
}

func (v *Viewport) AddContent(lines []ViewRow) {
	v.allLines = append(v.allLines, lines...)
	if v.searching {
		v.lines = append(v.lines, v.filterLines(lines)...)
	} else {
		v.lines = append(v.lines, lines...)
	}
	v.longestLineWidth = max(v.longestLineWidth, v.findLongestLineWidth(lines))
	v.scrollTo(v.selected)
}

func (v *Viewport) filterLines(all []ViewRow) []ViewRow {
	if v.searchFilter == "" {
		return all
	}
	var out []ViewRow
	for _, l := range all {
		if strings.Contains(strings.ToLower(l.Yank), v.searchFilter) ||
			(l.Search != "" && strings.Contains(strings.ToLower(l.Search), v.searchFilter)) {
			out = append(out, l)
		}
	}
	return out
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
	} else if v.onSelect != nil {
		v.onSelect(ViewRow{})
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
