// Package module defines the contract every sidekick tool implements.
package module

import tea "github.com/charmbracelet/bubbletea"

// Module is one tool in the sidekick panel (calculator, inspector, scratchpad...).
// Each module is a self-contained Bubble Tea model shown inside the popup frame.
type Module interface {
	tea.Model
	// ID is a stable identifier used in config and keybindings.
	ID() string
	// Title is the human-readable name shown in the menu.
	Title() string
}

// Closer is implemented by modules that hold unsaved state. The app calls
// Close when the module is left or sidekick quits.
type Closer interface {
	Close() error
}

// Planned lists the modules on the roadmap, in menu order. Entries that are
// not implemented yet are shown in the menu as placeholders.
var Planned = []struct{ ID, Title, Description string }{
	{"calc", "Calculator", "hex/bin, bitwise, epoch, chmod, CIDR, sizes"},
	{"inspect", "Inspector", "base64, URL, JWT, hashes, UUID, JSON"},
	{"notes", "Scratchpad", "per-project notes, detects the current git repo"},
	{"dial", "Connections", "SSH, PostgreSQL, URLs: dial to connect"},
	{"runbooks", "Runbooks", "commands with placeholders"},
	{"clip", "Clipboard", "history and snippets"},
	{"gitlab", "GitLab", "merge requests and pipeline status"},
}
