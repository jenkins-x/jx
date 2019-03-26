#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
RESET='\033[0m'

echo "Running validation scripts..."
#scripts=(
#    "hack/boilerplate.sh"
#    "hack/gofmt.sh"
#    "hack/linter.sh"
#    "hack/dep.sh"
#    "hack/check-samples.sh"
#    "hack/check-docs.sh"
#)
scripts=(
    "hack/gofmt.sh"
)
fail=0
for s in "${scripts[@]}"; do
    echo "RUN ${s}"
    set +e
    ./$s
    result=$?
    set -e
    if [[ $result  -eq 1 ]]; then
        echo -e "${RED}FAILED${RESET} ${s}"
        fail=1
    else
        echo -e "${GREEN}PASSED${RESET} ${s}"
    fi
done
exit $fail
