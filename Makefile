#
# Copyright (C) Original Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#         http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# Make does not offer a recursive wildcard function, so here's one:
rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))

SHELL := /bin/bash
NAME := jx
BUILD_TARGET = build
MAIN_SRC_FILE=cmd/jx/jx.go
GO := GO111MODULE=on go
GO_NOMOD :=GO111MODULE=off go
REV := $(shell git rev-parse --short HEAD 2> /dev/null || echo 'unknown')
REV := $(shell git rev-parse --short HEAD 2> /dev/null || echo 'unknown')
ORG := jenkins-x
ORG_REPO := $(ORG)/$(NAME)
RELEASE_ORG_REPO := $(ORG_REPO)
ROOT_PACKAGE := github.com/$(ORG_REPO)
GO_VERSION := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_DEPENDENCIES := $(call rwildcard,pkg/,*.go) $(call rwildcard,cmd/jx/,*.go)

BRANCH     := $(shell git rev-parse --abbrev-ref HEAD 2> /dev/null  || echo 'unknown')
BUILD_DATE := $(shell date +%Y%m%d-%H:%M:%S)
CGO_ENABLED = 0

REPORTS_DIR=$(BUILD_TARGET)/reports

GOTEST := $(GO) test
# If available, use gotestsum which provides more comprehensive output
# This is used in the CI builds
ifneq (, $(shell which gotestsum 2> /dev/null))
GOTESTSUM_FORMAT ?= standard-quiet
GOTEST := GO111MODULE=on gotestsum --junitfile $(REPORTS_DIR)/integration.junit.xml --format $(GOTESTSUM_FORMAT) --
endif

# set dev version unless VERSION is explicitly set via environment
VERSION ?= $(shell echo "$$(git for-each-ref refs/tags/ --count=1 --sort=-version:refname --format='%(refname:short)' 2>/dev/null)-dev+$(REV)" | sed 's/^v//')

# Build flags for setting build-specific configuration at build time - defaults to empty
BUILD_TIME_CONFIG_FLAGS ?= ""

# Full build flags used when building binaries. Not used for test compilation/execution.
BUILDFLAGS :=  -ldflags \
  " -X $(ROOT_PACKAGE)/pkg/version.Version=$(VERSION)\
		-X $(ROOT_PACKAGE)/pkg/version.Revision='$(REV)'\
		-X $(ROOT_PACKAGE)/pkg/version.Branch='$(BRANCH)'\
		-X $(ROOT_PACKAGE)/pkg/version.BuildDate='$(BUILD_DATE)'\
		-X $(ROOT_PACKAGE)/pkg/version.GoVersion='$(GO_VERSION)'\
		$(BUILD_TIME_CONFIG_FLAGS)"

# Some tests expect default values for version.*, so just use the config package values there.
TEST_BUILDFLAGS :=  -ldflags " $(BUILD_TIME_CONFIG_FLAGS)"

ifdef DEBUG
BUILDFLAGS := -gcflags "all=-N -l" $(BUILDFLAGS)
endif

ifdef PARALLEL_BUILDS
BUILDFLAGS += -p $(PARALLEL_BUILDS)
GOTEST += -p $(PARALLEL_BUILDS)
else
# -p 4 seems to work well for people
GOTEST += -p 4
endif

ifdef DISABLE_TEST_CACHING
GOTEST += -count=1
endif

# support for building a covered jx binary (one with the coverage instrumentation compiled in). The `build-covered`
# target also builds the covered binary explicitly
COVERED_MAIN_SRC_FILE=./cmd/jx
COVERAGE_BUILDFLAGS = -c -tags covered_binary -coverpkg=./... -covermode=count
COVERAGE_BUILD_TARGET = test
ifdef COVERED_BINARY
BUILDFLAGS += $(COVERAGE_BUILDFLAGS)
BUILD_TARGET = $(COVERAGE_BUILD_TARGET)
MAIN_SRC_FILE = $(COVERED_MAIN_SRC_FILE)

endif

TEST_PACKAGE ?= ./...
COVER_OUT:=$(REPORTS_DIR)/cover.out
COVERFLAGS=-coverprofile=$(COVER_OUT) --covermode=count --coverpkg=./...

.PHONY: list
list: ## List all make targets
	@$(MAKE) -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

all: build ## Build the binary
full: check ## Build and run the tests
check: build test ## Build and run the tests
get-test-deps: ## Install test dependencies
	$(GO_NOMOD) get github.com/axw/gocov/gocov
	$(GO_NOMOD) get -u gopkg.in/matm/v1/gocov-html


print-version: ## Print version
	@echo $(VERSION)

build: $(GO_DEPENDENCIES) ## Build jx binary for current OS
	CGO_ENABLED=$(CGO_ENABLED) $(GO) $(BUILD_TARGET) $(BUILDFLAGS) -o build/$(NAME) $(MAIN_SRC_FILE)

