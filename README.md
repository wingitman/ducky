# ducky
for duplicating ducks... ofcourse...?

`ducky` is a terminal UI for Docker and Podman, built with Bubble Tea v2 and Lip Gloss v2.

It provides a compact view of containers, images, volumes, networks, Compose projects, and Kubernetes pods. Commands are executed through the installed runtime CLI, so Docker and Podman can be used without requiring a separate daemon SDK integration.

## Run

```sh
go run ./cmd/ducky
go run ./cmd/ducky --runtime podman
```

The runtime is detected automatically, preferring Docker and then Podman. Use `--runtime` to select one explicitly.

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
| `s` | Start or bring up the selected resource |
| `x` | Stop the selected resource |
| `d` | Remove, delete, or bring down the selected resource |
| `enter` | Inspect the selected resource |
| `g` | Show logs |
| `e` | Edit the Compose file |
| `o` | Open ducky configuration |
| `r` | Refresh |
| `q`, `esc` | Quit |

All bindings are configurable. The bottom hint bar only displays actions
available in the active view and uses the current values from the config file.

Inspect output and logs are displayed in a bounded, scrollable viewport. Use
`ctrl+u` and `ctrl+d`, or page keys where supported, to scroll output.

## Generate a Dockerfile

Run this from a project directory:

```sh
ducky setup
ducky setup --target linux --mode production
ducky setup --force
```

The setup command detects common Node.js, .NET, Go, Python, Rust, Java, and static web projects. It previews a generated Dockerfile and never overwrites an existing Dockerfile unless `--force` is supplied.

## Kubernetes

The Kubernetes view requires `kubectl` and uses the current kubeconfig context. Docker Desktop Kubernetes, kind, minikube, Podman-based clusters, and remote clusters are supported where the configured context exposes them.
