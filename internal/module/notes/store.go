package notes

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Paths below the data directory.
const (
	notesDir    = "notes"
	globalNote  = "scratch.md"
	projectsDir = "projects"
)

// GlobalPath is the scratchpad used outside a git repository.
func GlobalPath(dataDir string) string {
	return filepath.Join(dataDir, notesDir, globalNote)
}

// ProjectPath is the scratchpad of project p.
func ProjectPath(dataDir string, p Project) string {
	return filepath.Join(dataDir, notesDir, projectsDir, filepath.FromSlash(p.Key)+".md")
}

// Note is a scratchpad file. It remembers what it last read or wrote, so a
// save never overwrites changes made to the file by something else, such as
// a sync or another sidekick window.
type Note struct {
	Path  string
	known string
}

// Open reads the note at path. A missing file is an empty note.
func Open(path string) (*Note, string, error) {
	text, err := read(path)
	if err != nil {
		return nil, "", err
	}
	return &Note{Path: path, known: text}, text, nil
}

// Save writes text to the note. When the file changed on disk since it was
// last read or written, text goes to a conflict copy next to it instead and
// the note follows that copy; conflict is then the copy's path.
func (n *Note) Save(text string) (conflict string, err error) {
	if text == n.known {
		return "", nil
	}
	disk, err := read(n.Path)
	if err != nil {
		return "", err
	}
	if disk != n.known {
		conflict = conflictPath(n.Path, time.Now())
		if err := write(conflict, text); err != nil {
			return "", err
		}
		n.Path, n.known = conflict, text
		return conflict, nil
	}
	if err := write(n.Path, text); err != nil {
		return "", err
	}
	n.known = text
	return "", nil
}

// Append adds text to the end of the note at path as new lines.
func Append(path, text string) error {
	old, err := read(path)
	if err != nil {
		return err
	}
	if old != "" && !strings.HasSuffix(old, "\n") {
		old += "\n"
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return write(path, old+text)
}

func conflictPath(path string, now time.Time) string {
	return strings.TrimSuffix(path, ".md") + ".conflict-" + now.Format("20060102-150405") + ".md"
}

func read(path string) (string, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return string(b), err
}

// write replaces the file atomically, so a crash or a sync never sees half a note.
func write(path, text string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name()) // no-op after the rename
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(f.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
