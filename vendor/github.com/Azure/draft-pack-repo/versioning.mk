GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_SHA := $(shell git rev-parse --short HEAD)
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null)

ifdef VERSION
	BINARY_VERSION = $(VERSION)
endif

BINARY_VERSION ?= ${GIT_TAG}-${GIT_SHA}

LDFLAGS += -X github.com/Azure/draft-pack-repo/version.Version=${GIT_TAG}
LDFLAGS += -X github.com/Azure/draft-pack-repo/version.GitCommit=${GIT_COMMIT}
