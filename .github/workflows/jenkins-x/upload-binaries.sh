#!/bin/bash
source .jx/variables.sh

git add * || true
git commit -a -m "chore: release $VERSION" --allow-empty
git tag -fa v$VERSION -m "Release version $VERSION"
git push origin v$VERSION

export BRANCH=$(git rev-parse --abbrev-ref HEAD)
export BUILDDATE=$(date)
export REV=$(git rev-parse HEAD)
export GOVERSION="1.15"
export ROOTPACKAGE="github.com/$REPO_OWNER/$REPO_NAME"
goreleaser release
