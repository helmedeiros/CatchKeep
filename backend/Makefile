.PHONY: check generate generate-check fmt-check vet lint test vuln build

GO ?= go

check: generate-check fmt-check vet lint test

generate:
	$(GO) generate ./...

# Fails if generated code is out of date with api/openapi.yaml.
generate-check: generate
	@if ! git diff --quiet -- '*.gen.go' || [ -n "$$(git ls-files --others --exclude-standard -- '*.gen.go')" ]; then \
		echo "generated code is stale or uncommitted: run make generate and stage it"; exit 1; fi

fmt-check:
	@out=$$($(GO) tool golangci-lint fmt --diff ./... 2>&1); \
	if [ -n "$$out" ]; then echo "$$out"; echo "run: go tool golangci-lint fmt ./..."; exit 1; fi

vet:
	$(GO) vet ./...

lint:
	$(GO) tool golangci-lint run ./...

test:
	$(GO) test -race -shuffle=on -count=1 ./...

# Known vulnerabilities in the code paths CatchKeep uses; needs the internet,
# so CI runs it rather than the commit hooks.
vuln:
	$(GO) tool govulncheck ./...

build:
	$(GO) build -trimpath -o bin/catchkeep ./cmd/catchkeep
