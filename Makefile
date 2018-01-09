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
GO := GO15VENDOREXPERIMENT=1 go
VERSION := $(shell cat pkg/version/VERSION)
ROOT_PACKAGE := $(shell $(GO) list .)
GO_VERSION := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
#PACKAGE_DIRS := pkg cmd
PACKAGE_DIRS := $(shell $(GO) list ./... | grep -v /vendor/)

GO_DEPENDENCIES := cmd/*/*.go cmd/*/*/*.go pkg/*/*.go pkg/*/*/*.go pkg/*//*/*/*.go

REV        := $(shell git rev-parse --short HEAD 2> /dev/null  || echo 'unknown')
BRANCH     := $(shell git rev-parse --abbrev-ref HEAD 2> /dev/null  || echo 'unknown')
BUILD_DATE := $(shell date +%Y%m%d-%H:%M:%S)
BUILDFLAGS := -ldflags \
  " -X $(ROOT_PACKAGE)/version.Version=$(VERSION)\
		-X $(ROOT_PACKAGE)/version.Revision='$(REV)'\
		-X $(ROOT_PACKAGE)/version.Branch='$(BRANCH)'\
		-X $(ROOT_PACKAGE)/version.BuildDate='$(BUILD_DATE)'\
		-X $(ROOT_PACKAGE)/version.GoVersion='$(GO_VERSION)'"
CGO_ENABLED = 0

VENDOR_DIR=vendor

all: build

check: fmt build test

build: $(GO_DEPENDENCIES)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(BUILDFLAGS) -o build/$(NAME) cmd/jx/jx.go

test:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test $(PACKAGE_DIRS) -test.v

#	CGO_ENABLED=$(CGO_ENABLED) $(GO) test github.com/jenkins-x/jx/cmds

install: $(GO_DEPENDENCIES)
	GOBIN=${GOPATH}/bin $(GO) install $(BUILDFLAGS) cmd/jx/jx.go

fmt:
	@FORMATTED=`$(GO) fmt $(PACKAGE_DIRS)`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true

arm:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm $(GO) build $(BUILDFLAGS) -o build/$(NAME)-arm cmd/jx/jx.go

win:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$(NAME).exe cmd/jx/jx.go

bootstrap: vendoring

vendoring:
	$(GO) get -u github.com/Masterminds/glide
	GO15VENDOREXPERIMENT=1 glide update --strip-vendor

release: check
	rm -rf build release && mkdir build release
	for os in linux darwin ; do \
		CGO_ENABLED=$(CGO_ENABLED) GOOS=$$os GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$(NAME)-$$os-amd64 cmd/jx/jx.go ; \
	done
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$(NAME)-windows-amd64.exe cmd/jx/jx.go
	zip --junk-paths release/$(NAME)-windows-amd64.zip build/$(NAME)-windows-amd64.exe README.md LICENSE
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm $(GO) build $(BUILDFLAGS) -o build/$(NAME)-linux-arm cmd/jx/jx.go
	chmod +x build/$(NAME)-*-amd64*
	chmod +x build/$(NAME)-*-arm*

	cd ./build; tar -zcvf ../release/jx-darwin-amd64.tgz jx-darwin-amd64
	cd ./build; tar -zcvf ../release/jx-linux-amd64.tgz jx-linux-amd64
	cd ./build; tar -zcvf ../release/jx-linux-arm.tgz jx-linux-arm

	go get -u github.com/progrium/gh-release
	gh-release checksums sha256
	gh-release create jenkins-x/$(NAME) $(VERSION) $(BRANCH) $(VERSION)


clean:
	rm -rf build release

linux:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$(NAME)-linux-amd64 cmd/jx/jx.go

docker: linux
	docker build -t jenkins-x/jx .

.PHONY: release clean arm
