package notes

import (
	"os"
	"path/filepath"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestPaths(t *testing.T) {
	if got := GlobalPath("/d"); got != filepath.FromSlash("/d/notes/scratch.md") {
		t.Errorf("GlobalPath = %q", got)
	}
	p := Project{Key: "github.com/a/b"}
	if got := ProjectPath("/d", p); got != filepath.FromSlash("/d/notes/projects/github.com/a/b.md") {
		t.Errorf("ProjectPath = %q", got)
	}
}

func TestSaveCreatesAndSkipsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x", "n.md")
	n, text, err := Open(path)
	if err != nil || text != "" {
		t.Fatalf("Open = %q, %v", text, err)
	}
	if _, err := n.Save(""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("an empty new note should not create a file")
	}
	if _, err := n.Save("hello\n"); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, path); got != "hello\n" {
		t.Errorf("file = %q", got)
	}
}

func TestSaveKeepsBothOnConflict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "n.md")
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, _, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("theirs"), 0o644); err != nil {
		t.Fatal(err)
	}
	conflict, err := n.Save("mine")
	if err != nil {
		t.Fatal(err)
	}
	if conflict == "" || n.Path != conflict {
		t.Fatalf("expected a conflict copy, got %q", conflict)
	}
	if readFile(t, path) != "theirs" || readFile(t, conflict) != "mine" {
		t.Error("both versions should be kept")
	}
	// Later saves go to the copy without another conflict.
	if c, err := n.Save("mine 2"); err != nil || c != "" || readFile(t, conflict) != "mine 2" {
		t.Errorf("second save: %q, %v", c, err)
	}
}

func TestAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "n.md")
	for _, s := range []string{"one", "two\n"} {
		if err := Append(path, s); err != nil {
			t.Fatal(err)
		}
	}
	if got := readFile(t, path); got != "one\ntwo\n" {
		t.Errorf("file = %q", got)
	}
	if err := os.WriteFile(path, []byte("no newline"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Append(path, "three"); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, path); got != "no newline\nthree\n" {
		t.Errorf("file = %q", got)
	}
}
