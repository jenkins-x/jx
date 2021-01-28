#!/usr/bin/env bash
if [ -d "charts/$REPO_NAME" ]; then source .jx/variables.sh
echo setting the git user.name
git config user.name jenkins-x-bot-test
git config --global --add user.name jenkins-x-bot-test
jx gitops helm release
fi
