.PHONY: build test test-v lint clean dev

GO := go

build:
	wails build

dev:
	wails dev

test:
	$(GO) test github.com/chazu/lignin/pkg/...

test-v:
	$(GO) test -v github.com/chazu/lignin/pkg/...

lint:
	$(GO) vet github.com/chazu/lignin/pkg/...

clean:
	$(GO) clean
	rm -rf build/bin/ dist/ frontend/dist/
