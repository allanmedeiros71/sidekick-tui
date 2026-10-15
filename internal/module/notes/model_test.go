package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/allanmedeiros71/sidekick-tui/internal/theme"
)

func typeText(m *Model, s string) tea.Cmd {
	var cmd tea.Cmd
	for _, r := range s {
		k := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		if r == '\n' {
			k = tea.KeyMsg{Type: tea.KeyEnter}
		}
		_, cmd = m.Update(k)
	}
	return cmd
}

func TestModelSavesAfterTyping(t *testing.T) {
	dir := gitRepo(t, "https://github.com/allan/app.git")
	data := t.TempDir()
	m := NewWith(theme.Default(), Options{Dir: dir, DataDir: data})
	want := filepath.Join(data, "notes", "projects", "github.com", "allan", "app.md")
	if m.Path() != want {
		t.Fatalf("path = %q, want %q", m.Path(), want)
	}
	typeText(m, "a\nb")
	// The pending save fires once typing stops; an older tick is ignored.
	m.Update(saveTickMsg{rev: m.rev - 1})
	if _, err := os.Stat(want); !os.IsNotExist(err) {
		t.Fatal("a stale tick should not save")
	}
	m.Update(saveTickMsg{rev: m.rev})
	if got := readFile(t, want); got != "a\nb" {
		t.Errorf("saved %q", got)
	}
	if v := m.View(); !strings.Contains(v, "allan/app") || !strings.Contains(v, "ctrl+g global") {
		t.Errorf("view should name the project:\n%s", v)
	}

	// Reopening shows the saved note; Close saves what is pending.
	m = NewWith(theme.Default(), Options{Dir: dir, DataDir: data})
	if m.area.Value() != "a\nb" {
		t.Fatalf("reopened with %q", m.area.Value())
	}
	typeText(m, "c")
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, want); got != "a\nbc" {
		t.Errorf("saved %q on close", got)
	}
}

func TestModelToggleGlobal(t *testing.T) {
	dir := gitRepo(t, "")
	data := t.TempDir()
	m := NewWith(theme.Default(), Options{Dir: dir, DataDir: data})
	typeText(m, "project")
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	if m.Path() != GlobalPath(data) || m.area.Value() != "" {
		t.Fatalf("ctrl+g should open the global note, got %q", m.Path())
	}
	typeText(m, "global")
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	if m.area.Value() != "project" {
		t.Errorf("back on the project note, got %q", m.area.Value())
	}
	if got := readFile(t, GlobalPath(data)); got != "global" {
		t.Errorf("global note = %q", got)
	}
}

func TestModelOutsideRepo(t *testing.T) {
	data := t.TempDir()
	m := NewWith(theme.Default(), Options{Dir: t.TempDir(), DataDir: data})
	if m.Path() != GlobalPath(data) {
		t.Fatalf("path = %q", m.Path())
	}
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	if !strings.Contains(m.View(), "not in a git repository") {
		t.Error("ctrl+g outside a repository should explain why nothing changed")
	}
}

func TestModelReadOnlyWhenLossy(t *testing.T) {
	data := t.TempDir()
	path := GlobalPath(data)
	text := "bell\a\n"
	if err := write(path, text); err != nil {
		t.Fatal(err)
	}
	m := NewWith(theme.Default(), Options{Dir: t.TempDir(), DataDir: data})
	if !m.readOnly {
		t.Fatal("a note the text area would change should open read-only")
	}
	typeText(m, "x")
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, path); got != text {
		t.Errorf("read-only note was rewritten as %q", got)
	}
}

func TestFits(t *testing.T) {
	if !fits("a    b", "a\tb") {
		t.Error("tabs becoming spaces is accepted")
	}
	if fits("ab", "a\x00b") {
		t.Error("dropped characters are not")
	}
	if fits("", strings.Repeat("\n", maxLines)) {
		t.Error("very long notes are not")
	}
}

func TestModelFitsWindow(t *testing.T) {
	m := NewWith(theme.Default(), Options{Dir: t.TempDir(), DataDir: t.TempDir()})
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	typeText(m, strings.Repeat("line\n", 40))
	if n := strings.Count(m.View(), "\n") + 1; n > 20-frameHeight {
		t.Errorf("view has %d rows, window fits %d", n, 20-frameHeight)
	}
}
