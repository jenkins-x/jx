#!/usr/bin/env sh

echo "REPO_NAME = $PULL_BASE_SHA"

export PULL_BASE_SHA=$(git rev-parse HEAD)

jx changelog create --verbose --header-file=hack/changelog-header.md --version=v$VERSION --rev=$PULL_BASE_SHA
