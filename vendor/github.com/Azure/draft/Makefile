DOCKER_REGISTRY ?= docker.io
IMAGE_PREFIX    ?= microsoft
IMAGE_TAG       ?= canary
SHORT_NAME      ?= draft
TARGETS         = darwin/amd64 linux/amd64 linux/386 linux/arm windows/amd64
DIST_DIRS       = find * -type d -exec
APP             = draft

# go option
GO        ?= go
TAGS      := kqueue
TESTS     := .
TESTFLAGS :=
LDFLAGS   :=
GOFLAGS   :=
BINDIR    := $(CURDIR)/bin
BINARIES  := draft

# Required for globs to work correctly
SHELL=/bin/bash

.PHONY: all
all: build

.PHONY: build
build:
	GOBIN=$(BINDIR) $(GO) install $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' github.com/Azure/draft/cmd/...

# usage: make clean build-cross dist APP=draft VERSION=v2.0.0-alpha.3
.PHONY: build-cross
build-cross: LDFLAGS += -extldflags "-static"
build-cross:
	CGO_ENABLED=0 gox -output="_dist/{{.OS}}-{{.Arch}}/{{.Dir}}" -osarch='$(TARGETS)' $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' github.com/Azure/draft/cmd/$(APP)

.PHONY: dist
dist:
	( \
		cd _dist && \
		$(DIST_DIRS) cp ../LICENSE {} \; && \
		$(DIST_DIRS) cp ../README.md {} \; && \
		$(DIST_DIRS) tar -zcf draft-${VERSION}-{}.tar.gz {} \; \
	)

.PHONY: checksum
checksum:
	for f in _dist/*.gz ; do \
		shasum -a 256 "$${f}"  | awk '{print $$1}' > "$${f}.sha256" ; \
	done

.PHONY: clean
clean:
	-rm bin/*
	-rm rootfs/bin/*
	-rm -rf _dist/

.PHONY: test
test: TESTFLAGS += -race -v
test: test-lint test-unit

test-cover:
	scripts/cover.sh

.PHONY: test-lint
test-lint:
	scripts/lint.sh

.PHONY: test-unit
test-unit:
	$(GO) test $(GOFLAGS) -cover -run $(TESTS) ./... $(TESTFLAGS)

HAS_GOMETALINTER := $(shell command -v gometalinter;)
HAS_DEP := $(shell command -v dep;)
HAS_GOX := $(shell command -v gox;)
HAS_GIT := $(shell command -v git;)
HAS_BINDATA := $(shell command -v go-bindata;)

.PHONY: bootstrap
bootstrap:
ifndef HAS_GOMETALINTER
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
endif
ifndef HAS_DEP
	go get -u github.com/golang/dep/cmd/dep
endif
ifndef HAS_GOX
	go get -u github.com/mitchellh/gox
endif
ifndef HAS_GIT
	$(error You must install git)
endif
ifndef HAS_BINDATA
	go get github.com/jteeuwen/go-bindata/...
endif
	dep ensure -v
	scripts/setup-apimachinery.sh
	scripts/setup-protobuf-include.sh

include versioning.mk
