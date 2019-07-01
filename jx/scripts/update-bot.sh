#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Update other repo's dependencies on jx to use the new version - updates repos as specified at .updatebot.yml
../../../build/linux/jx step create pr docker --name JX_VERSION --version $VERSION
../../../build/linux/jx step create pr chart --name jx --version $VERSION
../../../build/linux/jx step create pr regex --regex "\s*release = \"(.*)\"" --version $VERSION --files config.toml
../../../build/linux/jx step create pr regex --regex "JX_VERSION=(.*)" --version $VERSION --files install-jx.sh
../../../build/linux/jx step create pr regex --regex "\s*jxTag:\s*(.*)" --version $VERSION --files prow/values.yaml
