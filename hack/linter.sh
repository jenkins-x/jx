#!/bin/bash

set -e -o pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if ! [ -x "$(command -v golangci-lint)" ]; then
	echo "Installing GolangCI-Lint"
	${DIR}/install_golint.sh -b $GOPATH/bin v1.15.0
fi

golangci-lint run \
	--no-config \
	-E goconst \
	-E goimports \
	-E gocritic \
	-E golint \
	-E interfacer \
	-E maligned \
	-E misspell \
	-E unconvert \
	-E unparam \
	-D errcheck \
  --skip-dirs vendor
