# Scratchpad

Open it from the menu, or directly with `sidekick notes`. Each git
repository gets its own markdown note, picked from the directory sidekick is
started in; outside a repository you get one global note. Just type: the note
saves itself half a second after you stop, and again when you leave it.

| Key      | Does                                                       |
| -------- | ---------------------------------------------------------- |
| `esc`    | save and go back to the menu                               |
| `ctrl+s` | save now                                                   |
| `ctrl+g` | switch between the project's note and the global one       |
| `ctrl+e` | open the note in `$VISUAL` or `$EDITOR` (default `vi`) and reload it afterwards |

To add a line without opening the UI, pass it on the command line or pipe it
in. It goes to the end of the current project's note (or the global one):

```sh
sidekick notes 'check the retry timeout in the worker'
git log -1 --format='%h %s' | sidekick notes
```

## Inside tmux

The project comes from the working directory, so give the popup the pane's
directory:

```tmux
bind-key n display-popup -E -d '#{pane_current_path}' -w 80% -h 80% sidekick notes
```

## Where notes live

Notes are plain markdown files below the data directory
(`~/.local/share/sidekick` unless `XDG_DATA_HOME` is set):

| Note                         | File                                            |
| ---------------------------- | ----------------------------------------------- |
| Global                       | `notes/scratch.md`                              |
| Repository with a remote     | `notes/projects/<host>/<owner>/<repo>.md`, e.g. `notes/projects/github.com/allanmedeiros71/sidekick-tui.md` |
| Repository without a remote  | `notes/projects/local/<dir>-<hash>.md`          |

The key comes from the `origin` remote (or the first remote when there is no
`origin`), with the https and ssh forms of the same URL treated alike. That
way every clone and worktree of a repository, on every machine, shares one
note once the data directory is synced. A repository without a remote is
keyed by its path instead, so it only matches itself.

Detecting the repository uses the `git` on your `PATH`; without it, the
global note is used.

## Safety

- Files are replaced atomically, so a crash or a sync never sees half a note.
- If the file changed on disk while you were editing it (a sync, another
  sidekick window, your editor), your version is saved next to it as
  `<name>.conflict-<date>-<time>.md` instead of overwriting it, and the
  scratchpad keeps editing that copy. Merge the two by hand.
- The built-in editor turns tabs into spaces. A note it cannot hold as is
  (control characters, more than 9000 lines) opens read-only; use `ctrl+e`
  to edit it in your own editor.
