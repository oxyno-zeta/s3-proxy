TARGETS           ?= linux/amd64 darwin/amd64 linux/amd64 windows/amd64 linux/386 linux/ppc64le linux/s390x linux/arm linux/arm64
PROJECT_NAME	  := s3-proxy
PKG				  := github.com/oxyno-zeta/$(PROJECT_NAME)

# go option
GO        ?= go
TAGS      :=
LDFLAGS   := -w -s
GOFLAGS   := -trimpath
BINDIR    := $(CURDIR)/bin
DISTDIR   := dist
CURRENT_DIR = $(CURDIR)

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
HAS_GOLANGCI_LINT := $(shell command -v golangci-lint;)
HAS_CURL:=$(shell command -v curl;)
HAS_MOCKGEN:=$(shell command -v mockgen;)

.DEFAULT_GOAL := code/lint

#############
#   Build   #
#############

.PHONY: code/lint
code/lint: setup/dep/install
	golangci-lint run ./...

.PHONY: code/build
code/build: code/clean setup/dep/install
	$(GO) build -o $(BINDIR)/$(PROJECT_NAME) $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' $(PKG)/cmd/${PROJECT_NAME}

.PHONY: code/build-cross
code/build-cross: code/clean setup/dep/install
ifdef HAS_GORELEASER
	goreleaser -p 2 --snapshot --skip-publish
endif
ifndef HAS_GORELEASER
	curl -sL https://git.io/goreleaser | bash -s -- -p 2 --snapshot --skip-publish
endif

.PHONY: code/clean
code/clean:
	@rm -rf $(BINDIR) $(DISTDIR)

.PHONY: code/docs
code/docs: setup/docs
	docker run --rm -it --user 1000:1000 -v ${PWD}:/docs -p 8000:8000 mkdocs serve -a 0.0.0.0:8000

.PHONY: code/build/docs
code/build/docs: setup/docs
	docker run --rm -it --user 1000:1000 -v ${PWD}:/docs -p 8000:8000 mkdocs build

#############
#  Release  #
#############

.PHONY: release/all
release/all: code/clean setup/dep/install
ifdef HAS_GORELEASER
	goreleaser -p 2
endif
ifndef HAS_GORELEASER
	curl -sL https://git.io/goreleaser | bash -s -- -p 2
endif

#############
#   Tests   #
#############

.PHONY: test/all
test/all: setup/dep/install
	$(GO) test --tags=unit,integration -v -coverpkg=./pkg/... -covermode=count -coverprofile=c.out.tmp ./pkg/...

.PHONY: test/unit
test/unit: setup/dep/install
	$(GO) test --tags=unit -v -coverpkg=./pkg/... -covermode=count -coverprofile=c.out.tmp ./pkg/...

.PHONY: test/integration
test/integration: setup/dep/install
	$(GO) test --tags=integration -v -coverpkg=./pkg/... -covermode=count -coverprofile=c.out.tmp ./pkg/...

.PHONY: test/coverage
test/coverage:
	cat c.out.tmp | grep -v "mock_" > c.out
	$(GO) tool cover -html=c.out -o coverage.html
	$(GO) tool cover -func c.out

#############
#   Setup   #
#############

.PHONY: down/services
down/services:
	docker rm -f opa || true
	docker rm -f keycloak || true

.PHONY: down/tracing-services
down/tracing-services:
	docker rm -f jaeger || true

.PHONY: down/metrics-services
down/metrics-services:
	docker rm -f prometheus || true
	docker rm -f grafana || true

.PHONY: setup/metrics-services
setup/metrics-services:
	docker run --rm -d --name prometheus -v $(CURRENT_DIR)/local-resources/prometheus/prometheus.yml:/prometheus/prometheus.yml --network=host prom/prometheus:v2.18.0 --web.listen-address=:9191
	docker run --rm -d --name grafana --network=host grafana/grafana:7.0.3

.PHONY: setup/tracing-services
setup/tracing-services: down/tracing-services
	@echo "Setup tracing services"
	docker run --name jaeger -d -p 6831:6831/udp -p 16686:16686 jaegertracing/all-in-one:latest

.PHONY: setup/services
setup/services: down/services
	tar czvf local-resources/opa/bundle.tar.gz --directory=local-resources/opa/bundle example/
	docker run -d --rm --name opa -p 8181:8181 -v $(CURRENT_DIR)/local-resources/opa/bundle.tar.gz:/bundle.tar.gz openpolicyagent/opa run --server --log-level debug --log-format text --bundle /bundle.tar.gz
	docker run -d --rm --name keycloak -p 8088:8080 -e KEYCLOAK_IMPORT=/tmp/realm-export.json -v $(CURRENT_DIR)/local-resources/keycloak/realm-export.json:/tmp/realm-export.json -e KEYCLOAK_USER=admin -e KEYCLOAK_PASSWORD=admin quay.io/keycloak/keycloak:11.0.3

.PHONY: setup/docs
setup/docs:
	docker build -t mkdocs -f Dockerfile.docs .

.PHONY: setup/mocks
setup/mocks:
	go generate ./...

.PHONY: setup/dep/install
setup/dep/install:
ifndef HAS_GOLANGCI_LINT
	@echo "=> Installing golangci-lint tool"
ifndef HAS_CURL
	$(error You must install curl)
endif
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.42.1
endif
ifndef HAS_GIT
	$(error You must install Git)
endif
ifndef HAS_MOCKGEN
	@echo "=> Installing mockgen tool"
	go get -u github.com/golang/mock/mockgen@v1.5.0
endif
	go mod download
	go mod tidy

.PHONY: setup/dep/update
setup/dep/update:
	go get -u ./...
