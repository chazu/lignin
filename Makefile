.PHONY: build test lint clean

GO := go

build:
	$(GO) build ./...

test:
	$(GO) test ./...

test-v:
	$(GO) test -v ./...

lint:
	$(GO) vet ./...

clean:
	$(GO) clean
	rm -rf build/ dist/
