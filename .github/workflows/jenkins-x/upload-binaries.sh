#!/bin/bash

echo "HOME is $HOME"
echo current git configuration

git config --global --get user.name
git config --global --get user.email

echo "setting git user"

git config --global user.name jenkins-x-bot-test
git config --global user.email "jenkins-x@googlegroups.com"

git add * || true
git commit -a -m "chore: release $VERSION" --allow-empty
git tag -fa v$VERSION -m "Release version $VERSION"
git push origin v$VERSION

export BRANCH=$(git rev-parse --abbrev-ref HEAD)
export BUILDDATE=$(date)
export REV=$(git rev-parse HEAD)
export GOVERSION="1.15"
export ROOTPACKAGE="github.com/$REPOSITORY"

goreleaser release