build-all: $(GO_DEPENDENCIES) build make-reports-dir ## Build all files - runtime, all tests etc.
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -run=nope -tags=integration -failfast -short ./... $(BUILDFLAGS)

.PHONY: build-covered
build-covered: $(GO_DEPENDENCIES) ## Build jx binary for current OS with coverage instrumentation to build/$(NAME).covered
	CGO_ENABLED=$(CGO_ENABLED) $(GO) $(COVERAGE_BUILD_TARGET) $(BUILDFLAGS) $(COVERAGE_BUILDFLAGS) -o build/$(NAME).covered $(COVERED_MAIN_SRC_FILE)

tidy-deps: ## Cleans up dependencies
	$(GO) mod tidy
	# mod tidy only takes compile dependencies into account, let's make sure we capture tooling dependencies as well
	@$(MAKE) install-generate-deps

.PHONY: make-reports-dir
make-reports-dir:
	mkdir -p $(REPORTS_DIR)

test: make-reports-dir ## Run ONLY the tests that have no build flags (and example of a build tag is the "integration" build tag)
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) $(COVERFLAGS) -failfast -short ./... $(TEST_BUILDFLAGS)

test-cloud-gke: make-reports-dir ## Run ONLY the tests that have no build flags (and example of a build tag is the "integration" build tag)
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -coverprofile=$(COVER_OUT) --covermode=count --coverpkg=./pkg/cloud/gke -failfast -short ./pkg/cloud/gke $(TEST_BUILDFLAGS)

test-report: make-reports-dir get-test-deps test ## Create the test report
	@gocov convert $(COVER_OUT) | gocov report

test-report-html: make-reports-dir get-test-deps test ## Create the test report in HTML format
	@gocov convert $(COVER_OUT) | gocov-html > $(REPORTS_DIR)/cover.html && open $(REPORTS_DIR)/cover.html

test-verbose: make-reports-dir ## Run the unit tests in verbose mode
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -v $(COVERFLAGS) -failfast ./... $(TEST_BUILDFLAGS)

test-slow-report: get-test-deps test-slow make-reports-dir
	@gocov convert $(COVER_OUT) | gocov report

test-slow: make-reports-dir ## Run unit tests sequentially
	@CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) $(COVERFLAGS) ./... $(TEST_BUILDFLAGS)

test-slow-report-html: make-reports-dir get-test-deps test-slow
	@gocov convert $(COVER_OUT) | gocov-html > $(REPORTS_DIR)/cover.html && open $(REPORTS_DIR)/cover.html

test-integration: get-test-deps## Run the integration tests
	@CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -tags=integration  -short ./... $(TEST_BUILDFLAGS)

test-integration1: make-reports-dir
	@CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -tags=integration $(COVERFLAGS) -short ./... $(TEST_BUILDFLAGS) -test.v -run $(TEST)

test-integration1-pkg: make-reports-dir
	@CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -tags=integration $(COVERFLAGS) -short $(PKG) -test.v -run $(TEST)

test-rich-integration1: make-reports-dir
	@CGO_ENABLED=$(CGO_ENABLED) richgo test -tags=integration $(COVERFLAGS) -short -test.v $(TEST_PACKAGE) $(TEST_BUILDFLAGS) -run $(TEST)

test-integration-report: make-reports-dir get-test-deps test-integration ## Create the integration tests report
	@gocov convert $(COVER_OUT) | gocov report

test-integration-report-html: make-reports-dir get-test-deps test-integration
	@gocov convert $(COVER_OUT) | gocov-html > $(REPORTS_DIR)/cover.html && open $(REPORTS_DIR)/cover.html

test-slow-integration: make-reports-dir ## Run the any tests without a build tag as well as those that have the "integration" build tag. This target is run during CI.
	@CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -tags=integration $(COVERFLAGS) ./... $(TEST_BUILDFLAGS)

test-slow-integration-report: make-reports-dir get-test-deps test-slow-integration
	@gocov convert $(COVER_OUT) | gocov report

test-slow-integration-report-html: make-reports-dir get-test-deps test-slow-integration
	@gocov convert $(COVER_OUT) | gocov-html > $(REPORTS_DIR)/cover.html && open $(REPORTS_DIR)/cover.html

test-soak: make-reports-dir get-test-deps
	@CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -tags soak $(COVERFLAGS) ./... $(TEST_BUILDFLAGS)

test1: get-test-deps make-reports-dir ## Runs single test specified by test name, eg 'make test1 TEST=TestFoo'
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) ./... $(TEST_BUILDFLAGS) -test.v -run $(TEST)

test1-pkg: get-test-deps make-reports-dir ## Runs single test specified by path to test file, eg 'make test-single-file PKG=./pkg/util TEST=TestFoo'
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) $(PKG) $(TEST_BUILDFLAGS) -test.v $(TEST)

testbin: get-test-deps make-reports-dir
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -c github.com/jenkins-x/jx/pkg/cmd -o build/jx-test $(TEST_BUILDFLAGS)

