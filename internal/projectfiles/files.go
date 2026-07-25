// Package projectfiles discovers container and Kubernetes files below a project root.
package projectfiles

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/wingitman/ducky/internal/runtime"
)

var ignoredDirectories = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "bin": true,
	"obj": true, "target": true, "dist": true, "build": true,
}

// Discover recursively finds files that ducky can preview or run.
func Discover(root string) ([]runtime.Item, error) {
	var items []runtime.Item
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && ignoredDirectories[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		kind := fileKind(path)
		if kind == "" {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		items = append(items, runtime.Item{
			Name: relative, Path: path, Type: kind,
			Details: kind + " file",
		})
		return nil
	})
	return items, err
}

func fileKind(path string) string {
	base := filepath.Base(path)
	lower := strings.ToLower(base)
	switch {
	case base == "Dockerfile" || strings.HasPrefix(base, "Dockerfile.") || base == "Containerfile":
		return "dockerfile"
	case lower == "compose.yaml" || lower == "compose.yml" || lower == "docker-compose.yaml" || lower == "docker-compose.yml":
		return "compose"
	case lower == "kustomization.yaml" || lower == "kustomization.yml" || lower == "chart.yaml" || lower == "values.yaml":
		return "kubernetes"
	case strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml"):
		return kubernetesManifest(path)
	case lower == ".dockerignore":
		return "dockerignore"
	default:
		return ""
	}
}

func kubernetesManifest(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	apiVersion, kind := false, false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		apiVersion = apiVersion || strings.HasPrefix(line, "apiVersion:")
		kind = kind || strings.HasPrefix(line, "kind:")
		if apiVersion && kind {
			return "kubernetes"
		}
	}
	return ""
}
