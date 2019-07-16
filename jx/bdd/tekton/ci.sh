#!/usr/bin/env bash
set -e

echo "verifying Pull Request"

export GH_USERNAME="jenkins-x-bot-test"
export GH_OWNER="cb-kubecd"

export GH_CREDS_PSW="$(jx step credential -s jenkins-x-bot-test-github | sed -e 's/PASS//' -e 's/coverage: [0-9\.]*% of statements in [\w\.\/]*//' | tr -d [:space:])"
export JENKINS_CREDS_PSW="$(jx step credential -s  test-jenkins-user | sed -e 's/PASS//' -e 's/coverage: [0-9\.]*% of statements in [\w\.\/]*//' | tr -d [:space:])"
export GKE_SA="$(jx step credential -k bdd-credentials.json -s bdd-secret -f sa.json | sed -e 's/PASS//' -e 's/coverage: [0-9\.]*% of statements in [\w\.\/]*//' | tr -d [:space:])"
export REPORTS_DIR="${BASE_WORKSPACE}/build/reports"
export GINKGO_ARGS="-v"

# fix broken `BUILD_NUMBER` env var
export BUILD_NUMBER="$BUILD_ID"

JX_HOME="/tmp/jxhome"
KUBECONFIG="/tmp/jxhome/config"

mkdir -p $JX_HOME

# Disable coverage for jx version as we don't validate the output at all
COVER_JX_BINARY=false jx version
jx step git credentials

gcloud auth activate-service-account --key-file $GKE_SA

sed -e s/\$VERSION/${VERSION_PREFIX}${VERSION}/g -e s/\$CODECOV_TOKEN/${CODECOV_TOKEN}/g myvalues.yaml.tekton.template > myvalues.yaml

#echo the myvalues.yaml file is:
#cat myvalues.yaml

# lets setup git
git config --global --add user.name JenkinsXBot
git config --global --add user.email jenkins-x@googlegroups.com

echo "running the BDD tests with JX_HOME = $JX_HOME"
jx step bdd --versions-repo https://github.com/jenkins-x/jenkins-x-versions.git --config jx/bdd/tekton/cluster.yaml --gopath /tmp  --git-provider=github --git-username $GH_USERNAME --git-owner $GH_OWNER --git-api-token $GH_CREDS_PSW --default-admin-password $JENKINS_CREDS_PSW --no-delete-app --no-delete-repo --tekton --tests install --tests test-create-spring

