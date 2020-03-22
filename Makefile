TARGETS           ?= linux/amd64 darwin/amd64 linux/amd64 windows/amd64 linux/386 linux/ppc64le linux/s390x linux/arm linux/arm64
PROJECT_NAME	  := s3-proxy
PKG				  := github.com/oxyno-zeta/$(PROJECT_NAME)

# go option
GO        ?= go
TAGS      :=
TESTS     := .
TESTFLAGS :=
LDFLAGS   := -w -s
GOFLAGS   := -i
BINDIR    := $(CURDIR)/bin
DISTDIR   := dist

# Required for globs to work correctly
SHELL=/usr/bin/env bash

#  Version

GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_SHA    = $(shell git rev-parse --short HEAD)
GIT_TAG    = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
DATE	   = $(shell date +%F_%T%Z)

BINARY_VERSION = ${GIT_SHA}
LDFLAGS += -X ${PKG}/pkg/version.Version=${BINARY_VERSION}
LDFLAGS += -X ${PKG}/pkg/version.GitCommit=${GIT_COMMIT}
LDFLAGS += -X ${PKG}/pkg/version.BuildDate=${DATE}

HAS_GORELEASER := $(shell command -v goreleaser;)

#############
#   Build   #
#############

.PHONY: all
all: lint test build

.PHONY: lint
lint: dep
	golangci-lint run ./...

.PHONY: build
build: clean dep
	GOBIN=$(BINDIR) colorgo install $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' $(PKG)/cmd/${PROJECT_NAME}

.PHONY: build-cross
build-cross: clean dep
ifdef HAS_GORELEASER
	goreleaser --snapshot --skip-publish
endif
ifndef HAS_GORELEASER
	curl -sL https://git.io/goreleaser | bash -s -- --snapshot --skip-publish
endif

.PHONY: release
release: clean dep
ifdef HAS_GORELEASER
	goreleaser
endif
ifndef HAS_GORELEASER
	curl -sL https://git.io/goreleaser | bash
endif

.PHONY: test
test: dep
	$(GO) test --tags=unit,integration -v -coverpkg=./pkg/... -coverprofile=c.out ./pkg/...

.PHONY: test-unit
test-unit: dep
	$(GO) test --tags=unit -v -coverpkg=./pkg/... -coverprofile=c.out ./pkg/...

.PHONY: test-integration
test-integration: dep
	$(GO) test --tags=integration -v -coverpkg=./pkg/... -coverprofile=c.out ./pkg/...

.PHONY: coverage-report
coverage-report:
	$(GO) tool cover -html=c.out -o coverage.html
	$(GO) tool cover -func c.out

.PHONY: clean
clean:
	@rm -rf $(BINDIR) $(DISTDIR)

.PHONY: update-dep
update-dep:
	go get -u ./...

#############
# Bootstrap #
#############

HAS_GIT := $(shell command -v git;)
HAS_COLORGO := $(shell command -v colorgo;)
HAS_GOLANGCI_LINT := $(shell command -v golangci-lint;)
HAS_CURL:=$(shell command -v curl;)

.PHONY: dep
dep:
ifndef HAS_GOLANGCI_LINT
	@echo "=> Installing golangci-lint tool"
ifndef HAS_CURL
	$(error You must install curl)
endif
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.24.0
endif
ifndef HAS_COLORGO
	@echo "=> Installing colorgo tool"
	go get -u github.com/songgao/colorgo
endif
ifndef HAS_GIT
	$(error You must install Git)
endif
	go mod download
	go mod tidy
