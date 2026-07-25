// Package setup detects common projects and generates starter Dockerfiles.
package setup

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Kind identifies a detected project type.
type Kind string

const (
	Node   Kind = "Node.js"
	Dotnet Kind = ".NET"
	Go     Kind = "Go"
	Python Kind = "Python"
	Rust   Kind = "Rust"
	Java   Kind = "Java"
	Static Kind = "static web"
)

// Project describes a detected project.
type Project struct {
	Kind  Kind
	Name  string
	Files []string
}

// Options controls setup generation.
type Options struct {
	Target string
	Mode   string
	Force  bool
}

// Result describes the setup operation.
type Result struct {
	Project Project
	Path    string
	Written bool
}

// Run detects the current directory and interactively generates a Dockerfile.
func Run(ctx context.Context, in io.Reader, out io.Writer, options Options) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	directory, err := os.Getwd()
	if err != nil {
		return Result{}, fmt.Errorf("get current directory: %w", err)
	}
	project, err := Detect(directory)
	if err != nil {
		return Result{}, err
	}

	reader := bufio.NewReader(in)
	target := strings.ToLower(options.Target)
	if target == "" {
		target = prompt(reader, out, "Target OS [linux]", "linux")
	}
	mode := strings.ToLower(options.Mode)
	if mode == "" {
		mode = prompt(reader, out, "Image mode [production]", "production")
	}
	if target != "linux" && target != "windows" {
		return Result{}, errors.New("target must be linux or windows")
	}
	if mode != "development" && mode != "production" {
		return Result{}, errors.New("mode must be development or production")
	}

	path := filepath.Join(directory, "Dockerfile")
	if _, err := os.Stat(path); err == nil && !options.Force {
		return Result{}, fmt.Errorf("%s already exists; use --force to overwrite", path)
	}
	content, err := template(project, target, mode)
	if err != nil {
		return Result{}, err
	}
	fmt.Fprintf(out, "Detected %s project (%s).\n\n%s\n", project.Kind, strings.Join(project.Files, ", "), content)
	if prompt(reader, out, "Write Dockerfile? [Y/n]", "y") != "y" {
		return Result{Project: project, Path: path}, nil
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return Result{}, fmt.Errorf("write Dockerfile: %w", err)
	}
	return Result{Project: project, Path: path, Written: true}, nil
}

// Detect identifies a project from files in directory.
func Detect(directory string) (Project, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return Project{}, fmt.Errorf("read project directory: %w", err)
	}
	files := make(map[string]bool)
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files[entry.Name()] = true
		names = append(names, entry.Name())
	}
	project := Project{Name: filepath.Base(directory), Files: names}
	switch {
	case files["package.json"]:
		project.Kind = Node
	case hasSuffix(names, ".csproj") || hasSuffix(names, ".sln"):
		project.Kind = Dotnet
	case files["go.mod"]:
		project.Kind = Go
	case files["requirements.txt"] || files["pyproject.toml"] || files["Pipfile"]:
		project.Kind = Python
	case files["Cargo.toml"]:
		project.Kind = Rust
	case files["pom.xml"] || files["build.gradle"]:
		project.Kind = Java
	case files["index.html"]:
		project.Kind = Static
	default:
		return Project{}, errors.New("could not detect a supported project type")
	}
	return project, nil
}

func template(project Project, target, mode string) (string, error) {
	if target == "windows" {
		return windowsTemplate(project), nil
	}
	switch project.Kind {
	case Node:
		if mode == "development" {
			return "FROM node:22-bookworm-slim\nWORKDIR /app\nCOPY package*.json ./\nRUN npm install\nCOPY . .\nCMD [\"npm\", \"run\", \"dev\"]\n", nil
		}
		return "FROM node:22-bookworm-slim AS build\nWORKDIR /app\nCOPY package*.json ./\nRUN npm ci\nCOPY . .\nRUN npm run build\n\nFROM node:22-bookworm-slim\nWORKDIR /app\nCOPY --from=build /app ./\nCMD [\"npm\", \"start\"]\n", nil
	case Dotnet:
		return "FROM mcr.microsoft.com/dotnet/sdk:9.0 AS build\nWORKDIR /src\nCOPY . .\nRUN dotnet publish -c Release -o /out\n\nFROM mcr.microsoft.com/dotnet/aspnet:9.0\nWORKDIR /app\nCOPY --from=build /out .\nENTRYPOINT [\"dotnet\", \"APP.dll\"]\n", nil
	case Go:
		return "FROM golang:1.24-bookworm AS build\nWORKDIR /src\nCOPY go.* ./\nRUN go mod download\nCOPY . .\nRUN CGO_ENABLED=0 go build -o /out/app .\n\nFROM gcr.io/distroless/static-debian12\nCOPY --from=build /out/app /app\nENTRYPOINT [\"/app\"]\n", nil
	case Python:
		return "FROM python:3.13-slim\nWORKDIR /app\nCOPY requirements.txt* pyproject.toml* ./\nRUN if [ -f requirements.txt ]; then pip install --no-cache-dir -r requirements.txt; fi\nCOPY . .\nCMD [\"python\", \"-m\", \"app\"]\n", nil
	case Rust:
		return "FROM rust:1-bookworm AS build\nWORKDIR /src\nCOPY . .\nRUN cargo build --release\n\nFROM debian:bookworm-slim\nCOPY --from=build /src/target/release/app /usr/local/bin/app\nENTRYPOINT [\"app\"]\n", nil
	case Java:
		return "FROM eclipse-temurin:21-jdk AS build\nWORKDIR /src\nCOPY . .\nRUN ./mvnw package -DskipTests\n\nFROM eclipse-temurin:21-jre\nCOPY --from=build /src/target/*.jar /app.jar\nENTRYPOINT [\"java\", \"-jar\", \"/app.jar\"]\n", nil
	case Static:
		return "FROM nginx:alpine\nCOPY . /usr/share/nginx/html\n", nil
	default:
		return "", fmt.Errorf("no template for %s", project.Kind)
	}
}

func windowsTemplate(project Project) string {
	if project.Kind == Dotnet {
		return "FROM mcr.microsoft.com/dotnet/sdk:9.0-windowsservercore-ltsc2022 AS build\nWORKDIR C:/src\nCOPY . .\nRUN dotnet publish -c Release -o C:/out\n\nFROM mcr.microsoft.com/dotnet/aspnet:9.0-windowsservercore-ltsc2022\nWORKDIR C:/app\nCOPY --from=build C:/out .\nENTRYPOINT [\"dotnet\", \"APP.dll\"]\n"
	}
	return fmt.Sprintf("# Windows base image selected for %s.\n# Add the runtime-specific build steps for this project.\nFROM mcr.microsoft.com/windows/servercore:ltsc2022\nWORKDIR C:/app\nCOPY . .\n", project.Kind)
}

func prompt(reader *bufio.Reader, out io.Writer, label, fallback string) string {
	fmt.Fprintf(out, "%s: ", label)
	answer, err := reader.ReadString('\n')
	if err != nil && len(answer) == 0 {
		return fallback
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "" {
		return fallback
	}
	return answer
}

func hasSuffix(names []string, suffix string) bool {
	for _, name := range names {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}
