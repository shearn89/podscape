BIN              := podscape
PKG              := ./...
GOBIN            ?= $(shell go env GOPATH)/bin
GOLANGCI_LINT    := $(GOBIN)/golangci-lint
STATICCHECK      := $(GOBIN)/staticcheck
GOVULNCHECK      := $(GOBIN)/govulncheck
GOIMPORTS        := $(GOBIN)/goimports

.PHONY: help build install run clean tidy \
        fmt fmt-check imports vet \
        lint lint-install staticcheck staticcheck-install \
        vuln vuln-install \
        test test-race cover \
        check ci tools

help: ## Show available targets
	@awk 'BEGIN{FS=":.*##"; printf "Targets:\n"} /^[a-zA-Z0-9_-]+:.*##/{printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## ──────────────────────────── build / run ────────────────────────────

build: ## Build the binary into ./bin
	go build -o bin/$(BIN) ./cmd/podscape

install: ## Install the binary into $GOBIN
	go install ./cmd/podscape

run: ## Run from source. Pass ARGS=... to forward flags.
	go run ./cmd/podscape $(ARGS)

clean: ## Remove build output
	rm -rf bin coverage.out coverage.html

tidy: ## go mod tidy
	go mod tidy

## ───────────────────────── formatting / vet ─────────────────────────

fmt: ## gofmt + goimports (if installed) on the tree
	gofmt -w .
	@if [ -x "$(GOIMPORTS)" ]; then $(GOIMPORTS) -w -local github.com/shearn89/podscape .; fi

fmt-check: ## Fail if any file needs gofmt
	@out=$$(gofmt -l . | grep -v '^vendor/'); \
	if [ -n "$$out" ]; then echo "gofmt needed on:"; echo "$$out"; exit 1; fi

imports: ## goimports — install with: go install golang.org/x/tools/cmd/goimports@latest
	@if [ ! -x "$(GOIMPORTS)" ]; then echo "goimports missing — run: go install golang.org/x/tools/cmd/goimports@latest"; exit 1; fi
	$(GOIMPORTS) -w -local github.com/shearn89/podscape .

vet: ## go vet
	go vet $(PKG)

## ─────────────────────────── static analysis ────────────────────────

lint: lint-install ## golangci-lint run
	$(GOLANGCI_LINT) run $(PKG)

lint-install:
	@if [ ! -x "$(GOLANGCI_LINT)" ] || ! "$(GOLANGCI_LINT)" version 2>/dev/null | grep -q "version 2"; then \
		echo "installing golangci-lint v2…"; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
	fi

staticcheck: staticcheck-install ## staticcheck
	$(STATICCHECK) $(PKG)

staticcheck-install:
	@if [ ! -x "$(STATICCHECK)" ]; then \
		echo "installing staticcheck…"; \
		go install honnef.co/go/tools/cmd/staticcheck@latest; \
	fi

vuln: vuln-install ## govulncheck
	$(GOVULNCHECK) $(PKG)

vuln-install:
	@if [ ! -x "$(GOVULNCHECK)" ]; then \
		echo "installing govulncheck…"; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi

## ───────────────────────────── tests ───────────────────────────────

test: ## go test
	go test $(PKG)

test-race: ## go test with the race detector
	go test -race $(PKG)

cover: ## Coverage report (writes coverage.out + coverage.html)
	go test -coverprofile=coverage.out $(PKG)
	go tool cover -html=coverage.out -o coverage.html
	@echo "open coverage.html"

## ───────────────────── one-shot quality / CI ───────────────────────

check: fmt-check vet lint staticcheck test ## Format-check + vet + lint + staticcheck + tests

ci: tidy fmt-check vet lint staticcheck vuln test-race ## Full CI gate

tools: lint-install staticcheck-install vuln-install ## Install all auxiliary tools
	@if [ ! -x "$(GOIMPORTS)" ]; then go install golang.org/x/tools/cmd/goimports@latest; fi
