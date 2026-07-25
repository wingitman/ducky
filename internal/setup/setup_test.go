package setup

import (
	"os"
	"path/filepath"
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
