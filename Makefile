TARGETS           ?= linux/amd64 darwin/amd64 linux/amd64 windows/amd64 linux/386 linux/ppc64le linux/s390x linux/arm linux/arm64
PROJECT_NAME	  := s3-proxy
PKG				  := github.com/oxyno-zeta/$(PROJECT_NAME)

# go option
GO        ?= go
TAGS      :=
LDFLAGS   := -w -s
GOFLAGS   := -i
BINDIR    := $(CURDIR)/bin
DISTDIR   := dist
CURRENT_DIR = $(shell pwd)

# Required for globs to work correctly
SHELL=/usr/bin/env bash

#  Version

GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_SHA    = $(shell git rev-parse --short HEAD)
GIT_TAG    = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
DATE	   = $(shell date +%F_%T%Z)

BINARY_VERSION = ${GIT_SHA}
LDFLAGS += -X ${PKG}/pkg/${PROJECT_NAME}/version.Version=${BINARY_VERSION}
LDFLAGS += -X ${PKG}/pkg/${PROJECT_NAME}/version.GitCommit=${GIT_COMMIT}
LDFLAGS += -X ${PKG}/pkg/${PROJECT_NAME}/version.BuildDate=${DATE}

HAS_GORELEASER := $(shell command -v goreleaser;)
HAS_GIT := $(shell command -v git;)
HAS_COLORGO := $(shell command -v colorgo;)
HAS_GOLANGCI_LINT := $(shell command -v golangci-lint;)
HAS_CURL:=$(shell command -v curl;)

.DEFAULT_GOAL := code/lint

#############
#   Build   #
#############

.PHONY: code/lint
code/lint: setup/dep/install
	golangci-lint run ./...

.PHONY: code/build
code/build: code/clean setup/dep/install
	GOBIN=$(BINDIR) colorgo install $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' $(PKG)/cmd/${PROJECT_NAME}

.PHONY: code/build-cross
code/build-cross: code/clean setup/dep/install
ifdef HAS_GORELEASER
	goreleaser --snapshot --skip-publish
endif
ifndef HAS_GORELEASER
	curl -sL https://git.io/goreleaser | bash -s -- --snapshot --skip-publish
endif

.PHONY: code/clean
code/clean:
	@rm -rf $(BINDIR) $(DISTDIR)

#############
#  Release  #
#############

.PHONY: release/all
release/all: code/clean setup/dep/install
ifdef HAS_GORELEASER
	goreleaser
endif
ifndef HAS_GORELEASER
	curl -sL https://git.io/goreleaser | bash
endif

#############
#   Tests   #
#############

.PHONY: test/all
test/all: setup/dep/install
	$(GO) test --tags=unit,integration -v -coverpkg=./pkg/... -coverprofile=c.out ./pkg/...

.PHONY: test/unit
test/unit: setup/dep/install
	$(GO) test --tags=unit -v -coverpkg=./pkg/... -coverprofile=c.out ./pkg/...

.PHONY: test/integration
test/integration: setup/dep/install
	$(GO) test --tags=integration -v -coverpkg=./pkg/... -coverprofile=c.out ./pkg/...

.PHONY: test/coverage
test/coverage:
	$(GO) tool cover -html=c.out -o coverage.html
	$(GO) tool cover -func c.out

#############
#   Setup   #
#############

.PHONY: setup/services
setup/services:
	docker run -d --rm --name keycloak -p 8080:8080 -e KEYCLOAK_IMPORT=/tmp/realm-export.json -v $(CURRENT_DIR)/tests/realm-export.json:/tmp/realm-export.json -e KEYCLOAK_USER=admin -e KEYCLOAK_PASSWORD=admin jboss/keycloak:10.0.0

.PHONY: setup/dep/install
setup/dep/install:
ifndef HAS_GOLANGCI_LINT
	@echo "=> Installing golangci-lint tool"
ifndef HAS_CURL
	$(error You must install curl)
endif
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.25.0
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

.PHONY: setup/dep/update
setup/dep/update:
	go get -u ./...
