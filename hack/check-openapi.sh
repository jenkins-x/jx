#!/bin/bash

make generate-openapi

readonly DOCS_CHANGES=`git diff --name-status master | grep "docs/openapi" | wc -l`

if [ $DOCS_CHANGES -gt 0 ]; then
  echo "There are $DOCS_CHANGES changes in docs/openapi, run 'make generate-openapi' and commit the result"
  exit 1
fi
