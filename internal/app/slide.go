package app

import (
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/allanmedeiros71/sidekick-tui/internal/config"
)

// frameInterval paces the opening animation at roughly 60 frames per second.
const frameInterval = 16 * time.Millisecond

// slideTickMsg advances the opening animation by one frame.
type slideTickMsg struct{}

func slideTick() tea.Cmd {
	return tea.Tick(frameInterval, func(time.Time) tea.Msg { return slideTickMsg{} })
}

// slide is the state of the opening animation: the frame slides in from
// one side of the window, like the original SideKick windows did.
type slide struct {
	from   string // "left" or "right"
	frame  int
	frames int
}

func newSlide(cfg config.AnimationConfig) slide {
	if cfg.Slide != "left" && cfg.Slide != "right" {
		return slide{}
	}
	return slide{from: cfg.Slide, frames: int(time.Duration(cfg.DurationMS) * time.Millisecond / frameInterval)}
}

func (s slide) running() bool { return s.frame < s.frames }

// progress returns how far the frame has travelled, from 0 to 1, eased out
// so it decelerates as it reaches its place.
func (s slide) progress() float64 {
	if s.frames == 0 {
		return 1
	}
	p := float64(s.frame) / float64(s.frames)
	return 1 - math.Pow(1-p, 3)
}

// place draws box inside a width x height window, shifted horizontally
// toward the side it enters from and clipped at the window edge.
func (s slide) place(box string, width, height int) string {
	boxWidth := lipgloss.Width(box)
	final := max((width-boxWidth)/2, 0)
	rest := 1 - s.progress()

	var x int
	if s.from == "left" {
		x = final - int(math.Round(float64(final+boxWidth)*rest))
	} else {
		x = final + int(math.Round(float64(width-final)*rest))
	}

	lines := strings.Split(box, "\n")
	for i, l := range lines {
		if x < 0 {
			l = ansi.TruncateLeft(l, -x, "")
		}
		start := max(x, 0)
		l = ansi.Truncate(l, max(width-start, 0), "")
		lines[i] = strings.Repeat(" ", start) + l
	}
	return lipgloss.PlaceVertical(height, lipgloss.Center, strings.Join(lines, "\n"))
}
