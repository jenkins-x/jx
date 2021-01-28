#!/usr/bin/env bash
if [ -d "charts/$REPO_NAME" ]; then source .jx/variables.sh
git config user.name jenkins-x-bot-test
jx gitops helm release
fi
