# All possible values : darwin/amd64 linux/amd64 windows/amd64 linux/386 linux/ppc64le linux/s390x linux/arm linux/arm64
TARGETS           ?= linux/amd64
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
DISTDIR   := _dist

# Required for globs to work correctly
SHELL=/usr/bin/env bash

#  Version

GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_SHA    = $(shell git rev-parse --short HEAD)
GIT_TAG    = $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
DATE	   = $(shell date +%F_%T%Z)

BINARY_VERSION ?= ${GIT_TAG}
ifeq ($(BINARY_VERSION),)
	BINARY_VERSION = ${GIT_SHA}
endif
# Clear the "unreleased" string in Metadata
ifneq ($(GIT_TAG),)
	LDFLAGS += -X ${PKG}/pkg/version.Metadata=
endif
LDFLAGS += -X ${PKG}/pkg/version.Version=${BINARY_VERSION}
LDFLAGS += -X ${PKG}/pkg/version.GitCommit=${GIT_COMMIT}
LDFLAGS += -X ${PKG}/pkg/version.BuildDate=${DATE}

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
build-cross: LDFLAGS += -extldflags "-static"
build-cross: clean dep
	CGO_ENABLED=0 gox -output="$(DISTDIR)/bin/$(BINARY_VERSION)/{{.OS}}-{{.Arch}}/{{.Dir}}" -osarch='$(TARGETS)' $(if $(TAGS),-tags '$(TAGS)',) -ldflags '$(LDFLAGS)' ${PKG}/cmd/${PROJECT_NAME}

.PHONY: release
release: build-cross
	cp Dockerfile $(DISTDIR)/bin/$(BINARY_VERSION)/linux-amd64
	docker build -t oxynozeta/s3-proxy:$(BINARY_VERSION) $(DISTDIR)/bin/$(BINARY_VERSION)/linux-amd64

.PHONY: docker-latest
docker-latest:
	docker tag oxynozeta/s3-proxy:$(BINARY_VERSION) oxynozeta/s3-proxy:latest

.PHONY: docker-publish
docker-publish:
	docker push oxynozeta/s3-proxy:$(BINARY_VERSION)

.PHONY: docker-publish-latest
docker-publish-latest:
	docker push oxynozeta/s3-proxy:latest

test: dep ## Run unittests
	$(GO) test -short -cover -coverprofile=c.out ${PKG_LIST}

.PHONY: coverage-report
coverage-report:
	$(GO) tool cover -html=c.out -o coverage.html

.PHONY: clean
clean:
	@rm -rf $(BINDIR) $(DISTDIR)

#############
# Bootstrap #
#############

HAS_GIT := $(shell command -v git;)
HAS_GOX := $(shell command -v gox;)
HAS_GOLINT := $(shell command -v golint;)
HAS_COLORGO := $(shell command -v colorgo;)

.PHONY: dep
dep:
ifndef HAS_GOX
	GO111MODULE=off go get -u github.com/mitchellh/gox
endif
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
