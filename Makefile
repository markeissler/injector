#
# Makefile for injector.
#
BIN_FILE        = inject
MODULE          = $(shell env GO111MODULE=on go list -m)
DATE           ?= $(shell date +%FT%T%z)
VERSION        ?= $(shell git describe --tags --always --dirty --match="*" 2> /dev/null || \
		       		cat $(CURDIR)/.version 2> /dev/null || echo v0)
COMMIT         ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BRANCH         ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
BIN             = $(CURDIR)/.bin
LINT_CONFIG     = $(CURDIR)/.golangci.yml
TARGETOS       ?= $(shell go env GOOS)
TARGETARCH     ?= $(shell go env GOARCH)
LDFLAGS_VERSION = -s -w -X main.Version=$(VERSION) -X main.BuildDate=$(DATE) -X main.GitCommit=$(COMMIT) -X main.GitBranch=$(BRANCH)

DOCKER  = docker
GO      = go
TIMEOUT = 15
V       = 0
Q       = $(if $(filter 1,$V),,@)
M       = $(shell printf "\033[34;1m▶\033[0m")

export GO111MODULE=on
export CGO_ENABLED=0
export GOPROXY=https://proxy.golang.org

.DEFAULT_GOAL := all

#
# Build
#

.PHONY: all
all: fmt lint build

build: ; $(info $(M) building executable...) @ ## Build program binary
	$Q env GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) $(GO) build \
		-tags release \
		-ldflags "$(LDFLAGS_VERSION) -X main.Platform=$(TARGETOS)/$(TARGETARCH)" \
		-o $(BIN)/$(BIN_FILE) main.go

#
# Tools
#

setup-tools: setup-lint

setup-lint:
	$(GO) get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.39

GOLINT = golangci-lint

#
# Tests
#

.PHONY: lint
lint: setup-lint ; $(info $(M) running golangci-lint) @ ## Run golangci-lint
	$Q $(GOLINT) run --timeout=5m -v -c $(LINT_CONFIG) ./...

.PHONY: fmt
fmt: ; $(info $(M) running gofmt...) @ ## Run gofmt on all source files
	$Q $(GO) fmt $(PKGS)

#
# Misc
#

.PHONY: clean
clean: ; $(info $(M) cleaning...)	@ ## Cleanup everything
	@rm -rf $(BIN)
	@rm -rf test/tests.* test/coverage.*

.PHONY: help
help:
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.PHONY: version
version:
	@echo $(VERSION)
