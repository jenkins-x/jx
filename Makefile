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

REPORTS_DIR=$(BUILD_TARGET)/reports

GOTEST := go test
# If available, use gotestsum which provides more comprehensive output
# This is used in the CI builds
ifneq (, $(shell which gotestsum 2> /dev/null))
GOTESTSUM_FORMAT ?= standard-quiet
GOTEST := gotestsum --junitfile $(REPORTS_DIR)/integration.junit.xml --format $(GOTESTSUM_FORMAT) --
endif


# set dev version unless VERSION is explicitly set via environment
VERSION ?= $(shell echo "$$(git describe --abbrev=0 --tags 2>/dev/null)-dev+$(REV)" | sed 's/^v//')

# Various codecov.io variables that are set from the CI envrionment if present, otherwise from locally computed values

CODECOV_NAME ?= integration

#ARGS is extra args added to the codecov uploader
CODECOV_ARGS := -n $(CODECOV_NAME) -F $(CODECOV_NAME) -s $(REPORTS_DIR)


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


ifdef BUILD_NUMBER
CODECOV_ARGS += -b $(BUILD_NUMBER)
endif

ifdef BRANCH_NAME
CODECOV_BRANCH := $(BRANCH_NAME)
else
CODECOV_BRANCH := $(BRANCH)
endif

ifdef PULL_NUMBER
CODECOV_ARGS += -P $(PULL_NUMBER)
CODECOV_BRANCH := $(PULL_BASE_REF)
endif

ifeq ($(JOB_TYPE),postsubmit)
CODECOV_TAG := v$(VERSION)
CODECOV_ARGS += -T $(CODECOV_TAG)
endif

#End Codecov

BUILDFLAGS :=  -ldflags \
  " -X $(ROOT_PACKAGE)/pkg/version.Version=$(VERSION)\
		-X $(ROOT_PACKAGE)/pkg/version.Revision='$(REV)'\
		-X $(ROOT_PACKAGE)/pkg/version.Branch='$(BRANCH)'\
		-X $(ROOT_PACKAGE)/pkg/version.BuildDate='$(BUILD_DATE)'\
		-X $(ROOT_PACKAGE)/pkg/version.GoVersion='$(GO_VERSION)'\
		-X $(ROOT_PACKAGE)/cmd/jx/codecov.Flag=$(CODECOV_NAME)\
		-X $(ROOT_PACKAGE)/cmd/jx/codecov.Slug=$(CODECOV_SLUG)\
		-X $(ROOT_PACKAGE)/cmd/jx/codecov.Branch=$(CODECOV_BRANCH)\
		-X $(ROOT_PACKAGE)/cmd/jx/codecov.Sha=$(CODECOV_SHA)\
		-X $(ROOT_PACKAGE)/cmd/jx/codecov.BuildNumber=$(BUILD_NUMBER)\
		-X $(ROOT_PACKAGE)/cmd/jx/codecov.PullRequestNumber=$(PULL_NUMBER)\
		-X $(ROOT_PACKAGE)/cmd/jx/codecov.Tag=$(CODECOV_TAG)"

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

# Build the Jenkins X distribution
ifdef DISTRO
BUILDFLAGS += -ldflags "-X $(ROOT_PACKAGE)/pkg/features.FeatureFlagToken=$(FEATURE_FLAG_TOKEN)"
RELEASE_ORG_REPO := cloudbees/cloudbees-jenkins-x-distro
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
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

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

test: make-reports-dir ## Run the unit tests
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -count=1 $(COVERFLAGS) -failfast -short ./...

test-report: make-reports-dir get-test-deps test ## Create the test report
	@gocov convert $(COVER_OUT) | gocov report

test-report-html: make-reports-dir get-test-deps test ## Create the test report in HTML format
	@gocov convert $(COVER_OUT) | gocov-html > $(REPORTS_DIR)/cover.html && open $(REPORTS_DIR)/cover.html

test-verbose: make-reports-dir ## Run the unit tests in verbose mode
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -v $(COVERFLAGS) -failfast ./...

test-slow-report: get-test-deps test-slow make-reports-dir
	@gocov convert $(COVER_OUT) | gocov report

