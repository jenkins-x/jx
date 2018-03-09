#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'
DRAFT_ROOT="${BASH_SOURCE[0]%/*}/.."

cd "$DRAFT_ROOT"

# Skip on pull request builds
if [[ -n "${CIRCLE_PR_NUMBER:-}" ]]; then
  exit
fi

: ${DOCKER_USERNAME:?"DOCKER_USERNAME environment variable is not set"}
: ${DOCKER_PASSWORD:?"DOCKER_PASSWORD environment variable is not set"}

VERSION=
if [[ -n "${CIRCLE_TAG:-}" ]]; then
  VERSION="${CIRCLE_TAG}"
elif [[ "${CIRCLE_BRANCH:-}" == "master" ]]; then
  VERSION="canary"
else
  echo "skipping because this is neither a push to master or a pull request."
  exit
fi

echo "Installing Docker client"
VER="17.03.0-ce"
curl -L -o /tmp/docker-$VER.tgz https://get.docker.com/builds/Linux/x86_64/docker-$VER.tgz
tar -xz -C /tmp -f /tmp/docker-$VER.tgz
mv /tmp/docker/* /usr/bin

echo "Logging into DockerHub"
docker login -u "${DOCKER_USERNAME}" -p "${DOCKER_PASSWORD}"

echo "Building Draft image"
VERSION="${VERSION}" make docker-build

echo "Pushing image to DockerHub"
VERSION="${VERSION}" make docker-push
