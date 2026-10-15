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

Release builds are published on the
[Releases page](https://github.com/allanmedeiros71/sidekick-tui/releases) for
Linux and macOS (amd64 and arm64). The binary is static, so packages have no
dependencies. In the commands below, replace `VERSION` with the release you
want (without the leading `v`) and `amd64` with `arm64` on ARM machines.

### macOS and Linux (Homebrew)

```sh
brew install --cask allanmedeiros71/tap/sidekick
```

### Debian, Ubuntu and derivatives

```sh
curl -LO https://github.com/allanmedeiros71/sidekick-tui/releases/download/vVERSION/sidekick_VERSION_amd64.deb
sudo apt install ./sidekick_VERSION_amd64.deb
```

### Fedora, RHEL and derivatives

```sh
sudo dnf install https://github.com/allanmedeiros71/sidekick-tui/releases/download/vVERSION/sidekick-VERSION-1.x86_64.rpm
```

On ARM, the file ends in `.aarch64.rpm`.

### Arch Linux and derivatives

From the AUR, with any helper:

```sh
yay -S sidekick-bin
```

Or install the package from the release directly:

```sh
sudo pacman -U https://github.com/allanmedeiros71/sidekick-tui/releases/download/vVERSION/sidekick-VERSION-1-x86_64.pkg.tar.zst
```

### Prebuilt binary

Download `sidekick_VERSION_<os>_<arch>.tar.gz`, extract it and put `sidekick`
somewhere on your `PATH`.

### From source (Go 1.24+)

```sh
go install github.com/allanmedeiros71/sidekick-tui/cmd/sidekick@latest
```

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
goreleaser release --snapshot --clean   # local cross-build and packages
```

### Releasing

Push a `vX.Y.Z` tag. The release workflow builds the archives, the `.deb`,
`.rpm` and Arch packages, and attaches them to a GitHub release. Two optional
repository secrets turn on the extra channels; without them those steps are
skipped and the release still succeeds:

| Secret                      | Publishes                                        |
| --------------------------- | ------------------------------------------------ |
| `HOMEBREW_TAP_GITHUB_TOKEN` | `Casks/sidekick.rb` to `allanmedeiros71/homebrew-tap` |
| `AUR_KEY`                   | the `sidekick-bin` package to the AUR            |

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
