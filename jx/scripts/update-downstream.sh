#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Update other repo's dependencies on jx to use the new version - updates repos as specified at .updatebot.yml
updatebot push-version --kind brew jx $(VERSION)
updatebot push-version --kind docker JX_VERSION $(VERSION)
updatebot push-version --kind helm jx $(VERSION)
updatebot push-regex -r "\s*release = \"(.*)\"" -v $(VERSION) config.toml
updatebot push-regex -r "JX_VERSION=(.*)" -v $(VERSION) install-jx.sh
updatebot push-regex -r "\s*jxTag:\s*(.*)" -v $(VERSION) prow/values.yaml

echo "Updating the JX CLI & API reference docs"
./build/linux/jx create client docs --verbose
git clone https://github.com/jenkins-x/jx-docs.git
cp -r docs/apidocs/site jx-docs/static/apidocs
cd jx-docs/static/apidocs; git add *
cd jx-docs/content/commands; \
    ../../../build/linux/jx create docs; \
    git config credential.helper store; \
    git add *; \
    git commit --allow-empty -a -m "updated jx commands & API docs from $(VERSION)"; \
    git push origin