package components

import (
	"time"

	"github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"pitr.ca/otelui/server"
)

var TZUTC bool

type Timeseries struct {
	isFocused bool
	title     string

	model timeserieslinechart.Model

	w, h int

	name string
}

func NewTimeseries(title string) *Timeseries {
	return &Timeseries{
		title: title,
		model: timeserieslinechart.New(0, 0,
			timeserieslinechart.WithUpdateHandler(timeserieslinechart.SecondNoZoomUpdateHandler(1)),
		),
	}
}

func (t Timeseries) Help() []key.Binding     { return []key.Binding{} }
func (t Timeseries) Init() tea.Cmd           { return nil }
func (t *Timeseries) SetContent(name string) { t.name = name }
func (t Timeseries) IsFocused() bool         { return t.isFocused }

func (t *Timeseries) SetFocus(b bool) {
	t.isFocused = b
	if b {
		t.model.AxisStyle = lipgloss.NewStyle().Foreground(AccentColor)
	} else {
		t.model.AxisStyle = lipgloss.NewStyle()
	}
}

func (t *Timeseries) Update(msg tea.Msg) (cmd tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.w = msg.Width
		t.h = msg.Height
		t.model.Resize(t.w, t.h)
		t.model.Focus()
	default:
		t.model, cmd = t.model.Update(msg)
	}

	return cmd
}

func (t *Timeseries) View() string {
	if t.name == "" {
		return lipgloss.NewStyle().Width(t.w).Height(t.h).Render("")
	}
	if TZUTC {
		t.model.XLabelFormatter = func(_ int, v float64) string {
			return time.Unix(int64(v), 0).UTC().Format("15:04:05")
		}
	} else {
		t.model.XLabelFormatter = func(_ int, v float64) string {
			return time.Unix(int64(v), 0).Local().Format("15:04:05")
		}
	}
	t.model.Clear()
	t.model.ClearAllData()
	t.model.SetViewXYRange(float64(time.Now().Unix()), float64(time.Now().Unix()), 0, 1)
	t.model.SetYRange(0, 1)
	dps := server.GetDatapoints(t.name)
	if dps == nil {
		return lipgloss.NewStyle().Width(t.w).Height(t.h).Render("")
	}
	for i, ts := range dps.Times {
		t.model.Push(timeserieslinechart.TimePoint{Time: time.Unix(0, int64(ts)), Value: dps.Values[i]})
	}
	t.model.DrawAll()
	return t.model.View()
}