testbin-gits: get-test-deps make-reports-dir
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -c github.com/jenkins-x/jx/pkg/gits -o build/jx-test-gits $(TEST_BUILDFLAGS)

debugtest1: testbin
	cd pkg/jx/cmd && dlv --listen=:2345 --headless=true --api-version=2 exec ../../../build/jx-test -- $(TEST_BUILDFLAGS) -test.run $(TEST)

debugtest1gits: testbin-gits
	cd pkg/gits && dlv --log --listen=:2345 --headless=true --api-version=2 exec ../../build/jx-test-gits -- $(TEST_BUILDFLAGS) -test.run $(TEST)

inttestbin: get-test-deps
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -tags=integration -c github.com/jenkins-x/jx/pkg/cmd -o build/jx-inttest $(TEST_BUILDFLAGS)

debuginttest1: inttestbin
	cd pkg/jx/cmd && dlv --listen=:2345 --headless=true --api-version=2 exec ../../../build/jx-inttest -- $(TEST_BUILDFLAGS) -test.run $(TEST)

install: $(GO_DEPENDENCIES) ## Install the binary
	GOBIN=${GOPATH}/bin $(GO) install $(BUILDFLAGS) $(MAIN_SRC_FILE)

linux: ## Build for Linux
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO) $(BUILD_TARGET) $(BUILDFLAGS) -o build/linux/$(NAME) $(MAIN_SRC_FILE)
	chmod +x build/linux/$(NAME)

arm: ## Build for ARM
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm $(GO) $(BUILD_TARGET) $(BUILDFLAGS) -o build/arm/$(NAME) $(MAIN_SRC_FILE)
	chmod +x build/arm/$(NAME)

win: ## Build for Windows
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GO) $(BUILD_TARGET) $(BUILDFLAGS) -o build/win/$(NAME)-windows-amd64.exe $(MAIN_SRC_FILE)

win32:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=386 $(GO) $(BUILD_TARGET) $(BUILDFLAGS) -o build/win32/$(NAME)-386.exe $(MAIN_SRC_FILE)

darwin: ## Build for OSX
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 $(GO) $(BUILD_TARGET) $(BUILDFLAGS) -o build/darwin/$(NAME) $(MAIN_SRC_FILE)
	chmod +x build/darwin/$(NAME)

.PHONY: test-release
test-release: clean build
	git fetch --tags
	REV=$(REV) BRANCH=$(BRANCH) BUILDDATE=$(BUILD_DATE) GOVERSION=$(GO_VERSION) ROOTPACKAGE=$(ROOT_PACKAGE) VERSION=$(VERSION) goreleaser --config=./.goreleaser.yml --snapshot --skip-publish --rm-dist --skip-validate --debug

.PHONY: release
release: clean build test-slow-integration linux # Release the binary
	git fetch origin refs/tags/v$(VERSION)
	# Don't create a changelog for the distro
	@if [[ -z "${DISTRO}" ]]; then \
		./build/linux/jx step changelog --verbose --header-file=docs/dev/changelog-header.md --version=$(VERSION) --rev=$(PULL_BASE_SHA) --output-markdown=changelog.md --update-release=false; \
		GITHUB_TOKEN=$(GITHUB_ACCESS_TOKEN) REV=$(REV) BRANCH=$(BRANCH) BUILDDATE=$(BUILD_DATE) GOVERSION=$(GO_VERSION) ROOTPACKAGE=$(ROOT_PACKAGE) VERSION=$(VERSION) goreleaser release --config=.goreleaser.yml --rm-dist --release-notes=./changelog.md --skip-validate; \
	else \
		GITHUB_TOKEN=$(GITHUB_ACCESS_TOKEN) REV=$(REV) BRANCH=$(BRANCH) BUILDDATE=$(BUILD_DATE) GOVERSION=$(GO_VERSION) ROOTPACKAGE=$(ROOT_PACKAGE) VERSION=$(VERSION) goreleaser release --config=.goreleaser.yml --rm-dist; \
	fi

.PHONY: release-distro
release-distro:
	@$(MAKE) DISTRO=true release

.PHONY: clean
clean: ## Clean the generated artifacts
	rm -rf build release dist

get-fmt-deps: ## Install test dependencies
	$(GO_NOMOD) get golang.org/x/tools/cmd/goimports

.PHONY: fmt
fmt: importfmt ## Format the code
	$(eval FORMATTED = $(shell $(GO) fmt ./...))
	@if [ "$(FORMATTED)" == "" ]; \
      	then \
      	    echo "All Go files properly formatted"; \
      	else \
      		echo "Fixed formatting for: $(FORMATTED)"; \
      	fi

.PHONY: importfmt
importfmt: get-fmt-deps
	# $(GO_NOMOD) get golang.org/x/tools/cmd/goimports
	@echo "Formatting the imports..."
	goimports -w $(GO_DEPENDENCIES)

.PHONY: lint
lint: ## Lint the code
	./hack/gofmt.sh
	./hack/linter.sh
	./hack/generate.sh

include Makefile.docker
include Makefile.codegen
