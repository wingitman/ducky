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
	Kind     Kind
	Name     string
	Root     string
	Files    []string
	Projects []Subproject
}

// Subproject describes one runnable project found below the root.
type Subproject struct {
	Kind     Kind
	Name     string
	Dir      string
	Manifest string
}

// Options controls setup generation.
type Options struct {
	Target string
	Mode   string
	Force  bool
}

// Result describes the setup operation.
type Result struct {
	Project     Project
	Path        string
	ComposePath string
	IgnorePath  string
	Written     bool
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
	composePath := ""
	if len(project.Projects) > 1 {
		composePath = filepath.Join(directory, "compose.yaml")
		if _, statErr := os.Stat(composePath); statErr == nil && !options.Force {
			return Result{}, fmt.Errorf("%s already exists; use --force to overwrite", composePath)
		}
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return Result{}, fmt.Errorf("write Dockerfile: %w", err)
	}
	ignorePath := filepath.Join(directory, ".dockerignore")
	if _, statErr := os.Stat(ignorePath); statErr != nil || options.Force {
		if err := os.WriteFile(ignorePath, []byte(dockerignore), 0o644); err != nil {
			return Result{}, fmt.Errorf("write .dockerignore: %w", err)
		}
	}
	result := Result{Project: project, Path: path, IgnorePath: ignorePath, Written: true}
	if len(project.Projects) > 1 {
		compose, composeErr := multiCompose(project)
		if composeErr != nil {
			return Result{}, composeErr
		}
		if err := os.WriteFile(composePath, []byte(compose), 0o644); err != nil {
			return Result{}, fmt.Errorf("write compose.yaml: %w", err)
		}
		result.ComposePath = composePath
	}
	return result, nil
}

const dockerignore = `# Keep source-control metadata and local tooling out of Docker build contexts.
.git
.gitignore
.idea
.vscode

# Exclude generated and dependency-heavy directories.
**/bin
**/obj
**/node_modules
**/dist
**/build
**/target
coverage
*.log
`

// Detect identifies a project from files in directory.
func Detect(directory string) (Project, error) {
	project := Project{Name: filepath.Base(directory), Root: directory}
	err := filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != directory && ignoredSetupDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		relative, relErr := filepath.Rel(directory, path)
		if relErr != nil {
			return relErr
		}
		kind := manifestKind(filepath.Base(path))
		if kind == "" {
			return nil
		}
		project.Files = append(project.Files, relative)
		dir := filepath.Dir(relative)
		if dir == "." {
			dir = ""
		}
		project.Projects = append(project.Projects, Subproject{Kind: kind, Name: filepath.Base(strings.TrimSuffix(path, filepath.Ext(path))), Dir: dir, Manifest: relative})
		return nil
	})
	if err != nil {
		return Project{}, fmt.Errorf("read project directory: %w", err)
	}
	if len(project.Projects) == 0 {
		return Project{}, errors.New("could not detect a supported project type")
	}
	project.Kind = project.Projects[0].Kind
	if len(project.Projects) > 1 {
		project.Kind = Kind("multi-project")
	}
	return project, nil
}

var ignoredSetupDirs = map[string]bool{".git": true, "node_modules": true, "vendor": true, "bin": true, "obj": true, "target": true, "dist": true, "build": true}

func ignoredSetupDirectory(name string) bool { return ignoredSetupDirs[name] }

func manifestKind(name string) Kind {
	switch strings.ToLower(name) {
	case "package.json":
		return Node
	case "requirements.txt", "pyproject.toml", "pipfile":
		return Python
	case "cargo.toml":
		return Rust
	case "pom.xml", "build.gradle":
		return Java
	case "go.mod":
		return Go
	}
	if strings.HasSuffix(strings.ToLower(name), ".csproj") {
		return Dotnet
	}
	return ""
}

