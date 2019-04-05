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
GO := GO111MODULE=on go
GO_NOMOD :=GO111MODULE=off go
REV := $(shell git rev-parse --short HEAD 2> /dev/null || echo 'unknown')
ROOT_PACKAGE := github.com/jenkins-x/jx
GO_VERSION := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_DEPENDENCIES := $(call rwildcard,pkg/,*.go) $(call rwildcard,cmd/jx/,*.go)

BRANCH     := $(shell git rev-parse --abbrev-ref HEAD 2> /dev/null  || echo 'unknown')
BUILD_DATE := $(shell date +%Y%m%d-%H:%M:%S)
PEGOMOCK_SHA := $(shell $(GO) mod graph | grep pegomock | sed -n -e 's/^.*-//p')
GITHUB_ACCESS_TOKEN := $(shell cat /builder/home/git-token 2> /dev/null)
PEGOMOCK_PACKAGE := github.com/petergtz/pegomock/
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

TEST_PACKAGE ?= ./...

all: build
full: check
check: build test

fail:
	exit 1

print-version:
	@echo $(VERSION)

build: $(GO_DEPENDENCIES)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(BUILDFLAGS) -o build/$(NAME) cmd/jx/jx.go

get-test-deps:
	$(GO_NOMOD) get github.com/axw/gocov/gocov
	$(GO_NOMOD) get -u gopkg.in/matm/v1/gocov-html

test:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -p 1 -count=1 -coverprofile=cover.out \
	-failfast -short ./...

test-verbose:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -v -coverprofile=cover.out -failfast ./...

test-report: get-test-deps test
	@gocov convert cover.out | gocov report

test-report-html: get-test-deps test
	@gocov convert cover.out | gocov-html > cover.html && open cover.html

test-slow:
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -count=1 $(TESTFLAGS) -coverprofile=cover.out ./...

test-slow-report: get-test-deps test-slow
	@gocov convert cover.out | gocov report

test-slow-report-html: get-test-deps test-slow
	@gocov convert cover.out | gocov-html > cover.html && open cover.html

test-integration:
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -count=1 -tags=integration -coverprofile=cover.out -short ./...

test-integration1:
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -count=1 -tags=integration -coverprofile=cover.out -short ./... -test.v -run $(TEST)

test-rich-integration1:
	@CGO_ENABLED=$(CGO_ENABLED) richgo test -count=1 -tags=integration -coverprofile=cover.out -short -test.v $(TEST_PACKAGE) -run $(TEST)

test-integration-report: get-test-deps test-integration
	@gocov convert cover.out | gocov report

test-integration-report-html: get-test-deps test-integration
	@gocov convert cover.out | gocov-html > cover.html && open cover.html

test-slow-integration:
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -p 2 -count=1 -tags=integration -coverprofile=cover.out ./...

test-slow-integration-report: get-test-deps test-slow-integration
	@gocov convert cover.out | gocov report

test-slow-integration-report-html: get-test-deps test-slow-integration
	@gocov convert cover.out | gocov-html > cover.html && open cover.html

test-soak:
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) test -p 2 -count=1 -tags soak -coverprofile=cover.out ./...

#	CGO_ENABLED=$(CGO_ENABLED) $(GO) test github.com/jenkins-x/jx/cmds
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

install: $(GO_DEPENDENCIES)
	GOBIN=${GOPATH}/bin $(GO) install $(BUILDFLAGS) cmd/jx/jx.go

fmt:
	@FORMATTED=`$(GO) fmt ./...`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true

linux:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/linux/jx cmd/jx/jx.go

arm:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm $(GO) build $(BUILDFLAGS) -o build/$(NAME)-arm cmd/jx/jx.go

win:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$(NAME).exe cmd/jx/jx.go

win32:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=386 $(GO) build $(BUILDFLAGS) -o build/$(NAME)-386.exe cmd/jx/jx.go

darwin:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/darwin/jx cmd/jx/jx.go

# sleeps for about 30 mins
sleep:
	sleep 2000

.PHONY: release
release: check
	rm -rf build release && mkdir build release
	for os in linux darwin ; do \
		CGO_ENABLED=$(CGO_ENABLED) GOOS=$$os GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$$os/$(NAME) cmd/jx/jx.go ; \
	done
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$(NAME)-windows-amd64.exe cmd/jx/jx.go
	zip --junk-paths release/$(NAME)-windows-amd64.zip build/$(NAME)-windows-amd64.exe README.md LICENSE
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm $(GO) build $(BUILDFLAGS) -o build/arm/$(NAME) cmd/jx/jx.go


	chmod +x build/darwin/$(NAME)
	chmod +x build/linux/$(NAME)
	chmod +x build/arm/$(NAME)

	cd ./build/darwin; tar -zcvf ../../release/jx-darwin-amd64.tar.gz jx
	cd ./build/linux; tar -zcvf ../../release/jx-linux-amd64.tar.gz jx
	cd ./build/arm; tar -zcvf ../../release/jx-linux-arm.tar.gz jx

	go get -u github.com/progrium/gh-release
	gh-release checksums sha256
	GITHUB_ACCESS_TOKEN=$(GITHUB_ACCESS_TOKEN) gh-release create jenkins-x/$(NAME) $(VERSION) master $(VERSION)

	./build/linux/jx step changelog  --header-file docs/dev/changelog-header.md --version $(VERSION)

clean:
	rm -rf build release cover.out cover.html


include Makefile.docker
include Makefile.codegen

richgo:
	go get -u github.com/kyoh86/richgo

FGT := $(GOPATH)/bin/fgt
$(FGT):
	$(GO_NOMOD) get github.com/GeertJohan/fgt

.PHONY: lint
lint:
	./hack/run-all-checks.sh
