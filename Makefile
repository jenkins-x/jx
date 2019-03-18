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

SHELL := /bin/bash
NAME := jx
GO := GO111MODULE=on go
GO_NOMOD :=GO111MODULE=off go
REV := $(shell git rev-parse --short HEAD 2> /dev/null || echo 'unknown')
#ROOT_PACKAGE := $(shell $(GO) list .)
ROOT_PACKAGE := github.com/jenkins-x/jx
GO_VERSION := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
PKGS := $(shell go list ./... | grep -v generated)
GO_DEPENDENCIES := cmd/*/*.go cmd/*/*/*.go pkg/*/*.go pkg/*/*/*.go pkg/*//*/*/*.go

BRANCH     := $(shell git rev-parse --abbrev-ref HEAD 2> /dev/null  || echo 'unknown')
BUILD_DATE := $(shell date +%Y%m%d-%H:%M:%S)
PEGOMOCK_SHA := $(shell go mod graph | grep pegomock | sed -n -e 's/^.*-//p')
PEGOMOCK_PACKAGE := github.com/petergtz/pegomock/
CGO_ENABLED = 0

all: build
full: check
check: lint vet build test

version:
ifeq (,$(wildcard pkg/version/VERSION))
TAG := $(shell git fetch --all -q 2>/dev/null && git describe --abbrev=0 --tags 2>/dev/null)
ON_EXACT_TAG := $(shell git name-rev --name-only --tags --no-undefined HEAD 2>/dev/null | sed -n 's/^\([^^~]\{1,\}\)\(\^0\)\{0,1\}$$/\1/p')
VERSION := $(shell [ -z "$(ON_EXACT_TAG)" ] && echo "$(TAG)-dev+$(REV)" | sed 's/^v//' || echo "$(TAG)" | sed 's/^v//' )
else
VERSION := $(shell cat pkg/version/VERSION)
endif
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

print-version: version
	@echo $(VERSION)

build: $(GO_DEPENDENCIES) version
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(BUILDFLAGS) -o build/$(NAME) cmd/jx/jx.go

get-test-deps:
	$(GO_NOMOD) get github.com/axw/gocov/gocov
	$(GO_NOMOD) get -u gopkg.in/matm/v1/gocov-html

test:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -p 1 -count=1 -coverprofile=cover.out \
	-failfast -short ./...

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

docker-test:
	docker run --rm -v $(shell pwd):/go/src/github.com/jenkins-x/jx golang:1.11 sh -c "rm /usr/bin/git && cd /go/src/github.com/jenkins-x/jx && make test"

docker-test-slow:
	docker run --rm -v $(shell pwd):/go/src/github.com/jenkins-x/jx golang:1.11 sh -c "rm /usr/bin/git && cd /go/src/github.com/jenkins-x/jx && make test-slow"

# EASY WAY TO TEST IF YOUR TEST SHOULD BE A UNIT OR INTEGRATION TEST
docker-test-integration:
	docker run --rm -v $(shell pwd):/go/src/github.com/jenkins-x/jx golang:1.11 sh -c "rm /usr/bin/git && cd /go/src/github.com/jenkins-x/jx && make test-integration"

# EASY WAY TO TEST IF YOUR SLOW TEST SHOULD BE A UNIT OR INTEGRATION TEST
docker-test-slow-integration:
	docker run --rm -v $(shell pwd):/go/src/github.com/jenkins-x/jx golang:1.11 sh -c "rm /usr/bin/git && cd /go/src/github.com/jenkins-x/jx && make test-slow-integration"

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

install: $(GO_DEPENDENCIES) version
	GOBIN=${GOPATH}/bin $(GO) install $(BUILDFLAGS) cmd/jx/jx.go

fmt:
	@FORMATTED=`$(GO) fmt ./...`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true

arm: version
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm $(GO) build $(BUILDFLAGS) -o build/$(NAME)-arm cmd/jx/jx.go

win: version
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$(NAME).exe cmd/jx/jx.go

darwin: version
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/darwin/jx cmd/jx/jx.go

# sleeps for about 30 mins
sleep:
	sleep 2000

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
	gh-release create jenkins-x/$(NAME) $(VERSION) master $(VERSION)

	./build/linux/jx step changelog  --header-file docs/dev/changelog-header.md --version $(VERSION)

clean:
	rm -rf build release cover.out cover.html

linux: version
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/linux/jx cmd/jx/jx.go

docker: linux
	docker build -t rawlingsj/jx:dev207 .
	docker push rawlingsj/jx:dev207

docker-go: linux Dockerfile.builder-go
	docker build --no-cache -t builder-go -f Dockerfile.builder-go .

docker-maven: linux Dockerfile.builder-maven
	docker build --no-cache -t builder-maven -f Dockerfile.builder-maven .

jenkins-maven: linux Dockerfile.jenkins-maven
	docker build --no-cache -t jenkins-maven -f Dockerfile.jenkins-maven .

docker-base: linux
	docker build -t rawlingsj/builder-base:dev16 . -f Dockerfile.builder-base

docker-pull:
	docker images | grep -v REPOSITORY | awk '{print $$1}' | uniq -u | grep jenkinsxio | awk '{print $$1":latest"}' | xargs -L1 docker pull

docker-build-and-push:
	docker build --no-cache -t $(DOCKER_HUB_USER)/jx:dev .
	docker push $(DOCKER_HUB_USER)/jx:dev
	docker build --no-cache -t $(DOCKER_HUB_USER)/builder-base:dev -f Dockerfile.builder-base .
	docker push $(DOCKER_HUB_USER)/builder-base:dev
	docker build --no-cache -t $(DOCKER_HUB_USER)/builder-maven:dev -f Dockerfile.builder-maven .
	docker push $(DOCKER_HUB_USER)/builder-maven:dev
	docker build --no-cache -t $(DOCKER_HUB_USER)/builder-go:dev -f Dockerfile.builder-go .
	docker push $(DOCKER_HUB_USER)/builder-go:dev

docker-dev: build linux docker-pull docker-build-and-push

docker-dev-no-pull: build linux docker-build-and-push

docker-dev-all: build linux docker-pull docker-build-and-push
	docker build --no-cache -t $(DOCKER_HUB_USER)/builder-gradle:dev -f Dockerfile.builder-gradle .
	docker push $(DOCKER_HUB_USER)/builder-gradle:dev
	docker build --no-cache -t $(DOCKER_HUB_USER)/builder-rust:dev -f Dockerfile.builder-rust .
	docker push $(DOCKER_HUB_USER)/builder-rust:dev
	docker build --no-cache -t $(DOCKER_HUB_USER)/builder-scala:dev -f Dockerfile.builder-scala .
	docker push $(DOCKER_HUB_USER)/builder-scala:dev
	docker build --no-cache -t $(DOCKER_HUB_USER)/builder-swift:dev -f Dockerfile.builder-swift .
	docker push $(DOCKER_HUB_USER)/builder-swift:dev
	docker build --no-cache -t $(DOCKER_HUB_USER)/builder-terraform:dev -f Dockerfile.builder-terraform .
	docker push $(DOCKER_HUB_USER)/builder-terraform:dev
	docker build --no-cache -t $(DOCKER_HUB_USER)/builder-nodejs:dev -f Dockerfile.builder-nodejs .
	docker push $(DOCKER_HUB_USER)/builder-nodejs:dev
	docker build --no-cache -t $(DOCKER_HUB_USER)/builder-python:dev -f Dockerfile.builder-python .
	docker push $(DOCKER_HUB_USER)/builder-python:dev
	docker build --no-cache -t $(DOCKER_HUB_USER)/builder-python2:dev -f Dockerfile.builder-python2 .
	docker push $(DOCKER_HUB_USER)/builder-python2:dev
	docker build --no-cache -t $(DOCKER_HUB_USER)/builder-ruby:dev -f Dockerfile.builder-ruby .
	docker push $(DOCKER_HUB_USER)/builder-ruby:dev

# Generate go code using generate directives in files and kubernetes code generation
# Anything generated by this target should be checked in
generate: generate-mocks generate-openapi generate-client fmt
	@ECHO "Generation complete"

generate-mocks:
	@echo "Generating Mocks using pegomock"
	$(GO_NOMOD) get -d $(PEGOMOCK_PACKAGE)...
	cd $(GOPATH)/src/$(PEGOMOCK_PACKAGE); git checkout master; git fetch origin; git branch -f jx $(PEGOMOCK_SHA); \
	git checkout jx; $(GO_NOMOD) install ./pegomock
	$(GO) generate ./...

generate-client:
	@echo "Generating Kubernetes Clients for pkg/apis in pkg/client for jenkins.io:v1"
	jx create client go --output-package=pkg/client --input-package=pkg/apis --group-with-version=jenkins.io:v1

# Generated docs are not checked in
generate-docs:
	@echo "Generating HTML docs for Kubernetes Clients"
	jx create client docs

generate-openapi:
	@echo "Generating OpenAPI structs for Kubernetes Clients"
	jx create client openapi --output-package=pkg/client --input-package=pkg/apis --group-with-version=jenkins.io:v1

richgo:
	go get -u github.com/kyoh86/richgo

.PHONY: release clean arm

FGT := $(GOPATH)/bin/fgt
$(FGT):
	$(GO_NOMOD) get github.com/GeertJohan/fgt


GOLINT := $(GOPATH)/bin/golint
$(GOLINT):
	$(GO_NOMOD) get github.com/golang/lint/golint

.PHONY: lint
lint: $(GOLINT)
	@echo "--> linting code with 'go lint' tool"
	$(GOLINT) -min_confidence 1.1 ./...

.PHONY: vet
vet: tools.govet
	@echo "--> checking code correctness with 'go vet' tool"
	@go vet ./... || true


tools.govet:
	@go tool vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		echo "--> installing govet"; \
		$(GO_NOMOD) get golang.org/x/tools/cmd/vet; \
	fi

GOSEC := $(GOPATH)/bin/gosec
$(GOSEC):
	$(GO_NOMOD) get github.com/securego/gosec/cmd/gosec/...

.PHONY: sec
sec: $(GOSEC)
	@echo "SECURITY"
	@mkdir -p scanning
	$(GOSEC) -fmt=yaml -out=scanning/results.yaml ./...


