#!/usr/bin/env sh

echo "$CHANGELOG" > changelog.md
jx-updatebot pr --git-credentials --add-changelog changelog.md
