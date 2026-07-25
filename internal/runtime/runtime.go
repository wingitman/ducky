// Package runtime adapts common Docker and Podman CLI operations for ducky.
package runtime

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Kind identifies the container runtime.
type Kind string

const (
	Docker Kind = "docker"
	Podman Kind = "podman"
)

// Resource identifies a top-level ducky view.
type Resource int

const (
	Containers Resource = iota
	Images
	Volumes
	Networks
	Compose
	Kubernetes
)

// Item is one row in a resource view.
type Item struct {
	Name    string
	Details string
}

// Client runs commands for one container runtime.
type Client struct {
	Kind Kind
	Bin  string
}

// Create performs the common create/pull/apply operation for a resource.
func (c Client) Create(ctx context.Context, resource Resource, value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("a value is required")
	}
	switch resource {
	case Containers:
		return c.Run(ctx, "run", "-d", value)
	case Images:
		return c.Run(ctx, "pull", value)
	case Volumes:
		return c.Run(ctx, "volume", "create", value)
	case Networks:
		return c.Run(ctx, "network", "create", value)
	case Kubernetes:
		return runKubectl(ctx, "apply", "-f", value)
	default:
		return "", fmt.Errorf("create is not supported for resource %d", resource)
	}
}

// Detect finds an explicitly requested or locally installed runtime.
func Detect(ctx context.Context, requested string) (Client, error) {
	if requested != "" {
		kind := Kind(strings.ToLower(requested))
		if kind != Docker && kind != Podman {
			return Client{}, fmt.Errorf("unknown runtime %q", requested)
		}
		if _, err := exec.LookPath(string(kind)); err != nil {
			return Client{}, fmt.Errorf("%s is not installed: %w", kind, err)
		}
		return Client{Kind: kind, Bin: string(kind)}, nil
	}

	for _, kind := range []Kind{Docker, Podman} {
		if _, err := exec.LookPath(string(kind)); err == nil {
			client := Client{Kind: kind, Bin: string(kind)}
			if _, err := client.Run(ctx, "info"); err == nil {
				return client, nil
			}
		}
	}
	return Client{}, errors.New("no running Docker or Podman runtime found")
}

// Run executes a runtime command and returns combined output.
func (c Client) Run(ctx context.Context, args ...string) (string, error) {
	command := exec.CommandContext(ctx, c.Bin, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", fmt.Errorf("%s %s: %w", c.Bin, strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("%s", message)
	}
	return string(output), nil
}

// List returns rows for a top-level resource.
func (c Client) List(ctx context.Context, resource Resource) ([]Item, error) {
	var args []string
	switch resource {
	case Containers:
		args = []string{"ps", "-a", "--format", "{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}"}
	case Images:
		args = []string{"images", "--format", "{{.Repository}}:{{.Tag}}\t{{.ID}}\t{{.Size}}"}
	case Volumes:
		args = []string{"volume", "ls", "--format", "{{.Name}}\t{{.Driver}}"}
	case Networks:
		args = []string{"network", "ls", "--format", "{{.Name}}\t{{.Driver}}"}
	case Compose:
		args = []string{"compose", "ls", "--all", "--format", "table {{.Name}}\t{{.Status}}\t{{.ConfigFiles}}"}
	case Kubernetes:
		return listKubernetes(ctx)
	default:
		return nil, fmt.Errorf("unsupported resource %d", resource)
	}
	output, err := c.Run(ctx, args...)
	if err != nil {
		return nil, err
	}
	return parseItems(string(output)), nil
}

// Action performs a safe, common action against the selected resource.
func (c Client) Action(ctx context.Context, resource Resource, action string, name string) (string, error) {
	if resource == Kubernetes {
		return kubernetesAction(ctx, action, name)
	}
	if resource == Compose {
		switch action {
		case "inspect":
			return c.Run(ctx, "compose", "config")
		case "start":
			return c.Run(ctx, "compose", "up", "-d", name)
		case "stop":
			return c.Run(ctx, "compose", "stop", name)
		case "remove":
			return c.Run(ctx, "compose", "down", name)
		default:
			return "", fmt.Errorf("%s is not supported for Compose projects", action)
		}
	}
	switch resource {
	case Containers:
		switch action {
		case "inspect":
			return c.Run(ctx, "inspect", name)
		case "start", "stop":
			return c.Run(ctx, action, name)
		case "remove":
			return c.Run(ctx, "rm", name)
		default:
			return "", fmt.Errorf("%s is not supported for containers", action)
		}
	case Images:
		if action == "inspect" {
			return c.Run(ctx, "image", "inspect", name)
		}
		return "", fmt.Errorf("%s is not supported for images", action)
	case Volumes:
		if action == "inspect" {
			return c.Run(ctx, "volume", "inspect", name)
		}
		if action != "remove" {
			return "", fmt.Errorf("%s is not supported for volumes", action)
		}
		return c.Run(ctx, "volume", "rm", name)
	case Networks:
		if action == "inspect" {
			return c.Run(ctx, "network", "inspect", name)
		}
		if action != "remove" {
			return "", fmt.Errorf("%s is not supported for networks", action)
		}
		return c.Run(ctx, "network", "rm", name)
	default:
		return "", fmt.Errorf("unsupported resource %d", resource)
	}
}

// Logs returns recent logs for a container or pod.
func (c Client) Logs(ctx context.Context, resource Resource, name string) (string, error) {
	if resource == Kubernetes {
		return kubernetesLogs(ctx, name)
	}
	return c.Run(ctx, "logs", "--tail", "100", name)
}

func parseItems(output string) []Item {
	var items []Item
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "NAME") {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) == 1 {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				parts = []string{fields[0], strings.Join(fields[1:], " ")}
			}
		}
		item := Item{Name: parts[0]}
		if len(parts) == 2 {
			item.Details = parts[1]
		}
		items = append(items, item)
	}
	return items
}

func parseKubernetesItems(output string) []Item {
	var items []Item
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		items = append(items, Item{
			Name:    fields[0] + "/" + fields[1],
			Details: strings.Join(fields[2:], " "),
		})
	}
	return items
}

func listKubernetes(ctx context.Context) ([]Item, error) {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return nil, errors.New("kubectl is not installed")
	}
	output, err := exec.CommandContext(ctx, "kubectl", "get", "pods", "-A", "--no-headers", "-o", "custom-columns=:.metadata.namespace,:.metadata.name,:.status.phase").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("kubectl: %s", strings.TrimSpace(string(output)))
	}
	return parseKubernetesItems(string(output)), nil
}

func kubernetesAction(ctx context.Context, action, name string) (string, error) {
	if _, err := exec.LookPath("kubectl"); err != nil {
		return "", errors.New("kubectl is not installed")
	}
	namespace, pod := kubernetesName(name)
	if action == "inspect" {
		return runKubectl(ctx, "get", "pod", pod, "-n", namespace, "-o", "yaml")
	}
	verb := action
	if action == "remove" {
		verb = "delete"
	}
	return runKubectl(ctx, verb, "pod", pod, "-n", namespace)
}

func kubernetesLogs(ctx context.Context, name string) (string, error) {
	namespace, pod := kubernetesName(name)
	return runKubectl(ctx, "logs", "--tail=100", pod, "-n", namespace)
}

func kubernetesName(name string) (string, string) {
	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 {
		return "default", name
	}
	return parts[0], parts[1]
}

func runKubectl(ctx context.Context, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "kubectl", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("kubectl: %s", strings.TrimSpace(string(output)))
	}
	return string(output), nil
}