func template(project Project, target, mode string) (string, error) {
	if len(project.Projects) > 1 {
		return multiDockerfile(project, target, mode)
	}
	if target == "windows" {
		return windowsTemplate(project), nil
	}
	switch project.Kind {
	case Node:
		if mode == "development" {
			return `# Use a small Debian-based Node image for development.
FROM node:22-bookworm-slim
# Keep the source in a predictable working directory.
WORKDIR /app
# Copy dependency manifests first so npm installation can be cached.
COPY package*.json ./
# Install the project's JavaScript dependencies.
RUN npm install
# Copy the application source after dependencies for better build caching.
COPY . .
# Start the development server when the container launches.
CMD ["npm", "run", "dev"]
`, nil
		}
		return `# Use Node to compile the production frontend.
FROM node:22-bookworm-slim AS build
# Set the working directory for the build stage.
WORKDIR /app
# Copy lockfiles before source files so dependency installation is cached.
COPY package*.json ./
# Install exactly the versions recorded by the lockfile.
RUN npm ci
# Copy the application source into the build stage.
COPY . .
# Produce the optimized production assets.
RUN npm run build

# Use a clean Node image for the production process.
FROM node:22-bookworm-slim
# Set the runtime working directory.
WORKDIR /app
# Copy the compiled application from the build stage.
COPY --from=build /app ./
# Start the production application.
CMD ["npm", "start"]
`, nil
	case Dotnet:
		return `# Use the .NET SDK because this stage compiles the application.
FROM mcr.microsoft.com/dotnet/sdk:10.0 AS build
# Keep source code in a dedicated build directory.
WORKDIR /src
# Copy the repository so project references and solution files are available.
COPY . .
# Publish the application and its dependencies into a clean output directory.
RUN dotnet publish -c Release -o /out

# Use the smaller ASP.NET runtime image for the final container.
FROM mcr.microsoft.com/dotnet/aspnet:10.0
# Set the runtime working directory.
WORKDIR /app
# Copy only published output from the SDK stage.
COPY --from=build /out .
# Replace APP.dll with the output assembly for the selected project.
ENTRYPOINT ["dotnet", "APP.dll"]
`, nil
	case Go:
		return `# Use Go to compile a statically linked binary.
FROM golang:1.24-bookworm AS build
# Set the Go build directory.
WORKDIR /src
# Copy module files first so dependency downloads are cached.
COPY go.* ./
# Download Go dependencies.
RUN go mod download
# Copy the remaining source files.
COPY . .
# Build a binary without a libc dependency.
RUN CGO_ENABLED=0 go build -o /out/app .

# Use a minimal image containing only the compiled binary.
FROM gcr.io/distroless/static-debian12
# Copy the binary from the build stage.
COPY --from=build /out/app /app
# Run the compiled application.
ENTRYPOINT ["/app"]
`, nil
	case Python:
		return `# Use a slim Python runtime image.
FROM python:3.13-slim
# Set the application working directory.
WORKDIR /app
# Copy dependency declarations before source files.
COPY requirements.txt* pyproject.toml* ./
# Install dependencies only when a requirements file exists.
RUN if [ -f requirements.txt ]; then pip install --no-cache-dir -r requirements.txt; fi
# Copy the application source.
COPY . .
# Start the default Python module.
CMD ["python", "-m", "app"]
`, nil
	case Rust:
		return `# Use Rust to compile the release binary.
FROM rust:1-bookworm AS build
# Set the Rust build directory.
WORKDIR /src
# Copy the project source and manifest.
COPY . .
# Compile an optimized release binary.
RUN cargo build --release

# Use a small Debian runtime image.
FROM debian:bookworm-slim
# Copy the compiled binary into the runtime image.
COPY --from=build /src/target/release/app /usr/local/bin/app
# Run the application binary.
ENTRYPOINT ["app"]
`, nil
	case Java:
		return `# Use a JDK image to compile the application.
FROM eclipse-temurin:21-jdk AS build
# Set the Java build directory.
WORKDIR /src
# Copy project source and build configuration.
COPY . .
# Build the application without running tests in the image build.
RUN ./mvnw package -DskipTests

# Use a smaller JRE image for runtime.
FROM eclipse-temurin:21-jre
# Copy the built JAR into a stable location.
COPY --from=build /src/target/*.jar /app.jar
# Start the Java application.
ENTRYPOINT ["java", "-jar", "/app.jar"]
`, nil
	case Static:
		return `# Use Nginx to serve static web assets.
FROM nginx:alpine
# Copy the static site into Nginx's public directory.
COPY . /usr/share/nginx/html
`, nil
	default:
		return "", fmt.Errorf("no template for %s", project.Kind)
	}
}

func windowsTemplate(project Project) string {
	if project.Kind == Dotnet {
		return `# Use the Windows .NET SDK to compile the application.
FROM mcr.microsoft.com/dotnet/sdk:10.0-windowsservercore-ltsc2022 AS build
# Set the source directory.
WORKDIR C:/src
# Copy source and project references.
COPY . .
# Publish the application into a clean output directory.
RUN dotnet publish -c Release -o C:/out

# Use the Windows ASP.NET runtime image.
FROM mcr.microsoft.com/dotnet/aspnet:10.0-windowsservercore-ltsc2022
# Set the runtime working directory.
WORKDIR C:/app
# Copy published output from the build stage.
COPY --from=build C:/out .
# Replace APP.dll with the selected output assembly.
ENTRYPOINT ["dotnet", "APP.dll"]
`
	}
	return fmt.Sprintf("# Windows base image selected for %s.\n# Copy the project into the Windows container.\nFROM mcr.microsoft.com/windows/servercore:ltsc2022\n# Set the application directory.\nWORKDIR C:/app\n# Copy project files into the image.\nCOPY . .\n", project.Kind)
}

