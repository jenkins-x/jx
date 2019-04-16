#!/usr/bin/env bash
set -e

echo "verifying Pull Request"

export ORG="jenkinsxio"
export APP_NAME="jx"
export TEAM="$(echo ${BRANCH_NAME}-$BUILD_ID  | tr '[:upper:]' '[:lower:]')"

export GH_USERNAME="jenkins-x-bot-test"
export GH_OWNER="cb-kubecd"

export GH_CREDS_PSW="$(jx step credential -s jenkins-x-bot-test-github)"
export JENKINS_CREDS_PSW="$(jx step credential -s  test-jenkins-user)"
export GKE_SA="$(jx step credential -s gke-sa)"


# for BDD tests
export GIT_PROVIDER_URL="https://github.beescloud.com"
export GHE_TOKEN="$GHE_CREDS_PSW"
export GINKGO_ARGS="-v"

export JX_DISABLE_DELETE_APP="true"
export JX_DISABLE_DELETE_REPO="true"

echo ""
git config --global credential.helper store
git config --global --add user.name JenkinsXBot
git config --global --add user.email jenkins-x@googlegroups.com

# lets create a team for this PR and run the BDD tests
gcloud auth activate-service-account --key-file $GKE_SA
gcloud container clusters get-credentials anthorse --zone europe-west1-b --project jenkinsx-dev

sed s/\$VERSION/${VERSION}/g myvalues.yaml.template > myvalues.yaml

echo the myvalues.yaml file is:
cat myvalues.yaml

echo "creating team: $TEAM"

git config --global --add user.name JenkinsXBot
git config --global --add user.email jenkins-x@googlegroups.com

cp ./build/linux/jx /usr/bin

# lets trigger the BDD tests in a new team and git provider
./build/linux/jx step bdd -b --git-owner $GH_OWNER --git-username $GH_USERNAME --provider=gke --git-provider=github.com --git-api-token $GH_CREDS_PSW --default-admin-password $JENKINS_CREDS_PSW --no-delete-app --no-delete-repo --tekon --tests install --tests test-create-spring