BINARY := ducky
INSTALL_DIR := $(HOME)/.local/bin
BUILD_DIR := bin
RELEASES_DIR := releases
COMMIT := $(shell git rev-parse HEAD 2>/dev/null || printf dev)
LDFLAGS := -s -w -X github.com/wingitman/ducky/internal/version.Commit=$(COMMIT)

.PHONY: all build build-all install uninstall clean test test-all

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/ducky
	@echo "Built: $(BUILD_DIR)/$(BINARY)"

build-all:
	@mkdir -p $(RELEASES_DIR)/linux/amd64 $(RELEASES_DIR)/linux/arm64 $(RELEASES_DIR)/darwin/amd64 $(RELEASES_DIR)/darwin/arm64 $(RELEASES_DIR)/windows
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(RELEASES_DIR)/linux/amd64/$(BINARY) ./cmd/ducky
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(RELEASES_DIR)/linux/arm64/$(BINARY) ./cmd/ducky
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(RELEASES_DIR)/darwin/amd64/$(BINARY) ./cmd/ducky
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(RELEASES_DIR)/darwin/arm64/$(BINARY) ./cmd/ducky
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(RELEASES_DIR)/windows/$(BINARY).exe ./cmd/ducky
	@echo "Pre-built binaries written to $(RELEASES_DIR)/"

install:
	@mkdir -p $(INSTALL_DIR)
	@if command -v go >/dev/null 2>&1; then \
		echo "==> Go found - building ducky from source..."; \
		mkdir -p $(BUILD_DIR); \
		go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/ducky || exit 1; \
		cp $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY); \
	else \
		echo "==> Go not found - installing pre-built binary..."; \
		OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); ARCH=$$(uname -m); \
		case "$$ARCH" in x86_64|amd64) ARCH=amd64 ;; aarch64|arm64) ARCH=arm64 ;; *) echo "Unsupported architecture: $$ARCH"; exit 1 ;; esac; \
		if [ "$$OS" = "darwin" ] || [ "$$OS" = "linux" ]; then RELEASE_BIN="$(RELEASES_DIR)/$$OS/$$ARCH/$(BINARY)"; else echo "Unsupported OS: $$OS"; exit 1; fi; \
		 test -f "$$RELEASE_BIN" || { echo "Missing $$RELEASE_BIN; run make build-all first."; exit 1; }; \
		cp "$$RELEASE_BIN" $(INSTALL_DIR)/$(BINARY); chmod +x $(INSTALL_DIR)/$(BINARY); \
	fi
	@echo "Installed: $(INSTALL_DIR)/$(BINARY)"

uninstall:
	@rm -f $(INSTALL_DIR)/$(BINARY)
	@echo "Removed $(INSTALL_DIR)/$(BINARY)"

test:
	go test ./internal/... -timeout 30s

test-all: test
	go test ./... -timeout 30s

clean:
	rm -rf $(BUILD_DIR)
