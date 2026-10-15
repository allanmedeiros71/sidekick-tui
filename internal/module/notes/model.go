// Package notes is the scratchpad: one markdown note per git repository,
// picked from the directory sidekick is started in, and a global note for
// everything else.
package notes

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/allanmedeiros71/sidekick-tui/internal/config"
	"github.com/allanmedeiros71/sidekick-tui/internal/theme"
)

const (
	// frameWidth and frameHeight are what the popup frame (border and
	// padding) takes around the module.
	frameWidth  = 6
	frameHeight = 4
	// chromeHeight is the header and help rows around the text area.
	chromeHeight = 5
	// saveDelay is how long typing must pause before the note is saved.
	saveDelay = 500 * time.Millisecond
	// maxLines is the most lines the text area holds; longer notes open
	// read-only so they are never cut short on save.
	maxLines = 9000
)

type (
	saveTickMsg   struct{ rev int }
	editorDoneMsg struct{ err error }
)

// Options say where the scratchpad looks for the project and keeps notes.
type Options struct {
	Dir     string // working directory used to detect the git repository
	DataDir string // sidekick's data directory
}

// Model is the scratchpad's Bubble Tea model. The note is saved shortly
// after typing stops and when the module is closed.
type Model struct {
	theme    theme.Theme
	area     textarea.Model
	opts     Options
	project  Project
	inRepo   bool
	global   bool
	note     *Note
	readOnly bool
	rev      int
	notice   string
	err      error
	width    int
	height   int
}

// New returns a scratchpad for the current directory, stored in the
// configured data directory.
func New(t theme.Theme) *Model {
	dir, _ := os.Getwd()
	data, err := config.DataDir()
	m := NewWith(t, Options{Dir: dir, DataDir: data})
	if err != nil {
		m.err = err
	}
	return m
}

// NewWith returns a scratchpad with explicit directories.
func NewWith(t theme.Theme, opts Options) *Model {
	area := textarea.New()
	area.ShowLineNumbers = false
	area.Prompt = ""
	area.Placeholder = "type away; the note saves itself"
	area.CharLimit = 0
	area.MaxHeight = 0
	focused, blurred := textarea.DefaultStyles()
	for _, s := range []*textarea.Style{&focused, &blurred} {
		s.Base = t.Item
		s.Text = t.Item
		s.CursorLine = t.Item
		s.EndOfBuffer = t.Item
		s.Placeholder = t.Muted
	}
	area.FocusedStyle, area.BlurredStyle = focused, blurred
	area.Focus()

	m := &Model{theme: t, area: area, opts: opts}
	m.project, m.inRepo = Detect(opts.Dir)
	m.load()
	return m
}

// ID implements module.Module.
func (m *Model) ID() string { return "notes" }

// Title implements module.Module.
func (m *Model) Title() string { return "Scratchpad" }

// Path is the file of the note being edited.
func (m *Model) Path() string {
	if m.note == nil {
		return ""
	}
	return m.note.Path
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd { return textarea.Blink }

// Close saves pending changes. The app calls it when the module is left.
func (m *Model) Close() error {
	m.save()
	return m.err
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.area.SetWidth(max(20, msg.Width-frameWidth))
		m.area.SetHeight(max(3, msg.Height-frameHeight-chromeHeight))
		return m, nil
	case saveTickMsg:
		if msg.rev == m.rev {
			m.save()
		}
		return m, nil
	case editorDoneMsg:
		if msg.err != nil {
			m.notice = "editor: " + msg.err.Error()
		}
		m.reload()
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlS:
			if m.save() {
				m.notice = "saved"
			}
			return m, nil
		case tea.KeyCtrlG:
			return m, m.toggle()
		case tea.KeyCtrlE:
			return m, m.edit()
		}
		if m.readOnly {
			return m, nil
		}
	}
	before := m.area.Value()
	var cmd tea.Cmd
	m.area, cmd = m.area.Update(msg)
	if m.area.Value() != before {
		m.rev++
		m.notice = ""
		rev := m.rev
		cmd = tea.Batch(cmd, tea.Tick(saveDelay, func(time.Time) tea.Msg { return saveTickMsg{rev} }))
	}
	return m, cmd
}

