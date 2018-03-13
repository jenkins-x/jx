#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'
DRAFT_ROOT="${BASH_SOURCE[0]%/*}/.."

cd "$DRAFT_ROOT"

# Skip on pull request builds
if [[ -n "${CIRCLE_PR_NUMBER:-}" ]]; then
  exit
fi

: ${AZURE_CONTAINER:?"AZURE_CONTAINER environment variable is not set"}
: ${AZURE_STORAGE_ACCOUNT:?"AZURE_STORAGE_ACCOUNT environment variable is not set"}
: ${AZURE_STORAGE_KEY:?"AZURE_STORAGE_KEY environment variable is not set"}

VERSION=
if [[ -n "${CIRCLE_TAG:-}" ]]; then
  VERSION="${CIRCLE_TAG}"
elif [[ "${CIRCLE_BRANCH:-}" == "master" ]]; then
  VERSION="canary"
else
  exit 1
fi

echo "Installing Azure components"
AZCLI_VERSION=2.0.15
apt-get update && apt-get install -yq python-pip
pip install --disable-pip-version-check --no-cache-dir azure-cli==${AZCLI_VERSION}

echo "Building Draft binaries"
make build-cross
VERSION="${VERSION}" make dist checksum
if [[ -n "${CIRCLE_TAG:-}" ]]; then
  VERSION="latest" make dist checksum
fi

echo "Pushing binaries to Azure Blob Storage"
az storage blob upload-batch --source _dist/ --destination "${AZURE_CONTAINER}" --pattern *.tar.gz*
