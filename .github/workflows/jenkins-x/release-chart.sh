#!/usr/bin/env bash
if [ -d "charts/$REPO_NAME" ]; then source .jx/variables.sh
jx gitops helm release
fi
