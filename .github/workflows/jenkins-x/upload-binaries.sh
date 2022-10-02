#!/bin/bash

# See https://github.com/actions/checkout/issues/766
git config --global --add safe.directory "$GITHUB_WORKSPACE"
git config --global --get user.name
git config --global --get user.email

echo "setting git user"

git config --global user.name jenkins-x-bot-test
git config --global user.email "jenkins-x@googlegroups.com"

export BRANCH=$(git rev-parse --abbrev-ref HEAD)
export BUILDDATE=$(date)
export REV=$(git rev-parse HEAD)
export GOVERSION="$(go version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')"
export ROOTPACKAGE="github.com/$REPOSITORY"

# Install syft in this script, not sure why using download syft results in goreleaser unable to find the syft executable
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | \
sh -s -- -b /usr/local/bin v0.55.0
chmod +x /usr/local/bin/syft

# Install Go version 1.18.6
curl -O -sSfL https://go.dev/dl/go1.18.6.linux-amd64.tar.gz
rm -rf /usr/local/go/
tar -C /usr/local -xzf go1.18.6.linux-amd64.tar.gz
apk add gcompat 

goreleaser release