test-slow: make-reports-dir ## Run unit tests sequentially
	@CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -count=1 $(COVERFLAGS) ./...

test-slow-report-html: make-reports-dir get-test-deps test-slow
	@gocov convert $(COVER_OUT) | gocov-html > $(REPORTS_DIR)/cover.html && open $(REPORTS_DIR)/cover.html

test-integration: get-test-deps## Run the integration tests
	@CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -count=1 -tags=integration  -short ./...

test-integration1: make-reports-dir
	@CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -count=1 -tags=integration $(COVERFLAGS) -short ./... -test.v -run $(TEST)

test-rich-integration1: make-reports-dir
	@CGO_ENABLED=$(CGO_ENABLED) richgo test -count=1 -tags=integration $(COVERFLAGS) -short -test.v $(TEST_PACKAGE) -run $(TEST)

test-integration-report: make-reports-dir get-test-deps test-integration ## Create the integration tests report
	@gocov convert $(COVER_OUT) | gocov report

test-integration-report-html: make-reports-dir get-test-deps test-integration
	@gocov convert $(COVER_OUT) | gocov-html > $(REPORTS_DIR)/cover.html && open $(REPORTS_DIR)/cover.html


test-slow-integration: make-reports-dir ## Run the integration tests sequentially
	@CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -count=1 -tags=integration $(COVERFLAGS) ./...

test-slow-integration-report: make-reports-dir get-test-deps test-slow-integration
	@gocov convert $(COVER_OUT) | gocov report

test-slow-integration-report-html: make-reports-dir get-test-deps test-slow-integration
	@gocov convert $(COVER_OUT) | gocov-html > $(REPORTS_DIR)/cover.html && open $(REPORTS_DIR)/cover.html

test-soak: make-reports-dir get-test-deps
	@CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -count=1 -tags soak $(COVERFLAGS) ./...

test1: get-test-deps make-reports-dir
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) ./... -test.v -run $(TEST)

testbin: get-test-deps make-reports-dir
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -c github.com/jenkins-x/jx/pkg/jx/cmd -o build/jx-test

testbin-gits: get-test-deps make-reports-dir
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -c github.com/jenkins-x/jx/pkg/gits -o build/jx-test-gits

debugtest1: testbin
	cd pkg/jx/cmd && dlv --listen=:2345 --headless=true --api-version=2 exec ../../../build/jx-test -- -test.run $(TEST)

debugtest1gits: testbin-gits
	cd pkg/gits && dlv --log --listen=:2345 --headless=true --api-version=2 exec ../../build/jx-test-gits -- -test.run $(TEST)

inttestbin: get-test-deps
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -tags=integration -c github.com/jenkins-x/jx/pkg/jx/cmd -o build/jx-inttest

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
	@if [[ -z "${DISTRO}" ]]; then \
		cd ./build/arm; tar -zcvf ../../release/jx-linux-arm.tar.gz jx; \
	fi

	go get -u github.com/progrium/gh-release
	gh-release checksums sha256
	GITHUB_ACCESS_TOKEN=$(GITHUB_ACCESS_TOKEN) gh-release create $(RELEASE_ORG_REPO) $(VERSION) master $(VERSION)

	@if [[ -z "${DISTRO}" ]]; then \
		./build/linux/jx step changelog  --verbose --header-file docs/dev/changelog-header.md --version $(VERSION) --rev $(PULL_BASE_SHA); \
	fi

.PHONY: release-distro
release-distro:
	@$(MAKE) DISTRO=true release

.PHONY: clean
clean: ## Clean the generated artifacts
	rm -rf build release

.PHONY: codecov-upload
codecov-upload:
	DOCKER_REPO="$(CODECOV_SLUG)" \
	SOURCE_COMMIT="$(CODECOV_SHA)" \
	SOURCE_BRANCH="$(CODECOV_BRANCH)" \
	bash <(curl -s https://codecov.io/bash) $(CODECOV_ARGS)

.PHONY: codecov-validate
codecov-validate:
	./jx/scripts/codecov-validate.sh


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
