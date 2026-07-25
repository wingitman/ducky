// Package buildall cross-compiles ducky release binaries.
package buildall

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type target struct {
	os   string
	arch string
	ext  string
}

var targets = []target{
	{os: "linux", arch: "amd64", ext: ""},
	{os: "linux", arch: "arm64", ext: ""},
	{os: "darwin", arch: "amd64", ext: ""},
	{os: "darwin", arch: "arm64", ext: ""},
	{os: "windows", arch: "amd64", ext: ".exe"},
}

// Run builds all supported release targets from sourceDir.
func Run(ctx context.Context, sourceDir string, out io.Writer) error {
	if _, err := os.Stat(filepath.Join(sourceDir, "go.mod")); err != nil {
		return errors.New("-build-all must be run from the ducky source directory")
	}
	commit := gitCommit(ctx, sourceDir)
	for _, target := range targets {
		output := filepath.Join(sourceDir, "releases", target.os, target.arch, "ducky"+target.ext)
		if target.os == "windows" {
			output = filepath.Join(sourceDir, "releases", target.os, "ducky"+target.ext)
		}
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return fmt.Errorf("create release directory: %w", err)
		}
		fmt.Fprintf(out, "Building %s/%s...\n", target.os, target.arch)
		command := exec.CommandContext(ctx, "go", "build", "-ldflags", "-s -w -X github.com/wingitman/ducky/internal/version.Commit="+commit, "-o", output, "./cmd/ducky")
		command.Dir = sourceDir
		command.Env = append(os.Environ(), "GOOS="+target.os, "GOARCH="+target.arch, "CGO_ENABLED=0")
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("build %s/%s: %s: %w", target.os, target.arch, strings.TrimSpace(string(output)), err)
		}
	}
	fmt.Fprintln(out, "Release binaries written to releases/")
	return nil
}

func gitCommit(ctx context.Context, sourceDir string) string {
	command := exec.CommandContext(ctx, "git", "-C", sourceDir, "rev-parse", "HEAD")
	output, err := command.Output()
	if err != nil || strings.TrimSpace(string(output)) == "" {
		return "dev"
	}
	return strings.TrimSpace(string(output))
}
