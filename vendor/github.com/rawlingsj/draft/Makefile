DOCKER_REGISTRY ?= docker.io
IMAGE_PREFIX    ?= microsoft
IMAGE_TAG       ?= canary
SHORT_NAME      ?= draft
TARGETS         = darwin/amd64 linux/amd64 linux/386 linux/arm windows/amd64
DIST_DIRS       = find * -type d -exec
APP             = draft

# go option
GO        ?= go
PKG       := $(shell glide novendor)
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

# usage: make clean build-cross dist APP=draft|draftd VERSION=v2.0.0-alpha.3
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

.PHONY: check-docker
check-docker:
	@if [ -z $$(which docker) ]; then \
	  echo "Missing \`docker\` client which is required for development"; \
	  exit 2; \
	fi

.PHONY: check-helm
check-helm:
	@if [ -z $$(which helm) ]; then \
	  echo "Missing \`helm\` client which is required for development"; \
	  exit 2; \
	fi

.PHONY: docker-binary
docker-binary: BINDIR = ./rootfs/bin
docker-binary: GOFLAGS += -a -installsuffix cgo
docker-binary:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO) build -o $(BINDIR)/draftd $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' github.com/Azure/draft/cmd/draftd

.PHONY: docker-build
docker-build: check-docker docker-binary compress-binary
	docker build --rm -t ${IMAGE} .
	docker tag ${IMAGE} ${MUTABLE_IMAGE}

.PHONY: compress-binary
compress-binary: BINDIR = ./rootfs/bin
compress-binary:
	@if [ -z $$(which upx) ]; then \
	  echo "Missing \`upx\` tool to compress binaries"; \
	else \
	  upx --quiet ${BINDIR}/draftd; \
	fi

.PHONY: serve
serve: check-helm
	helm install chart/ --name ${APP} --namespace kube-system \
		--set image.repository=${DOCKER_REGISTRY}/${IMAGE_PREFIX}/${SHORT_NAME},image.tag=${IMAGE_TAG}

.PHONY: unserve
unserve: check-helm
	-helm delete --purge ${APP}

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
	$(GO) test $(GOFLAGS) -cover -run $(TESTS) $(PKG) $(TESTFLAGS)

.PHONY: test-e2e
test-e2e:
	./tests/e2e.sh

HAS_GOMETALINTER := $(shell command -v gometalinter;)
HAS_GLIDE := $(shell command -v glide;)
HAS_GOX := $(shell command -v gox;)
HAS_GIT := $(shell command -v git;)
HAS_BINDATA := $(shell command -v go-bindata;)

.PHONY: bootstrap
bootstrap:
ifndef HAS_GOMETALINTER
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
endif
ifndef HAS_GLIDE
	go get -u github.com/Masterminds/glide
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
	glide install --strip-vendor
	scripts/setup-apimachinery.sh

include versioning.mk
