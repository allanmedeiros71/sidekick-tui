// Package theme holds the Lip Gloss styles used by the popup.
package theme

import "github.com/charmbracelet/lipgloss"

// Theme groups the styles the UI draws with.
type Theme struct {
	Frame    lipgloss.Style
	Title    lipgloss.Style
	Item     lipgloss.Style
	Selected lipgloss.Style
	Muted    lipgloss.Style
}

// Get returns the theme with the given name, falling back to the default.
func Get(name string) Theme {
	if name == "cga" {
		return CGA()
	}
	return Default()
}

// Default is the standard theme, adapting to light and dark terminals.
func Default() Theme {
	accent := lipgloss.AdaptiveColor{Light: "#5A4FCF", Dark: "#9D8CFF"}
	return Theme{
		Frame:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(1, 2),
		Title:    lipgloss.NewStyle().Bold(true).Foreground(accent),
		Item:     lipgloss.NewStyle(),
		Selected: lipgloss.NewStyle().Bold(true).Foreground(accent),
		Muted:    lipgloss.NewStyle().Faint(true),
	}
}

// CGA mimics the cyan-on-blue look of the original 1984 SideKick.
func CGA() Theme {
	blue := lipgloss.Color("#0000AA")
	cyan := lipgloss.Color("#55FFFF")
	white := lipgloss.Color("#FFFFFF")
	yellow := lipgloss.Color("#FFFF55")
	base := lipgloss.NewStyle().Background(blue).Foreground(cyan)
	return Theme{
		Frame:    base.Border(lipgloss.DoubleBorder()).BorderForeground(white).BorderBackground(blue).Padding(1, 2),
		Title:    base.Bold(true).Foreground(yellow),
		Item:     base,
		Selected: lipgloss.NewStyle().Background(cyan).Foreground(blue).Bold(true),
		Muted:    base.Foreground(white),
	}
}
