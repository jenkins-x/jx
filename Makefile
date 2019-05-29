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
GITHUB_ACCESS_TOKEN := $(shell cat /builder/home/git-token 2> /dev/null)
FEATURE_FLAG_TOKEN := $(shell cat /builder/home/feature-flag-token 2> /dev/null)
CGO_ENABLED = 0

# set dev version unless VERSION is explicitly set via environment
VERSION ?= $(shell echo "$$(git describe --abbrev=0 --tags 2>/dev/null)-dev+$(REV)" | sed 's/^v//')

BUILDFLAGS :=  -ldflags \
  " -X $(ROOT_PACKAGE)/pkg/version.Version=$(VERSION)\
		-X $(ROOT_PACKAGE)/pkg/version.Revision='$(REV)'\
		-X $(ROOT_PACKAGE)/pkg/version.Branch='$(BRANCH)'\
		-X $(ROOT_PACKAGE)/pkg/version.BuildDate='$(BUILD_DATE)'\
		-X $(ROOT_PACKAGE)/pkg/version.GoVersion='$(GO_VERSION)'"

ifdef DEBUG
BUILDFLAGS := -gcflags "all=-N -l" $(BUILDFLAGS)
endif

ifdef PARALLEL_BUILDS
BUILDFLAGS := -p $(PARALLEL_BUILDS) $(BUILDFLAGS)
TESTFLAGS := -p $(PARALLEL_BUILDS)
else
TESTFLAGS := -p 8
endif

# Various codecov.io variables that are set from the CI envrionment if present, otherwise from locally computed values

CODECOV_NAME := integration

#ARGS is extra args added to the codecov uploader
CODECOV_ARGS := "-n $(CODECOV_NAME)"


ifdef ($(andd $(REPO_NAME), $(REPO_OWNER)),)
CODECOV_SLUG := $(REPO_OWNER)/$(REPO_NAME)
else
CODECOV_SLUG := $(ORG_REPO)
endif

ifdef PULL_PULL_SHA
CODECOV_SHA := $(PULL_PULL_SHA)
else ifdef PULL_BASE_SHA
CODECOV_SHA := $(PULL_BASE_SHA)
else
CODECOV_SHA := $(shell git rev-parse HEAD 2> /dev/null || echo '')
endif

ifdef BRANCH_NAME
CODECOV_BRANCH := $(BRANCH_NAME)
else
CODECOV_BRANCH := $(BRANCH)
endif


ifdef BUILD_NUMBER
CODECOV_ARGS += "-b $(BUILD_NUMBER)"
endif

ifdef PULL_NUMBER
CODECOV_ARGS += "-P $(PULL_NUMBER)"
endif

ifeq ($(JOB_TYPE),postsubmit)
CODECOV_ARGS +="-T v$(VERSION)"
endif

#End Codecov

# support for building a covered jx binary (one with the coverage instrumentation compiled in). The `build-covered`
# target also builds the covered binary explicitly
COVERED_MAIN_SRC_FILE=./cmd/jx
COVERAGE_BUILDFLAGS = -c -coverpkg=./... -covermode=count
COVERAGE_BUILD_TARGET = test
ifdef COVERED_BINARY
BUILDFLAGS += $(COVERAGE_BUILDFLAGS)
BUILD_TARGET = $(COVERAGE_BUILD_TARGET)
MAIN_SRC_FILE = $(COVERED_MAIN_SRC_FILE)
endif

# Build the Jenkins X distribution
ifdef DISTRO
BUILDFLAGS += -ldflags "-X $(ROOT_PACKAGE)/pkg/features.FeatureFlagToken=$(FEATURE_FLAG_TOKEN)"
RELEASE_ORG_REPO := cloudbees/cloudbees-jenkins-x-distro
endif

TEST_PACKAGE ?= ./...
COVERFLAGS=-coverprofile=cover.out --covermode=count --coverpkg=./...

.PHONY: list
list: ## List all make targets
	@$(MAKE) -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

all: build ## Build the binary
full: check ## Build and run the tests
check: build test ## Build and run the tests

print-version: ## Print version
	@echo $(VERSION)

build: $(GO_DEPENDENCIES) ## Build jx binary for current OS
	CGO_ENABLED=$(CGO_ENABLED) $(GO) $(BUILD_TARGET) $(BUILDFLAGS) -o build/$(NAME) $(MAIN_SRC_FILE)

.PHONY: build-covered
build-covered: $(GO_DEPENDENCIES) ## Build jx binary for current OS with coverage instrumentation to build/$(NAME).covered
	CGO_ENABLED=$(CGO_ENABLED) $(GO) $(COVERAGE_BUILD_TARGET) $(BUILDFLAGS) $(COVERAGE_BUILDFLAGS) -o build/$(NAME).covered $(COVERED_MAIN_SRC_FILE)

get-test-deps: ## Install test dependencies
	$(GO_NOMOD) get github.com/axw/gocov/gocov
	$(GO_NOMOD) get -u gopkg.in/matm/v1/gocov-html

tidy-deps: ## Cleans up dependencies
	$(GO) mod tidy
	# mod tidy only takes compile dependencies into account, let's make sure we capture tooling dependencies as well
	@$(MAKE) install-generate-deps

