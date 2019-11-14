TARGETS           ?= linux/amd64 darwin/amd64 linux/amd64 windows/amd64 linux/386 linux/ppc64le linux/s390x linux/arm linux/arm64
PROJECT_NAME	  := s3-proxy
PKG				  := github.com/oxyno-zeta/$(PROJECT_NAME)
PKG_LIST		  := $(shell go list ${PKG}/... | grep -v /vendor/)

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
	golint -set_exit_status ${PKG_LIST}

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

test: dep ## Run unittests
	$(GO) test -short -cover -coverprofile=c.out ${PKG_LIST}

.PHONY: coverage-report
coverage-report:
	$(GO) tool cover -html=c.out -o coverage.html

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
HAS_GOLINT := $(shell command -v golint;)
HAS_COLORGO := $(shell command -v colorgo;)

.PHONY: dep
dep:
ifndef HAS_GOLINT
	GO111MODULE=off go get -u golang.org/x/lint/golint
endif
ifndef HAS_COLORGO
	GO111MODULE=off go get -u github.com/songgao/colorgo
endif
ifndef HAS_GIT
	$(error You must install Git)
endif
	go mod download
	go mod tidy
