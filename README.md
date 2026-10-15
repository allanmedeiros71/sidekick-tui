# sidekick

A pop-up toolbox for developers, in the spirit of Borland SideKick (the 1984
MS-DOS TSR). Hit a shortcut, a panel appears on top of your work, you do the
quick thing, and it disappears without taking you out of context.

> **Status:** early skeleton. The popup opens and lists the planned modules;
> none of them is implemented yet.

## Planned modules

1. **Calculator**: hex/bin, bitwise, epoch, chmod, CIDR, sizes
2. **Inspector**: base64, URL, JWT, hashes, UUID, JSON
3. **Scratchpad**: per-project notes (detects the current git repo)
4. **Connections**: SSH, PostgreSQL, URLs; "dial" to connect
5. **Runbooks**: commands with placeholders (podman, psql, ...)
6. **Clipboard**: history and snippets
7. **GitLab**: merge requests and pipeline status

Extra tools can be added as plugins: external executables speaking JSON over
stdin/stdout, written in any language.

## Install

From source (Go 1.24+):

```sh
go install github.com/allanmedeiros71/sidekick-tui/cmd/sidekick@latest
```

Prebuilt binaries for Linux and macOS (amd64, arm64) will be published on the
Releases page.

## Usage

Inside tmux, bind it to a popup:

```tmux
bind-key k display-popup -E -w 80% -h 80% sidekick
```

Outside tmux, bind a global shortcut in your desktop environment that opens a
terminal running `sidekick` (e.g. `ghostty -e sidekick`).

The binary is called `sidekick`. It deliberately does not ship as `sk`, which
belongs to [skim](https://github.com/skim-rs/skim); pick your own short alias
if you want one (`alias kick=sidekick`).

## Configuration

sidekick uses XDG paths on every platform, macOS included, so the same
dotfiles work everywhere:

| What   | Path                                              |
| ------ | ------------------------------------------------- |
| Config | `$XDG_CONFIG_HOME/sidekick/config.toml` (default `~/.config/sidekick`) |
| Data   | `$XDG_DATA_HOME/sidekick` (default `~/.local/share/sidekick`)          |

```toml
# ~/.config/sidekick/config.toml
theme = "default"   # or "cga", a nod to the original

[sync]
backend = "none"    # "git", "folder" or "none" (sync is not implemented yet)
```

Data is stored one file per item (notes as `.md`, runbooks and connections as
`.toml`), so it diffs and syncs cleanly. Secrets are never stored: only
references such as `infisical://...` or `bw://...`, resolved at use time.

## Development

```sh
go build ./cmd/sidekick
go vet ./...
go test ./...
goreleaser release --snapshot --clean   # local cross-build
```

Layout:

```
cmd/sidekick/      entry point
internal/app/      root Bubble Tea model (popup frame and menu)
internal/config/   TOML config and XDG directories
internal/module/   module contract and roadmap
internal/theme/    Lip Gloss themes (default, CGA)
```

## License

MIT