test: ## Run the unit tests
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -p 4 -count=1 $(COVERFLAGS) -failfast -short ./...

test-verbose: ## Run the unit tests in verbose mode
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -v $(COVERFLAGS) -failfast ./...

test-report: get-test-deps test ## Create the test report
	@gocov convert cover.out | gocov report

test-report-html: get-test-deps test ## Create the test report in HTML format
	@gocov convert cover.out | gocov-html > cover.html && open cover.html

test-slow: ## Run unit tests sequentially
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -count=1 $(TESTFLAGS) $(COVERFLAGS) ./...

test-slow-report: get-test-deps test-slow
	@gocov convert cover.out | gocov report

test-slow-report-html: get-test-deps test-slow
	@gocov convert cover.out | gocov-html > cover.html && open cover.html

test-integration: ## Run the integration tests 
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -count=1 -tags=integration  -short ./...

test-integration1:
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -count=1 -tags=integration $(COVERFLAGS) -short ./... -test.v -run $(TEST)

test-rich-integration1:
	@CGO_ENABLED=$(CGO_ENABLED) richgo test -count=1 -tags=integration $(COVERFLAGS) -short -test.v $(TEST_PACKAGE) -run $(TEST)

test-integration-report: get-test-deps test-integration ## Create the integration tests report
	@gocov convert cover.out | gocov report

test-integration-report-html: get-test-deps test-integration
	@gocov convert cover.out | gocov-html > cover.html && open cover.html

test-slow-integration: ## Run the integration tests sequentially
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -p 1 -count=1 -tags=integration  $(COVERFLAGS) ./...

test-slow-integration-report: get-test-deps test-slow-integration
	@gocov convert cover.out | gocov report

test-slow-integration-report-html: get-test-deps test-slow-integration
	@gocov convert cover.out | gocov-html > cover.html && open cover.html

test-soak:
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -p 2 -count=1 -tags soak $(COVERFLAGS) ./...

test1:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./... -test.v -run $(TEST)

testbin:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -c github.com/jenkins-x/jx/pkg/jx/cmd -o build/jx-test

testbin-gits:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -c github.com/jenkins-x/jx/pkg/gits -o build/jx-test-gits

debugtest1: testbin
	cd pkg/jx/cmd && dlv --listen=:2345 --headless=true --api-version=2 exec ../../../build/jx-test -- -test.run $(TEST)

debugtest1gits: testbin-gits
	cd pkg/gits && dlv --log --listen=:2345 --headless=true --api-version=2 exec ../../build/jx-test-gits -- -test.run $(TEST)

inttestbin:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -tags=integration -c github.com/jenkins-x/jx/pkg/jx/cmd -o build/jx-inttest

debuginttest1: inttestbin
	cd pkg/jx/cmd && dlv --listen=:2345 --headless=true --api-version=2 exec ../../../build/jx-inttest -- -test.run $(TEST)

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

.PHONY: release
release: clean build test-slow-integration linux darwin win arm ## Release the binary
	mkdir release
	zip --junk-paths release/$(NAME)-windows-amd64.zip build/win/$(NAME)-windows-amd64.exe README.md LICENSE

	cd ./build/darwin; tar -zcvf ../../release/jx-darwin-amd64.tar.gz jx
	cd ./build/linux; tar -zcvf ../../release/jx-linux-amd64.tar.gz jx
	# Don't build the ARM zip for the distro
	@if [[ -z "${DISTRO}" ]]; then \
		cd ./build/arm; tar -zcvf ../../release/jx-linux-arm.tar.gz jx; \
	fi

	go get -u github.com/progrium/gh-release
	gh-release checksums sha256
	GITHUB_ACCESS_TOKEN=$(GITHUB_ACCESS_TOKEN) gh-release create $(RELEASE_ORG_REPO) $(VERSION) master $(VERSION)

	# Don't create a changelog for the distro
	@if [[ -z "${DISTRO}" ]]; then \
		./build/linux/jx step changelog  --header-file docs/dev/changelog-header.md --version $(VERSION); \
	fi

.PHONY: release-distro
release-distro:
	@$(MAKE) DISTRO=true release

.PHONY: clean
clean: ## Clean the generated artifacts
	rm -rf build release cover.out cover.html

.PHONY: codecov-upload
codecov-upload:
	DOCKER_REPO="$(CODECOV_SLUG)" \
	SOURCE_COMMIT="$(CODECOV_SHA)" \
	SOURCE_BRANCH="$(CODECOV_BRANCH)" \
	bash <(curl -s https://codecov.io/bash) $(CODECOV_ARGS)

fmt: ## Format the code
	$(eval FORMATTED = $(shell $(GO) fmt ./...))
	@if [ "$(FORMATTED)" == "" ]; \
      	then \
      	    echo "All Go files properly formatted"; \
      	else \
      		echo "Fixed formatting for: $(FORMATTED)"; \
      	fi

.PHONY: lint
lint: ## Lint the code
	./hack/run-all-checks.sh

include Makefile.docker
include Makefile.codegen
