package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectNodeProject(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	project, err := Detect(directory)
	if err != nil {
		t.Fatal(err)
	}
	if project.Kind != Node {
		t.Fatalf("project kind = %q, want %q", project.Kind, Node)
	}
}

func TestDetectDotnetProject(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "app.csproj"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	project, err := Detect(directory)
	if err != nil {
		t.Fatal(err)
	}
	if project.Kind != Dotnet {
		t.Fatalf("project kind = %q, want %q", project.Kind, Dotnet)
	}
}

func TestDetectsNestedMultiProjectRepository(t *testing.T) {
	directory := t.TempDir()
	if err := os.MkdirAll(filepath.Join(directory, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(directory, "frontend"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "api", "App.csproj"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "frontend", "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	project, err := Detect(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Projects) != 2 || project.Kind != Kind("multi-project") {
		t.Fatalf("project = %+v, want two-project detection", project)
	}
	content, err := template(project, "linux", "production")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "AS api") || !strings.Contains(content, "AS frontend") || !strings.Contains(content, "#") {
		t.Fatalf("multi-project Dockerfile missing stages/comments: %s", content)
	}
}
