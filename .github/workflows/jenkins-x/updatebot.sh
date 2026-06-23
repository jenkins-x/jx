#!/usr/bin/env sh

git config --global --add safe.directory "$GITHUB_WORKSPACE"
echo "$CHANGELOG" > changelog.md
jx-updatebot pr --git-credentials --add-changelog changelog.md
