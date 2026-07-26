# ducky
for duplicating ducks... ofcourse...?

`ducky` is a terminal UI for Docker and Podman, built with Bubble Tea v2 and Lip Gloss v2.

It provides a compact view of containers, images, volumes, networks, Compose projects, Kubernetes pods, and recursively discovered project files. Commands are executed through the installed runtime CLI, so Docker and Podman can be used without requiring a separate daemon SDK integration.

## Run

```sh
go run ./cmd/ducky
go run ./cmd/ducky --runtime podman
go install github.com/wingitman/ducky/cmd/ducky@latest
```

The runtime is detected automatically, preferring Docker and then Podman. Use `--runtime` to select one explicitly.

## Install and Releases

Build and install locally:

```sh
make install
```

Build Linux, macOS, and Windows release binaries:

```sh
make build-all
```

The same release build can be requested from a ducky binary launched inside
the source checkout:

```sh
ducky -build-all
```

Windows users can use the matching PowerShell script:

```powershell
.\install.ps1
.\install.ps1 -BuildAll
```

Release builds inject the Git commit into the binary. ducky checks the
configured GitHub repository asynchronously at startup and reports an update
in the status line when one is available. Network or GitHub failures never
prevent the TUI from launching. Set `disable_checks = true` under `[updates]`
to disable the check.

## Configuration

On first launch ducky creates `~/.config/delbysoft/ducky.toml` on Linux. The
platform-specific config directory is selected with `os.UserConfigDir()`.
Press `o` at any time to open the config in `$EDITOR` (then `$VISUAL`, then
`nano`). The file is reloaded when the editor closes, so changed bindings and
the contextual hint bar take effect immediately.

## Keys

| Key | Action |
| --- | --- |
| `tab`, `shift+tab` | Change view |
| `up`, `down` | Move through resources |
| `n` | Create, pull, run, or apply for the current view |
| `S` | Open the in-TUI project setup wizard |
| `s` | Start or bring up the selected resource |
| `x` | Stop the selected resource |
| `d` | Remove, delete, or bring down the selected resource |
| `enter` | Inspect the selected resource |
| `g` | Show logs |
| `/` | Filter the current view live |
| `E` | Open the current inspect/log preview in the editor |
| `e` | Edit the Compose file |
| `R` | Run the selected project file |
| `o` | Open ducky configuration |
| `r` | Refresh |
| `q`, `esc` | Quit |

All bindings are configurable. The bottom hint bar only displays actions
available in the active view and uses the current values from the config file.

Inspect output and logs are displayed in a bounded, scrollable viewport. Use
the configured up/down or page bindings to scroll output while the preview is
focused. The literal arrow keys are ignored unless they are mapped in the
config. Press `E` to open the preview in the default editor.

Search is live and supports free text plus field filters such as
`status:running`, `image:postgres`, `name:db`, and `port:5432`. Press Enter to
keep the filter or Escape to restore the previous filter.

When an inspect or log preview is open, the configured up/down and page keys
scroll only the preview. They do not move the resource selection. Unmapped
literal arrow or Vim keys are ignored.

## Generate a Dockerfile

Run this from a project directory:

```sh
ducky setup
ducky setup --target linux --mode production
ducky setup --force
```

The setup command detects common Node.js, .NET, Go, Python, Rust, Java, and static web projects. It previews a generated Dockerfile and never overwrites an existing Dockerfile unless `--force` is supplied.

Setup scans nested project directories and recognizes multi-project repositories.
When it finds multiple runnable services, it generates a commented multi-stage
`Dockerfile` plus a commented `compose.yaml` with separate services. This is
used for repositories such as a .NET API plus a Node/React frontend.

The `Files` tab recursively discovers Dockerfiles, Containerfiles, Compose
files, Docker ignore files, Kubernetes manifests, Kustomize files, and Helm
files. Press Enter to preview a file, `E` to edit it, and `R` to run it when
the file type supports execution.

## Kubernetes

The Kubernetes view requires `kubectl` and uses the current kubeconfig context. Docker Desktop Kubernetes, kind, minikube, Podman-based clusters, and remote clusters are supported where the configured context exposes them.

## Support
<a href='https://ko-fi.com/W7W21WP5L7' target='_blank'><img height='36' style='border:0px;height:36px;' src='https://storage.ko-fi.com/cdn/kofi4.png?v=6' border='0' alt='Buy Me a Coffee at ko-fi.com' /></a>

## License

MIT — see [LICENSE](LICENSE).

Copyright (c) 2026 [delbysoft](https://github.com/wingitman)
