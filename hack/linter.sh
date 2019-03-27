#!/bin/bash

set -e -o pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if ! [ -x "$(command -v golangci-lint)" ]; then
	echo "Installing GolangCI-Lint"
	${DIR}/install_golint.sh -b $GOPATH/bin v1.15.0
fi

golangci-lint run \
	--no-config \
	-E goimports \
	-E misspell \
	-E unconvert \
	-D errcheck \
    -D ineffassign \
    -D deadcode \
    -D govet \
    -D varcheck \
    -D structcheck \
  --skip-dirs vendor \
  --deadline 10m0s

# ? deadcode / unused

# -E goconst
# -E golint
# -E gosec
# -E unparam
# -E gocritic
# -E interfacer
# -E maligned
