#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

CHECKSUMS="./dist/jx-checksums.txt"

if [ -f $CHECKSUMS ]
then
  SHA256=$(grep 'darwin' $CHECKSUMS | cut -d' ' -f1)
  if [ ! -z $SHA256 ]
  then
    ./build/linux/jx step create pr brew --version $VERSION --sha $SHA256 --repo https://github.com/jenkins-x/homebrew-jx.git --src-repo https://github.com/jenkins-x/jx.git
  fi
fi

./build/linux/jx step create pr docker --name JX_VERSION --version $VERSION --repo https://github.com/jenkins-x/jenkins-x-builders.git --repo https://github.com/jenkins-x/jenkins-x-serverless.git --repo https://github.com/jenkins-x/dev-env-base.git
./build/linux/jx step create pr chart --name jx --version $VERSION  --repo https://github.com/jenkins-x/jenkins-x-platform.git
./build/linux/jx step create pr regex --regex "\s*release = \"(.*)\"" --version $VERSION --files config.toml --repo https://github.com/jenkins-x/jx-docs.git
./build/linux/jx step create pr regex --regex "JX_VERSION=(.*)" --version $VERSION --files install-jx.sh --repo https://github.com/jenkins-x/jx-tutorial.git
./build/linux/jx step create pr regex --regex "\s*jxTag:\s*(.*)" --version $VERSION --files prow/values.yaml --repo https://github.com/jenkins-x-charts/prow.git