// path is the note to show: the project's, unless outside a repository or
// switched to the global one.
func (m *Model) path() string {
	if m.inRepo && !m.global {
		return ProjectPath(m.opts.DataDir, m.project)
	}
	return GlobalPath(m.opts.DataDir)
}

func (m *Model) load() {
	m.openFile(m.path())
}

// reload reads the current note again, after an external editor changed it.
func (m *Model) reload() {
	if m.note != nil {
		m.openFile(m.note.Path)
	}
}

func (m *Model) openFile(path string) {
	note, text, err := Open(path)
	if err != nil {
		m.err, m.note = err, nil
		m.area.SetValue("")
		return
	}
	m.err, m.note = nil, note
	m.area.SetValue(text)
	m.readOnly = !fits(m.area.Value(), text)
	if m.readOnly {
		m.notice = "too long or unusual to edit here: read-only, ctrl+e opens $EDITOR"
	}
}

// fits reports whether the text area holds text without losing anything
// on save. Tabs turn into spaces, which is accepted.
func fits(value, text string) bool {
	if strings.Count(text, "\n") >= maxLines {
		return false
	}
	return value == strings.ReplaceAll(text, "\t", "    ")
}

// save writes the note if it changed and reports whether it succeeded.
func (m *Model) save() bool {
	if m.note == nil || m.readOnly {
		return m.err == nil
	}
	conflict, err := m.note.Save(m.area.Value())
	m.err = err
	if conflict != "" {
		m.notice = "the note changed on disk; yours is in " + filepath.Base(conflict)
	}
	return err == nil
}

func (m *Model) toggle() tea.Cmd {
	if !m.inRepo {
		m.notice = "not in a git repository: this is the global note"
		return nil
	}
	if !m.save() {
		return nil
	}
	m.global = !m.global
	m.notice = ""
	m.load()
	return nil
}

// edit opens the note in $VISUAL or $EDITOR and reloads it afterwards.
func (m *Model) edit() tea.Cmd {
	if m.note == nil || !m.save() {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(m.note.Path), 0o755); err != nil {
		m.err = err
		return nil
	}
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	// Through the shell, so EDITOR can carry arguments ("code -w").
	cmd := exec.Command("sh", "-c", editor+` "$1"`, "sh", m.note.Path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return editorDoneMsg{err} })
}

// View implements tea.Model.
func (m *Model) View() string {
	t := m.theme
	width := m.width
	if width == 0 {
		width = 80
	}
	width -= frameWidth

	name := "global"
	if m.inRepo && !m.global {
		name = m.project.Name
	}
	var b strings.Builder
	b.WriteString(t.Title.Render("Scratchpad") + t.Muted.Render(" · ") + t.Item.Render(truncate(name, max(10, width-13))) + "\n")
	b.WriteString(t.Muted.Render(truncate(tildePath(m.Path()), width)) + "\n\n")
	b.WriteString(m.area.View() + "\n\n")

	help := "esc back · ctrl+s save · ctrl+e $EDITOR"
	if m.inRepo {
		other := "global"
		if m.global {
			other = "project"
		}
		help += " · ctrl+g " + other
	}
	switch {
	case m.err != nil:
		b.WriteString(t.Error.Render(truncate(m.err.Error(), width)))
	case m.notice != "":
		b.WriteString(t.Muted.Render(truncate(m.notice, width)))
	default:
		b.WriteString(t.Muted.Render(truncate(help, width)))
	}
	return b.String()
}

func tildePath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if rel, err := filepath.Rel(home, p); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.Join("~", rel)
	}
	return p
}

// truncate shortens s to at most width cells, ending in "…" when cut.
func truncate(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	r := []rune(s)
	r = r[:min(len(r), width)]
	for len(r) > 0 && lipgloss.Width(string(r))+1 > width {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}
