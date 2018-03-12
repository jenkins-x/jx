#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'
DRAFT_ROOT="${BASH_SOURCE[0]%/*}/.."

cd "$DRAFT_ROOT"

# install glide
curl -sSL https://github.com/Masterminds/glide/releases/download/v0.12.3/glide-v0.12.3-linux-amd64.tar.gz | tar -xzC /usr/local/bin --strip-components=1 linux-amd64/glide

make bootstrap
