#!/usr/bin/env sh

if [ -n "$GITHUB_WORKSPACE" ] && [ -d "$GITHUB_WORKSPACE" ]; then
  cd "$GITHUB_WORKSPACE"
fi

echo "$CHANGELOG" > changelog.md
jx-updatebot pr --git-credentials --add-changelog changelog.md
