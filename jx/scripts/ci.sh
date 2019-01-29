#!/usr/bin/env bash
set -e

echo "verifying Pull Request"

export ORG="jenkinsxio"
export APP_NAME="jx"
export PREVIEW_VERSION="0.0.0-SNAPSHOT-${BRANCH_NAME}-${BUILD_ID}"
export TEAM="$(echo ${BRANCH_NAME}-$BUILD_ID  | tr '[:upper:]' '[:lower:]')"
export PREVIEW_IMAGE_TAG="SNAPSHOT-JX-$BRANCH_NAME-$BUILD_ID"

export GHE_CREDS_PSW="$(jx step credential -s jx-pipeline-git-github-ghe)"
export JENKINS_CREDS_PSW="$(jx step credential -s  test-jenkins-user)"
export GKE_SA="$(jx step credential -s gke-sa)"

# for BDD tests
export GIT_PROVIDER_URL="https://github.beescloud.com"
export GHE_TOKEN="$GHE_CREDS_PSW"
export GINKGO_ARGS="-v"

export JX_DISABLE_DELETE_APP="true"
export JX_DISABLE_DELETE_REPO="true"

echo "building Pull Request for preview $TEAM"

make linux
#git add . && git diff --exit-code HEAD

echo ""
git config --global credential.helper store
git config --global --add user.name JenkinsXBot
git config --global --add user.email jenkins-x@googlegroups.com

jx step git credentials


echo ""
echo "Running the integration tests"

make test-slow-integration
./build/linux/jx --help

docker build -t docker.io/$ORG/$APP_NAME:$PREVIEW_VERSION .

make preview

# lets create a team for this PR and run the BDD tests
gcloud auth activate-service-account --key-file $GKE_SA
gcloud container clusters get-credentials anthorse --zone europe-west1-b --project jenkinsx-dev


sed 's/\$VERSION/${PREVIEW_IMAGE_TAG}/g' myvalues.yaml.template > myvalues.yaml
echo the myvalues.yaml file is:
cat myvalues.yaml

echo "creating team: $TEAM"

git config --global --add user.name JenkinsXBot
git config --global --add user.email jenkins-x@googlegroups.com

cp ./build/linux/jx /usr/bin

# lets trigger the BDD tests in a new team and git provider
./build/linux/jx step bdd -b  --provider=gke --git-provider=ghe --git-provider-url=https://github.beescloud.com --git-username dev1 --git-api-token $GHE_CREDS_PSW --default-admin-password $JENKINS_CREDS_PSW --no-delete-app --no-delete-repo --tests install --tests test-create-spring
                    
                    
