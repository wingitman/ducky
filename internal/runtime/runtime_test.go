package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseItems(t *testing.T) {
	items := parseItems("web\tnginx\trunning\napi\tgo\tstopped\n")
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Name != "web" || items[0].Details != "nginx\trunning" {
		t.Fatalf("first item = %+v", items[0])
	}
}

func TestParseKubernetesStyleItems(t *testing.T) {
	items := parseItems("default web Running\nkube-system dns Pending\n")
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Name != "default" || items[0].Details != "web Running" {
		t.Fatalf("first item = %+v", items[0])
	}
}

func TestParseContainerItems(t *testing.T) {
	items := parseContainerItems("running\tweb\tnginx:alpine\t80/tcp\tAbout an hour\tbridge\nexited\tdb\tpostgres:16\t5432/tcp\t2 weeks ago\tbridge\n")
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Status != "running" || items[0].Name != "web" || items[0].Image != "nginx:alpine" || items[0].Ports != "80/tcp" {
		t.Fatalf("first container = %+v", items[0])
	}
}

func TestRunFileComposeUsesSelectedFile(t *testing.T) {
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args")
	bin := filepath.Join(dir, "runtime")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + argsPath + "\"\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(dir, "compose.yaml")
	output, err := (Client{Bin: bin}).RunFile(context.Background(), Item{Type: "compose", Path: file})
	if err != nil {
		t.Fatalf("RunFile returned error: %v", err)
	}
	if output != "" {
		t.Fatalf("RunFile output = %q, want empty script output", output)
	}
	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "compose\n-f\n" + file + "\nup\n-d\n"
	if strings.TrimSpace(string(args)) != strings.TrimSpace(want) {
		t.Fatalf("runtime args = %q, want %q", string(args), want)
	}
}

func TestRunFileRejectsNonRunnableFile(t *testing.T) {
	_, err := (Client{}).RunFile(context.Background(), Item{Type: "dockerignore", Path: ".dockerignore"})
	if err == nil || !strings.Contains(err.Error(), "cannot be run") {
		t.Fatalf("RunFile error = %v", err)
	}
}

func TestStartFileStreamsOutput(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "runtime")
	script := "#!/bin/sh\nprintf 'building\\n'\nprintf 'finished\\n'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	events, err := (Client{Bin: bin}).StartFile(context.Background(), Item{Type: "compose", Path: filepath.Join(dir, "compose.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	var output []string
	var completed bool
	for event := range events {
		if event.Output != "" {
			output = append(output, event.Output)
		}
		completed = completed || event.Done
	}
	if !completed || strings.Join(output, "\n") != "building\nfinished" {
		t.Fatalf("streamed events = %q, completed = %v", strings.Join(output, "\n"), completed)
	}
}
