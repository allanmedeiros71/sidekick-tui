package notes

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoteKey(t *testing.T) {
	tests := []struct{ url, key, name string }{
		{"https://github.com/allan/sidekick-tui.git", "github.com/allan/sidekick-tui", "allan/sidekick-tui"},
		{"git@github.com:allan/sidekick-tui.git", "github.com/allan/sidekick-tui", "allan/sidekick-tui"},
		{"ssh://git@GitLab.example.com:2222/group/sub/app/", "gitlab.example.com/group/sub/app", "group/sub/app"},
		{"https://user:pw@host.io/a/b", "host.io/a/b", "a/b"},
		{"https://host.io/../a/./b c", "host.io/a/b_c", "a/b_c"},
		{"file:///srv/git/app.git", "", ""},
		{"/srv/git/app.git", "", ""},
		{"../app", "", ""},
		{"https://host.io/", "", ""},
	}
	for _, tt := range tests {
		key, name := remoteKey(tt.url)
		if key != tt.key || name != tt.name {
			t.Errorf("remoteKey(%q) = %q, %q; want %q, %q", tt.url, key, name, tt.key, tt.name)
		}
	}
}

func TestSanitize(t *testing.T) {
	for in, want := range map[string]string{"app": "app", "my app": "my_app", ".hidden": "hidden", "...": "_", "a/b": "a_b"} {
		if got := sanitize(in); got != want {
			t.Errorf("sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}

// gitRepo creates a repository in a temporary directory and returns its path.
func gitRepo(t *testing.T, remote string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	if remote != "" {
		run("remote", "add", "upstream", remote)
	}
	// Resolve symlinks (macOS /var -> /private/var) the way git reports them.
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestDetect(t *testing.T) {
	dir := gitRepo(t, "git@github.com:allan/sidekick-tui.git")
	sub := filepath.Join(dir, "a", "b")
	if err := exec.Command("mkdir", "-p", sub).Run(); err != nil {
		t.Fatal(err)
	}
	p, ok := Detect(sub)
	if !ok {
		t.Fatal("expected a repository")
	}
	if p.Key != "github.com/allan/sidekick-tui" || p.Name != "allan/sidekick-tui" || p.Root != dir {
		t.Errorf("Detect = %+v", p)
	}
}

func TestDetectWithoutRemote(t *testing.T) {
	dir := gitRepo(t, "")
	p, ok := Detect(dir)
	if !ok {
		t.Fatal("expected a repository")
	}
	if !strings.HasPrefix(p.Key, "local/"+sanitize(filepath.Base(dir))+"-") || p.Name != filepath.Base(dir) {
		t.Errorf("Detect = %+v", p)
	}
}

func TestDetectOutsideRepo(t *testing.T) {
	if _, ok := Detect(t.TempDir()); ok {
		t.Error("a plain directory is not a repository")
	}
}
