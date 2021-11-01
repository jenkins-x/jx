#!/bin/bash

set -e -o pipefail

if [ "$DISABLE_LINTER" == "true" ]
then
  exit 0
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if ! [ -x "$(command -v golangci-lint)" ]; then
	echo "Installing GolangCI-Lint"
	${DIR}/install_golint.sh -b $GOPATH/bin v1.42.1
fi

export GO111MODULE=on
golangci-lint run \
  --verbose \
  --build-tags build
