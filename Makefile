GO ?= go

.PHONY: tools fmt lint test build check

tools:
	$(GO) install mvdan.cc/gofumpt@latest

fmt:
	gofumpt -w .

lint:
	$(GO) vet ./...

test:
	$(GO) test ./...

build:
	$(GO) build ./cmd/csm

check: fmt lint test build