func multiDockerfile(project Project, target, mode string) (string, error) {
	if target == "windows" {
		return windowsTemplate(project), nil
	}
	var dotnetProject, nodeProject *Subproject
	for i := range project.Projects {
		candidate := &project.Projects[i]
		switch candidate.Kind {
		case Dotnet:
			if dotnetProject == nil || strings.Contains(strings.ToLower(candidate.Name), "api") {
				dotnetProject = candidate
			}
		case Node:
			if nodeProject == nil {
				nodeProject = candidate
			}
		}
	}
	if dotnetProject == nil || nodeProject == nil {
		return "", fmt.Errorf("multi-project setup currently requires a .NET project and a Node.js project")
	}
	apiPath := filepath.ToSlash(dotnetProject.Manifest)
	frontendDir := filepath.ToSlash(dotnetProject.Dir)
	_ = frontendDir
	if nodeProject.Dir == "" {
		nodeProject.Dir = "."
	}
	frontendDir = filepath.ToSlash(nodeProject.Dir)
	frontendPackage := filepath.ToSlash(filepath.Join(nodeProject.Dir, "package*.json"))
	frontendSource := filepath.ToSlash(nodeProject.Dir)
	apiAssembly := dotnetProject.Name + ".dll"
	commentMode := "# Production targets build optimized API and frontend artifacts."
	if mode == "development" {
		commentMode = "# Development mode keeps the same build stages; Compose can override commands for live reload."
	}
	return fmt.Sprintf(`# Multi-project Dockerfile generated by ducky.
%s

# Use the .NET SDK to build the API and its referenced projects.
FROM mcr.microsoft.com/dotnet/sdk:10.0 AS api-build
# Put the repository in a stable build directory.
WORKDIR /src
# Copy the complete repository so solution and project references resolve.
COPY . .
# Publish the selected API project and its dependencies.
RUN dotnet publish %s -c Release -o /out/api

# Use only the ASP.NET runtime for the API service.
FROM mcr.microsoft.com/dotnet/aspnet:10.0 AS api
# Set the API runtime directory.
WORKDIR /app
# Copy published API output from the build stage.
COPY --from=api-build /out/api .
# Start the API process.
ENTRYPOINT ["dotnet", "%s"]

# Use Node.js to compile the frontend.
FROM node:22-bookworm-slim AS frontend-build
# Set the frontend build directory.
WORKDIR /src
# Copy package manifests first so dependency installation can be cached.
COPY %s ./%s/
# Install frontend dependencies.
RUN npm ci --prefix ./%s
# Copy frontend source files.
COPY %s ./%s/
# Build optimized frontend assets.
RUN npm run build --prefix ./%s

# Use Nginx to serve the compiled frontend.
FROM nginx:alpine AS frontend
# Copy the frontend build output into Nginx's public directory.
COPY --from=frontend-build /src/%s/dist /usr/share/nginx/html
`, commentMode, apiPath, apiAssembly, frontendPackage, frontendDir, frontendDir, frontendSource, frontendDir, frontendDir, frontendDir), nil
}

func multiCompose(project Project) (string, error) {
	var api, frontend *Subproject
	for i := range project.Projects {
		candidate := &project.Projects[i]
		if candidate.Kind == Dotnet && (api == nil || strings.Contains(strings.ToLower(candidate.Name), "api")) {
			api = candidate
		}
		if candidate.Kind == Node && frontend == nil {
			frontend = candidate
		}
	}
	if api == nil || frontend == nil {
		return "", errors.New("could not create Compose services for multi-project setup")
	}
	return `# Compose file generated by ducky for the detected API and frontend.
services:
  api:
    # Build the API target from the generated multi-stage Dockerfile.
    build:
      context: .
      dockerfile: Dockerfile
      target: api
    # Expose the API on the host for local development and testing.
    ports:
      - "5110:8080"
  frontend:
    # Build the frontend target from the same generated Dockerfile.
    build:
      context: .
      dockerfile: Dockerfile
      target: frontend
    # Expose the Nginx frontend on the host.
    ports:
      - "3000:80"
    # Start the API before the frontend service.
    depends_on:
      - api
`, nil
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
