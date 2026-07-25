package projectfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSkipsDependenciesAndFindsProjectFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Dockerfile"), []byte("FROM alpine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "deploy.yaml"), []byte("apiVersion: apps/v1\nkind: Deployment\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "Dockerfile"), []byte("FROM bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	items, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d files, want 2", len(items))
	}
}
