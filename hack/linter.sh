#!/bin/bash

set -e -o pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if ! [ -x "$(command -v golangci-lint)" ]; then
	echo "Installing GolangCI-Lint"
	${DIR}/install_golint.sh -b $GOPATH/bin v1.16.0
fi

golangci-lint run \
	--no-config \
    --disable-all \
	-E misspell \
	-E unconvert \
    -E deadcode \
    -E unconvert \
    -E errcheck \
    -E gosec \
    --skip-dirs vendor \
    --deadline 5m0s \
    --verbose

# -E gofmt \
# -E structcheck \
# -E varcheck \
# -E govet \
# -E interfacer \
# -E unparam \
# -E megacheck \
# -E goconst \
# -E typecheck \
# -E ineffassign \
# -E goimports \
# -E golint
# -E unparam
# -E gocritic
# -E maligned
