#!/usr/bin/env bash
if [ -d "charts/$REPO_NAME" ]; then source .jx/variables.sh
cd charts/$REPO_NAME
make release; else echo no charts; fi
