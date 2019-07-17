#!/usr/bin/env bash
set -e

echo "verifying Pull Request"
export ORG="jenkinsxio"
export APP_NAME="jx"
export TEAM="$(echo ${BRANCH_NAME}-$BUILD_ID  | tr '[:upper:]' '[:lower:]')"

export GHE_CREDS_PSW="$(jx step credential -s jx-pipeline-git-github-ghe | sed -e 's/PASS//' -e 's/coverage: [0-9\.]*% of statements in [\w\.\/]*//' | tr -d [:space:])"
export JENKINS_CREDS_PSW="$(jx step credential -s  test-jenkins-user | sed -e 's/PASS//' -e 's/coverage: [0-9\.]*% of statements in [\w\.\/]*//' | tr -d [:space:])"
export GKE_SA="$(jx step credential -k bdd-credentials.json -s bdd-secret -f sa.json)"
export REPORTS_DIR="${BASE_WORKSPACE}/build/reports"

# for BDD tests
export GIT_PROVIDER_URL="https://github.beescloud.com"
export GHE_TOKEN="$GHE_CREDS_PSW"
export GINKGO_ARGS="-v"

export JX_DISABLE_DELETE_APP="true"
export JX_DISABLE_DELETE_REPO="true"

# Disable manual promotion test for bdd context
export JX_BDD_SKIP_MANUAL_PROMOTION="true"

echo ""
git config --global credential.helper store
git config --global --add user.name JenkinsXBot
git config --global --add user.email jenkins-x@googlegroups.com

JX_HOME="/tmp/jxhome"

export GIT_COMMITTER_NAME="dev1"

mkdir -p $JX_HOME

# Disable coverage for jx version as we don't validate the output at all
COVER_JX_BINARY=false jx version
jx step git credentials

# lets create a team for this PR and run the BDD tests
gcloud auth activate-service-account --key-file $GKE_SA

sed -e s/\$VERSION/${VERSION_PREFIX}${VERSION}/g -e s/\$CODECOV_TOKEN/${CODECOV_TOKEN}/g myvalues.yaml.template > myvalues.yaml

#echo the myvalues.yaml file is:
#cat myvalues.yaml

echo "creating team: $TEAM"

git config --global --add user.name JenkinsXBot
git config --global --add user.email jenkins-x@googlegroups.com

git clone https://github.com/jenkins-x/jenkins-x-versions.git
git fetch
git checkout upgrade-chart-versions-871d94c0-a53b-11e9-b7af-5263cf64ba3b

# lets trigger the BDD tests in a new team and git provider
jx step bdd -b \
    --provider=gke \
    --versions-repo https://github.com/jenkins-x/jenkins-x-versions.git \
    --config jenkins-x-versions/jx/bdd/static/cluster.yaml \
    --gopath /tmp \
    --git-provider=ghe \
    --git-provider-url=https://github.beescloud.com \
    --git-username dev1 \
    --git-api-token $GHE_CREDS_PSW \
    --default-admin-password $JENKINS_CREDS_PSW \
    --no-delete-app \
    --no-delete-repo \
    --tests install \
    --tests test-create-spring

# Reset the namespace back to jx after test for any followup steps
COVER_JX_BINARY=false jx ns jx
