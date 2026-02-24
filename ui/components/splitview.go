package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	minSplit  = 6
	splitStep = 2
)

type SplitableModel[V any] interface {
	IsFocused() bool
	SetFocus(bool)
	Init() tea.Cmd
	Update(tea.Msg) tea.Cmd
	View() string
	Helpful
}

type keysSplitview struct {
	Increase key.Binding
	Decrease key.Binding
	Next     key.Binding
	Prev     key.Binding
	Enter    key.Binding
	Esc      key.Binding
}

type Splitview[T SplitableModel[T], B SplitableModel[B]] struct {
	w         int
	h         [2]int
	top       T
	bot       B
	keyMap    keysSplitview
	_selected lipgloss.Style
}

func NewSplitview[T SplitableModel[T], B SplitableModel[B]](top T, bot B) Splitview[T, B] {
	top.SetFocus(true)
	bot.SetFocus(false)

	return Splitview[T, B]{
		h:   [2]int{0, 0},
		top: top,
		bot: bot,
		keyMap: keysSplitview{
			Increase: key.NewBinding(key.WithKeys("="), key.WithHelp("- =", "resize")),
			Decrease: key.NewBinding(key.WithKeys("-")),
			Next:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
			Prev:     key.NewBinding(key.WithKeys("shift+tab")),
			Enter:    key.NewBinding(key.WithKeys("enter")),
			Esc:      key.NewBinding(key.WithKeys("esc")),
		},
	}
}

func (m Splitview[T, B]) Init() tea.Cmd {
	return nil
}

func (m Splitview[T, B]) Update(msg tea.Msg) (Splitview[T, B], tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h[0] = msg.Height / 2
		m.h[1] = msg.Height - m.h[0]
		cmd = tea.Batch(
			m.top.Update(tea.WindowSizeMsg{Width: m.w, Height: m.h[0]}),
			m.bot.Update(tea.WindowSizeMsg{Width: m.w, Height: m.h[1]}),
		)
	case tea.KeyMsg:
		capturing := false
		if ic, ok := any(m.top).(InputCapture); ok && m.top.IsFocused() {
			capturing = ic.IsCapturingInput()
		} else if ic, ok := any(m.bot).(InputCapture); ok && m.bot.IsFocused() {
			capturing = ic.IsCapturingInput()
		}
		switch {
		case key.Matches(msg, m.keyMap.Increase) && !capturing:
			i := 0
			if m.bot.IsFocused() {
				i = 1
			}
			if m.h[1-i] >= minSplit+splitStep {
				m.h[1-i] -= splitStep
				m.h[i] += splitStep
				cmd = tea.Batch(
					m.top.Update(tea.WindowSizeMsg{Width: m.w, Height: m.h[0]}),
					m.bot.Update(tea.WindowSizeMsg{Width: m.w, Height: m.h[1]}),
				)
			}
		case key.Matches(msg, m.keyMap.Decrease) && !capturing:
			i := 0
			if m.bot.IsFocused() {
				i = 1
			}
			if m.h[i] >= minSplit+splitStep {
				m.h[i] -= splitStep
				m.h[1-i] += splitStep
				cmd = tea.Batch(
					m.top.Update(tea.WindowSizeMsg{Width: m.w, Height: m.h[0]}),
					m.bot.Update(tea.WindowSizeMsg{Width: m.w, Height: m.h[1]}),
				)
			}
		case key.Matches(msg, m.keyMap.Next, m.keyMap.Prev):
			fallthrough
		case key.Matches(msg, m.keyMap.Enter) && m.top.IsFocused():
			fallthrough
		case key.Matches(msg, m.keyMap.Esc) && m.bot.IsFocused() && !capturing:
			m.top.SetFocus(!m.top.IsFocused())
			m.bot.SetFocus(!m.bot.IsFocused())
		default:
			if m.top.IsFocused() {
				cmd = m.top.Update(msg)
			} else {
				cmd = m.bot.Update(msg)
			}
		}
	default:
		if m.top.IsFocused() {
			cmd = m.top.Update(msg)
		} else {
			cmd = m.bot.Update(msg)
		}
	}

	return m, cmd
}

func (m Splitview[T, B]) View() string {
	var buf strings.Builder
	buf.WriteString(m.top.View())
	buf.WriteByte('\n')
	buf.WriteString(m.bot.View())
	return buf.String()
}

func (m *Splitview[T, B]) Top() T { return m.top }
func (m *Splitview[T, B]) Bot() B { return m.bot }

func (m Splitview[T, B]) Help() []key.Binding {
	keys := []key.Binding{m.keyMap.Next, m.keyMap.Increase}
	ckeys := []key.Binding{m.keyMap.Next}
	if m.top.IsFocused() {
		if c, ok := any(m.top).(InputCapture); ok && m.top.IsFocused() && c.IsCapturingInput() {
			keys = append(ckeys, m.top.Help()...)
		} else {
			keys = append(keys, m.top.Help()...)
		}
	} else {
		if c, ok := any(m.bot).(InputCapture); ok && m.bot.IsFocused() && c.IsCapturingInput() {
			keys = append(ckeys, m.bot.Help()...)
		} else {
			keys = append(keys, m.bot.Help()...)
		}
	}
	return keys
}
