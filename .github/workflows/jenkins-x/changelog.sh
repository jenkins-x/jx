#!/usr/bin/env sh

echo "REPO_NAME = $PULL_BASE_SHA"

export PULL_BASE_SHA=$(git rev-parse HEAD)

if [ -d "charts/$REPO_NAME" ]; then
  sed -i -e "s/^version:.*/version: $VERSION/" ./charts/$REPO_NAME/Chart.yaml
  sed -i -e "s/tag:.*/tag: $VERSION/" ./charts/$REPO_NAME/values.yaml;

  # sed -i -e "s/repository:.*/repository: $DOCKER_REGISTRY\/$DOCKER_REGISTRY_ORG\/$APP_NAME/" ./charts/$REPO_NAME/values.yaml
else
  echo no charts
fi

jx changelog create --verbose --header-file=hack/changelog-header.md --version=$VERSION --rev=$PULL_BASE_SHA --output-markdown=changelog.md --update-release=false
