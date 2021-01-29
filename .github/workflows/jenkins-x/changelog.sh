#!/usr/bin/env sh
source .jx/variables.sh

sed -i -e "s/^version:.*/version: $VERSION/" ./charts/$REPO_NAME/Chart.yaml
sed -i -e "s/repository:.*/repository: $DOCKER_REGISTRY\/$DOCKER_REGISTRY_ORG\/$APP_NAME/" ./charts/$REPO_NAME/values.yaml
sed -i -e "s/tag:.*/tag: $VERSION/" ./charts/$REPO_NAME/values.yaml;

jx changelog create --verbose --header-file=hack/changelog-header.md --version=$VERSION --rev=$PULL_BASE_SHA --output-markdown=changelog.md --update-release=false

git config --global user.name "jenkins-x-bot-test"
git config --global user.email "jenkins-x@googlegroups.com"

git add * || true
git commit -a -m "chore: release $VERSION" --allow-empty

#git tag -fa v$VERSION -m "Release version $VERSION"
#git push origin v$VERSION
