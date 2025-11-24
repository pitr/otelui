package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	minSplit        = 6
	splitStep       = 2
	splitBorderSize = 1
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
}

type Splitview[T SplitableModel[T]] struct {
	w         int
	h         [2]int
	views     [2]T
	keyMap    keysSplitview
	_selected lipgloss.Style
}

func NewSplitview[T SplitableModel[T]](top, bottom T) Splitview[T] {
	top.SetFocus(true)
	bottom.SetFocus(false)

	return Splitview[T]{
		h:     [2]int{0, 0},
		views: [2]T{top, bottom},
		keyMap: keysSplitview{
			Increase: key.NewBinding(key.WithKeys("="), key.WithHelp("-/=", "resize")),
			Decrease: key.NewBinding(key.WithKeys("-")),
			Next:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
			Prev:     key.NewBinding(key.WithKeys("shift+tab")),
		},
	}
}

func (m Splitview[T]) Init() tea.Cmd {
	return nil
}

func (m Splitview[T]) Update(msg tea.Msg) (Splitview[T], tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h[0] = msg.Height / 2
		m.h[1] = msg.Height - m.h[0]
		cmd = tea.Batch(
			m.views[0].Update(tea.WindowSizeMsg{Width: m.w, Height: m.h[0]}),
			m.views[1].Update(tea.WindowSizeMsg{Width: m.w, Height: m.h[1]}),
		)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Increase):
			i := 0
			if m.views[1].IsFocused() {
				i = 1
			}
			if m.h[1-i] >= minSplit+splitStep {
				m.h[1-i] -= splitStep
				m.h[i] += splitStep
				cmd = tea.Batch(
					m.views[0].Update(tea.WindowSizeMsg{Width: m.w, Height: m.h[0]}),
					m.views[1].Update(tea.WindowSizeMsg{Width: m.w, Height: m.h[1]}),
				)
			}
		case key.Matches(msg, m.keyMap.Decrease):
			i := 0
			if m.views[1].IsFocused() {
				i = 1
			}
			if m.h[i] >= minSplit+splitStep {
				m.h[i] -= splitStep
				m.h[1-i] += splitStep
				cmd = tea.Batch(
					m.views[0].Update(tea.WindowSizeMsg{Width: m.w, Height: m.h[0]}),
					m.views[1].Update(tea.WindowSizeMsg{Width: m.w, Height: m.h[1]}),
				)
			}
		case key.Matches(msg, m.keyMap.Next, m.keyMap.Prev):
			m.views[0].SetFocus(!m.views[0].IsFocused())
			m.views[1].SetFocus(!m.views[1].IsFocused())
		default:
			for i := range m.views {
				if m.views[i].IsFocused() {
					cmd = m.views[i].Update(msg)
					break
				}
			}
		}
	case tea.MouseMsg:
		leftClick := msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress
		view := 0
		inside := false
		switch {
		case msg.Y > 0 && msg.Y < m.h[0]-splitBorderSize:
			inside = true
			view = 0
			msg.Y -= splitBorderSize
		case msg.Y > m.h[0] && msg.Y < m.h[0]+m.h[1]-splitBorderSize:
			inside = true
			msg.Y -= m.h[0] + splitBorderSize
			fallthrough
		case msg.Y == m.h[0] || msg.Y == m.h[0]+m.h[1]-splitBorderSize:
			view = 1
		}

		switch {
		case leftClick:
			m.views[1-view].SetFocus(false)
			m.views[view].SetFocus(true)
			fallthrough
		default:
			if inside {
				cmd = m.views[view].Update(msg)
			}
		}
	default:
		for i := range m.views {
			if m.views[i].IsFocused() {
				cmd = m.views[i].Update(msg)
				break
			}
		}
	}

	return m, cmd
}

func (m Splitview[T]) View() string {
	var buf strings.Builder
	buf.WriteString(m.views[0].View())
	buf.WriteByte('\n')
	buf.WriteString(m.views[1].View())
	return buf.String()
}

func (m *Splitview[T]) Get(i int) T {
	return m.views[i]
}

func (m Splitview[T]) Help() []key.Binding {
	keys := []key.Binding{m.keyMap.Increase, m.keyMap.Next}
	for i := range m.views {
		if m.views[i].IsFocused() {
			keys = append(m.views[i].Help(), keys...)
			break
		}
	}
	return keys
}
