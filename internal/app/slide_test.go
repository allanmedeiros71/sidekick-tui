package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/allanmedeiros71/sidekick-tui/internal/config"
)

func animated(from string) config.Config {
	cfg := config.Default()
	cfg.Animation = config.AnimationConfig{Slide: from, DurationMS: 160}
	return cfg
}

func TestSlideRunsToCompletion(t *testing.T) {
	var m tea.Model = New(animated("right"))
	if m.Init() == nil {
		t.Fatal("expected Init to start the animation")
	}
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	for i := 0; i < 100 && m.(Model).slide.running(); i++ {
		var cmd tea.Cmd
		m, cmd = m.Update(slideTickMsg{})
		if m.(Model).slide.running() && cmd == nil {
			t.Fatal("expected another tick while the slide runs")
		}
	}
	if m.(Model).slide.running() {
		t.Fatal("slide never finished")
	}
	want := New(config.Config{Theme: "default"})
	want.width, want.height = 120, 40
	if m.View() != want.View() {
		t.Fatal("finished slide should match the static layout")
	}
}

func TestSlideNoneDoesNotAnimate(t *testing.T) {
	m := New(animated("none"))
	if m.slide.running() {
		t.Fatal(`slide = "none" should not animate`)
	}
}

func TestSlideStaysInsideWindow(t *testing.T) {
	for _, from := range []string{"left", "right"} {
		m := New(animated(from))
		m.width, m.height = 80, 30
		for ; m.slide.running(); m.slide.frame++ {
			view := m.View()
			if w := lipgloss.Width(view); w > m.width {
				t.Fatalf("%s frame %d: width %d > %d", from, m.slide.frame, w, m.width)
			}
			if h := lipgloss.Height(view); h != m.height {
				t.Fatalf("%s frame %d: height %d != %d", from, m.slide.frame, h, m.height)
			}
		}
	}
}

func TestSlideStartsOffscreenOnItsSide(t *testing.T) {
	m := New(animated("right"))
	m.width, m.height = 80, 30
	m.slide.frame = 1
	for _, l := range strings.Split(m.View(), "\n") {
		if strings.TrimSpace(l) != "" && !strings.HasPrefix(l, strings.Repeat(" ", 50)) {
			t.Fatalf("early right slide should hug the right edge, got %q", l)
		}
	}
}
