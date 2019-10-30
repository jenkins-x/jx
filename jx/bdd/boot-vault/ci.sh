#!/usr/bin/env bash
set -e
set -x

export GH_USERNAME="jenkins-x-bot-test"
export GH_EMAIL="jenkins-x@googlegroups.com"
export GH_OWNER="jenkins-x-bot-test"

export REPORTS_DIR="${BASE_WORKSPACE}/build/reports"
export GINKGO_ARGS="-v"

# fix broken `BUILD_NUMBER` env var
export BUILD_NUMBER="$BUILD_ID"

BOOT_CONFIG_FROM_VERSION_STREAM=v$(jx step get dependency-version --host=github.com --owner=jenkins-x --repo=jenkins-x-boot-config | sed 's/.*: \(.*\)/\1/')

JX_HOME="/tmp/jxhome"
KUBECONFIG="/tmp/jxhome/config"

mkdir -p $JX_HOME

jx --version
jx step git credentials

gcloud auth activate-service-account --key-file $GKE_SA

# lets setup git 
git config --global --add user.name JenkinsXBot
git config --global --add user.email jenkins-x@googlegroups.com

echo "running the BDD tests with JX_HOME = $JX_HOME"

sed -e s/\$VERSION/${VERSION_PREFIX}${VERSION}/g -e s/\$CODECOV_TOKEN/${CODECOV_TOKEN}/g boot-vault.platform.yaml.template > boot-vault.platform.yaml
sed -e s/\$VERSION/${VERSION_PREFIX}${VERSION}/g -e s/\$CODECOV_TOKEN/${CODECOV_TOKEN}/g boot-vault.prow.yaml.template > boot-vault.prow.yaml

# setup jx boot parameters
export JX_VALUE_ADMINUSER_PASSWORD="$JENKINS_PASSWORD"
export JX_VALUE_PIPELINEUSER_USERNAME="$GH_USERNAME"
export JX_VALUE_PIPELINEUSER_EMAIL="$GH_EMAIL"
export JX_VALUE_PIPELINEUSER_TOKEN="$GH_ACCESS_TOKEN"
export JX_VALUE_PROW_HMACTOKEN="$GH_ACCESS_TOKEN"

# TODO temporary hack until the batch mode in jx is fixed...
export JX_BATCH_MODE="true"

git clone https://github.com/jenkins-x/jenkins-x-boot-config.git boot-source
cd boot-source
git fetch --tags
git checkout ${BOOT_CONFIG_FROM_VERSION_STREAM}
cp ../jx/bdd/boot-vault/jx-requirements.yml .
cp ../jx/bdd/boot-vault/parameters.yaml env

cp env/jenkins-x-platform/values.tmpl.yaml tmp.yaml
cat tmp.yaml ../boot-vault.platform.yaml > env/jenkins-x-platform/values.tmpl.yaml
rm tmp.yaml

cp env/prow/values.tmpl.yaml tmp.yaml
cat tmp.yaml ../boot-vault.prow.yaml > env/prow/values.tmpl.yaml
rm tmp.yaml

# TODO hack until we fix boot to do this too!
helm init --client-only
helm repo add jenkins-x https://storage.googleapis.com/chartmuseum.jenkins-x.io

jx step bdd \
    --versions-repo https://github.com/jenkins-x/jenkins-x-versions.git \
    --config ../jx/bdd/boot-vault/cluster.yaml \
    --gopath /tmp \
    --git-provider=github \
    --git-username $GH_USERNAME \
    --git-owner $GH_OWNER \
    --git-api-token $GH_ACCESS_TOKEN \
    --default-admin-password $JENKINS_PASSWORD \
    --no-delete-app \
    --no-delete-repo \
    --tests test-quickstart-golang-http \
    --tests test-app-lifecycle
