#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

echo "Updating the JX CLI & API reference docs"
./build/linux/jx create docs --verbose
git clone https://github.com/jenkins-x/jx-docs.git
cp -r docs/apidocs/site jx-docs/static/apidocs
cd static/apidocs; git add *
cd content/commands; \
    ../../build/linux/jx create docs; \
    git config credential.helper store; \
    git add *; \
    git commit --allow-empty -a -m "updated jx commands & API docs from $VERSION"; \
    git push origin