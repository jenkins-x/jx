#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'
DRAFT_ROOT="${BASH_SOURCE[0]%/*}/.."

cd "$DRAFT_ROOT"

run_unit_test() {
  echo "Running unit tests"
  make test-unit
}

run_style_check() {
  echo "Running style checks"
  make test-lint
}

run_unit_test
run_style_check
