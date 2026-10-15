package notes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// gitTimeout bounds each git call, so a slow filesystem never holds up the popup.
const gitTimeout = 2 * time.Second

// Project is the git repository a scratchpad belongs to.
type Project struct {
	// Name is shown in the header: owner/repo for a repository with a
	// remote, the directory name otherwise.
	Name string
	// Key is the note's path below the projects directory, without the
	// extension. It comes from the remote URL when there is one, so clones
	// of the same repository on different machines share the note.
	Key string
	// Root is the repository's top-level directory.
	Root string
}

// Detect returns the git repository containing dir. ok is false outside a
// repository or when git is not installed.
func Detect(dir string) (p Project, ok bool) {
	root, err := git(dir, "rev-parse", "--show-toplevel")
	if err != nil || root == "" {
		return Project{}, false
	}
	p.Root = root
	if url := remoteURL(dir); url != "" {
		if key, name := remoteKey(url); key != "" {
			p.Key, p.Name = key, name
			return p, true
		}
	}
	base := sanitize(filepath.Base(root))
	sum := sha256.Sum256([]byte(root))
	p.Key = "local/" + base + "-" + hex.EncodeToString(sum[:4])
	p.Name = filepath.Base(root)
	return p, true
}

// remoteURL returns the URL of the origin remote or, without one, of the
// first remote listed.
func remoteURL(dir string) string {
	if url, err := git(dir, "config", "--get", "remote.origin.url"); err == nil && url != "" {
		return url
	}
	remotes, err := git(dir, "remote")
	if err != nil || remotes == "" {
		return ""
	}
	first, _, _ := strings.Cut(remotes, "\n")
	url, _ := git(dir, "config", "--get", "remote."+first+".url")
	return url
}

func git(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// remoteKey turns a git remote URL into host/owner/repo, so that the
// https and ssh forms of the same remote give the same key. name is the
// URL's path (owner/repo). Local remotes (file paths) have no key.
func remoteKey(url string) (key, name string) {
	u := strings.TrimSpace(url)
	u = strings.TrimSuffix(u, "/")
	u = strings.TrimSuffix(u, ".git")

	var host, path string
	if scheme, rest, ok := strings.Cut(u, "://"); ok {
		if scheme == "file" {
			return "", ""
		}
		host, path, _ = strings.Cut(rest, "/")
	} else if h, p, ok := strings.Cut(u, ":"); ok && !strings.Contains(h, "/") {
		host, path = h, p // scp-like: git@host:owner/repo
	} else {
		return "", ""
	}
	if i := strings.LastIndex(host, "@"); i >= 0 {
		host = host[i+1:]
	}
	if i := strings.LastIndex(host, ":"); i >= 0 {
		host = host[:i] // port
	}
	host = sanitize(strings.ToLower(host))

	var segs []string
	for _, s := range strings.Split(path, "/") {
		if s == "" || s == "." || s == ".." {
			continue
		}
		segs = append(segs, sanitize(s))
	}
	if host == "" || len(segs) == 0 {
		return "", ""
	}
	return host + "/" + strings.Join(segs, "/"), strings.Join(segs, "/")
}

// sanitize keeps a path segment to characters that are safe in file names
// on every platform.
func sanitize(s string) string {
	b := []byte(s)
	for i, c := range b {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '-', c == '_':
		default:
			b[i] = '_'
		}
	}
	if s := strings.TrimLeft(string(b), "."); s != "" {
		return s
	}
	return "_"
}
