NAME            = pack-repo
DIST_DIRS       = find * -type d -exec

# go option
GO        ?= go
TAGS      := kqueue
TESTFLAGS :=
LDFLAGS   :=
GOFLAGS   :=
BINDIR    := $(CURDIR)/bin

# Required for globs to work correctly
SHELL=/bin/bash

.PHONY: all
all: build

.PHONY: build
build:
	GOBIN=$(BINDIR) $(GO) install $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' github.com/Azure/draft-pack-repo/cmd/...

.PHONY: build-cross
build-cross: LDFLAGS += -extldflags "-static"
build-cross:
	CGO_ENABLED=0 gox -output="_dist/{{.OS}}-{{.Arch}}/{{.Dir}}" -osarch='$(TARGETS)' $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' github.com/Azure/draft-pack-repo/cmd/$(NAME)

.PHONY: dist
dist:
	( \
		cd _dist && \
		$(DIST_DIRS) cp ../LICENSE {} \; && \
		$(DIST_DIRS) cp ../README.md {} \; && \
		$(DIST_DIRS) tar -zcf $(NAME)-${VERSION}-{}.tar.gz {} \; \
	)

.PHONY: checksum
checksum:
	for f in _dist/*.gz ; do \
		shasum -a 256 "$${f}"  | awk '{print $$1}' > "$${f}.sha256" ; \
	done

.PHONY: compress-binary
compress-binary: BINDIR = ./rootfs/bin
compress-binary:
	@if [ -z $$(which upx) ]; then \
	  echo "Missing \`upx\` tool to compress binaries"; \
	else \
	  upx --quiet ${BINDIR}/${NAME}; \
	fi

.PHONY: clean
clean:
	-rm bin/*
	-rm -rf _dist/

.PHONY: test
test: TESTFLAGS += -race -v
test: test-lint test-cover

test-cover:
	scripts/cover.sh

.PHONY: test-lint
test-lint:
	scripts/lint.sh

.PHONY: test-unit
test-unit:
	$(GO) test $(GOFLAGS) -cover -run $(TESTFLAGS) ./cmd/... ./version/...

HAS_GOMETALINTER := $(shell command -v gometalinter;)
HAS_DEP := $(shell command -v dep;)
HAS_GOX := $(shell command -v gox;)
HAS_GIT := $(shell command -v git;)

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
	dep ensure

include versioning.mk

# Set VERSION to build release assets for a specific version
.PHONY: release-assets
release-assets: build-cross dist checksum
