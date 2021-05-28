#
# Makefile for injector.
#
BIN_FILE         = inject
MODULE           = $(shell env GO111MODULE=on go list -m)
DATE            ?= $(shell date +%FT%T%z)
VERSION         ?= $(shell git describe --tags --always --dirty --match="*" 2> /dev/null || \
		       		cat $(CURDIR)/.version 2> /dev/null || echo v0)
COMMIT          ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BRANCH          ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
BIN              = $(CURDIR)/.bin
LINT_CONFIG      = $(CURDIR)/.golangci.yml
PLATFORMS        = darwin/amd64 darwin/arm64 linux/amd64 linux/arm64
PLATFORM_CURRENT = $(shell go env GOOS)_$(shell go env GOARCH)
LDFLAGS_VERSION  = -s -w -X main.Version=$(VERSION) -X main.BuildDate=$(DATE) -X main.GitCommit=$(COMMIT) -X main.GitBranch=$(BRANCH)

DOCKER  = docker
GO      = go
TIMEOUT = 15
V       = 0 # set to `1` to echo suppressed commands for debugging!
Q       = $(if $(filter 1,$V),,@)
M       = $(shell printf "\033[34;1m▶\033[0m")

export GO111MODULE=on
export CGO_ENABLED=0
export GOPROXY=https://proxy.golang.org

.DEFAULT_GOAL := all

#
# Build for current platform
#
sources := $(wildcard *.go)
build    = $(info $(M) building executable for $(1)/$(2)...) \
	$Q env GOOS=$(1) GOARCH=$(2) $(GO) build \
		-tags release \
		-ldflags "$(LDFLAGS_VERSION) -X main.Platform=$(1)/$(2)" \
		-o $(BIN)/$(BIN_FILE)$(3)
tar      = $(info $(M) tar archiving executable for $(1)/$(2)...) \
	$Q cd $(BIN) && tar -czf $(1)_$(2).tar.gz $(BIN_FILE)$(3)
zip      = $(info $(M) zip archiving executable for $(1)/$(2)...) \
	$Q cd $(BIN) && zip -q $(1)_$(2).zip $(BIN_FILE)$(3)

.PHONY: all
all: fmt lint build/all

.PHONY: build
build: fmt lint build/$(PLATFORM_CURRENT)

.PHONY: build/all build/darwin build/linux
build/all: build/darwin build/linux
build/darwin: build/darwin_amd64.tar.gz build/darwin_amd64.zip build/darwin_arm64.tar.gz build/darwin_arm64.zip
	@ rm $(BIN)/$(BIN_FILE)
build/linux: build/linux_amd64.tar.gz build/linux_amd64.zip build/linux_arm64.tar.gz build/linux_arm64.zip
	@ rm $(BIN)/$(BIN_FILE)

.PHONY: build/darwin_amd64 build/darwin_amd64.tar.gz build/darwin_amd64.zip
build/darwin_amd64: $(sources) clean
	$(call build,darwin,amd64,)
build/darwin_amd64.tar.gz: build/darwin_amd64
	$(call tar,darwin,amd64)
build/darwin_amd64.zip: build/darwin_amd64
	$(call zip,darwin,amd64)

build/darwin_arm64: $(sources) clean
	$(call build,darwin,arm64,)
build/darwin_arm64.tar.gz: build/darwin_arm64
	$(call tar,darwin,arm64)
build/darwin_arm64.zip: build/darwin_arm64
	$(call zip,darwin,arm64)

.PHONY: build/linux_amd64 build/linux_amd64.tar.gz build/linux_amd64.zip
build/linux_amd64: $(sources) clean
	$(call build,linux,amd64,)
build/linux_amd64.tar.gz: build/linux_amd64
	$(call tar,linux,amd64)
build/linux_amd64.zip: build/linux_amd64
	$(call zip,linux,amd64)

.PHONY: build/linux_arm64 build/linux_arm64.tar.gz build/linux_arm64.zip
build/linux_arm64: $(sources) clean
	$(call build,linux,arm64,)
build/linux_arm64.tar.gz: build/linux_arm64
	$(call tar,linux,arm64)
build/linux_arm64.zip: build/linux_arm64
	$(call zip,linux,arm64)

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

.PHONY: changelog
changelog: ; $(info $(M) generating changelog...)	@ ## Generating CHANGELOG.md
ifndef GITHUB_TOKEN
	$(error GITHUB_TOKEN is undefined)
endif
	$Q $(DOCKER) run --rm \
		-v $(CURDIR):/usr/local/src/app \
		-w /usr/local/src/app ferrarimarco/github-changelog-generator \
		--user AlphaFlow --project injector \
		--token $(GITHUB_TOKEN)

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
