.PHONY: build build-macos build-linux dev test test-v lint clean

GO := go

# Build for the current platform (delegates to Wails)
build:
	wails build

# Build for macOS (arm64 + amd64 universal binary)
# Wails produces a .app bundle under build/bin/
build-macos:
	wails build -platform darwin/universal

# Build for Linux amd64
# Requires cross-compilation toolchain or building on Linux.
# On Linux, webkit2gtk-4.0 dev headers must be installed.
build-linux:
	wails build -platform linux/amd64

# Development mode with hot reload
dev:
	wails dev

# Run all Go tests
test:
	$(GO) test github.com/chazu/lignin/pkg/...

# Run Go tests with verbose output
test-v:
	$(GO) test -v github.com/chazu/lignin/pkg/...

# Lint with go vet
lint:
	$(GO) vet github.com/chazu/lignin/pkg/...

# Clean build artifacts
clean:
	$(GO) clean
	rm -rf build/bin/ dist/ frontend/dist/
