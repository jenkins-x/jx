.PHONY: all clean freebsd linux mac pi win current vendor test
.DEFAULT_GOAL := current

BUILD_FLAGS=GO15VENDOREXPERIMENT=1 GORACE="halt_on_error=0" CGO_ENABLED=1

all: clean vendor test freebsd linux mac pi win current

clean:
	@go clean
    
linux:
	@echo "Building for Linux"
	@GOOS=linux GOARCH=amd64 $(BUILD_FLAGS) go build

mac:
	@echo "Building for MacOS X"
	@GOOS=darwin GOARCH=amd64 $(BUILD_FLAGS) go build

freebsd:
	@echo "Building for FreeBSD"
	@GOOS=freebsd GOARCH=amd64 $(BUILD_FLAGS) go build

win:
	@echo "Building for Windows"
	@GOOS=windows GOARCH=amd64 $(BUILD_FLAGS) go build

pi:
	@echo "Building for Raspberry Pi"
	@GOOS=linux GOARCH=arm GOARM=6 $(BUILD_FLAGS) go build

current:
	@$(BUILD_FLAGS) go build

vendor:
	@go get -u github.com/Masterminds/glide
	@glide update

test:
	@$(BUILD_FLAGS) go test -cover -v -timeout 10s -race $$(go list ./...|grep -v vendor)