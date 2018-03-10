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
#ROOT_PACKAGE := $(shell $(GO) list .)
ROOT_PACKAGE := github.com/jenkins-x/jx
GO_VERSION := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
#PACKAGE_DIRS := pkg cmd
PACKAGE_DIRS := $(shell $(GO) list ./... | grep -v /vendor/)
PKGS := $(shell go list ./... | grep -v /vendor | grep -v generated)


GO_DEPENDENCIES := cmd/*/*.go cmd/*/*/*.go pkg/*/*.go pkg/*/*/*.go pkg/*//*/*/*.go

REV        := $(shell git rev-parse --short HEAD 2> /dev/null  || echo 'unknown')
BRANCH     := $(shell git rev-parse --abbrev-ref HEAD 2> /dev/null  || echo 'unknown')
BUILD_DATE := $(shell date +%Y%m%d-%H:%M:%S)
BUILDFLAGS := -ldflags \
  " -X $(ROOT_PACKAGE)/pkg/version.Version=$(VERSION)\
		-X $(ROOT_PACKAGE)/pkg/version.Revision='$(REV)'\
		-X $(ROOT_PACKAGE)/pkg/version.Branch='$(BRANCH)'\
		-X $(ROOT_PACKAGE)/pkg/version.BuildDate='$(BUILD_DATE)'\
		-X $(ROOT_PACKAGE)/pkg/version.GoVersion='$(GO_VERSION)'"
CGO_ENABLED = 0

VENDOR_DIR=vendor

all: build

check: fmt build test

build: $(GO_DEPENDENCIES)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(BUILDFLAGS) -o build/$(NAME) cmd/jx/jx.go

test: 
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test $(PACKAGE_DIRS) -test.v

#	CGO_ENABLED=$(CGO_ENABLED) $(GO) test github.com/jenkins-x/jx/cmds

full: $(PKGS)

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

	jx step changelog  --header-file docs/dev/changelog-header.md --version $(VERSION)

	updatebot push-version --kind brew jx $(VERSION)
	updatebot push-version --kind docker JX_VERSION $(VERSION)
	updatebot update-loop

	echo "Updating the JX CLI reference docs"
	git clone https://github.com/jenkins-x/jx-docs.git
	cd jx-docs/content/commands; \
		../../../build/linux/jx create docs; \
		git config credential.helper store; \
		git add *; \
		git commit --allow-empty -a -m "updated jx commands from $(VERSION)"; \
		git push origin

clean:
	rm -rf build release

linux:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO) build $(BUILDFLAGS) -o build/$(NAME)-linux-amd64 cmd/jx/jx.go

docker-go: linux Dockerfile.buildgo
	docker build --no-cache -t builder-go -f Dockerfile.buildgo .

docker-maven: linux Dockerfile.maven
	docker build --no-cache -t builder-maven -f Dockerfile.maven .

docker-pipeline: linux
	docker build -t rawlingsj/builder-base:dev . -f Dockerfile-pipeline

.PHONY: release clean arm

preview: linux
	docker build --no-cache -t docker.io/jenkinsxio/builder-maven:SNAPSHOT-JX-$(BRANCH_NAME)-$(BUILD_NUMBER) -f Dockerfile.maven .
	docker push docker.io/jenkinsxio/builder-maven:SNAPSHOT-JX-$(BRANCH_NAME)-$(BUILD_NUMBER)
	docker build --no-cache -t docker.io/jenkinsxio/builder-go:SNAPSHOT-JX-$(BRANCH_NAME)-$(BUILD_NUMBER) -f Dockerfile.buildgo .
	docker push docker.io/jenkinsxio/builder-go:SNAPSHOT-JX-$(BRANCH_NAME)-$(BUILD_NUMBER)
	docker build --no-cache -t docker.io/jenkinsxio/builder-nodejs:SNAPSHOT-JX-$(BRANCH_NAME)-$(BUILD_NUMBER) -f Dockerfile.nodejs .
	docker push docker.io/jenkinsxio/builder-nodejs:SNAPSHOT-JX-$(BRANCH_NAME)-$(BUILD_NUMBER)

FGT := $(GOPATH)/bin/fgt
$(FGT):
	go get github.com/GeertJohan/fgt

GOLINT := $(GOPATH)/bin/golint
$(GOLINT):
	go get github.com/golang/lint/golint

#	@echo "FORMATTING"
#	@$(FGT) gofmt -l=true $(GOPATH)/src/$@/*.go

$(PKGS): $(GOLINT) $(FGT)
	@echo "LINTING"
	@$(FGT) $(GOLINT) $(GOPATH)/src/$@/*.go
	@echo "VETTING"
	@go vet -v $@
	@echo "TESTING"
	@go test -v $@

.PHONY: lint
lint: vendor | $(PKGS) $(GOLINT) # ❷
	@cd $(BASE) && ret=0 && for pkg in $(PKGS); do \
	    test -z "$$($(GOLINT) $$pkg | tee /dev/stderr)" || ret=1 ; \
	done ; exit $$ret


