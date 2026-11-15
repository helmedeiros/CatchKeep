.PHONY: check fmt-check vet lint test build

GO ?= go

check: fmt-check vet lint test

fmt-check:
	@out=$$($(GO) tool golangci-lint fmt --diff ./... 2>&1); \
	if [ -n "$$out" ]; then echo "$$out"; echo "run: go tool golangci-lint fmt ./..."; exit 1; fi

vet:
	$(GO) vet ./...

lint:
	$(GO) tool golangci-lint run ./...

test:
	$(GO) test -race -shuffle=on -count=1 ./...

build:
	$(GO) build -trimpath -o bin/catchkeep ./cmd/catchkeep
