#!/bin/bash

make generate-docs

readonly DOCS_CHANGES=`git diff --name-status master | grep "docs/" | wc -l`

if [ $DOCS_CHANGES -gt 0 ]; then
  echo "There are $DOCS_CHANGES changes in docs, testing site generation..."
  exit 1
fi
