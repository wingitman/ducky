// Command ducky provides a terminal interface for Docker and Podman.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/wingitman/ducky/internal/app"
	"github.com/wingitman/ducky/internal/config"
	"github.com/wingitman/ducky/internal/runtime"
	"github.com/wingitman/ducky/internal/setup"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		if err := runSetup(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "ducky setup: %v\n", err)
			os.Exit(1)
		}
		return
	}

	flags := flag.NewFlagSet("ducky", flag.ExitOnError)
	runtimeName := flags.String("runtime", "", "runtime to use: docker or podman")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return
	}

	r, err := runtime.Detect(context.Background(), *runtimeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ducky: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ducky: config warning: %v\n", err)
		cfg = config.Default()
	}
	model := app.New(r, cfg)
	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ducky: %v\n", err)
		os.Exit(1)
	}
}

func runSetup(args []string) error {
	flags := flag.NewFlagSet("ducky setup", flag.ContinueOnError)
	target := flags.String("target", "", "target OS: linux or windows")
	mode := flags.String("mode", "", "image mode: development or production")
	force := flags.Bool("force", false, "overwrite an existing Dockerfile")
	if err := flags.Parse(args); err != nil {
		return err
	}

	options := setup.Options{Target: *target, Mode: *mode, Force: *force}
	result, err := setup.Run(context.Background(), os.Stdin, os.Stdout, options)
	if err != nil {
		return err
	}
	if result.Written {
		fmt.Fprintf(os.Stdout, "Generated %s for %s (%s).\n", result.Path, result.Project.Name, strings.ToLower(string(result.Project.Kind)))
	}
	return nil
}
