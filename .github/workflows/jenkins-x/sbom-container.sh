#!/bin/sh

syft ghcr.io/$DOCKER_REGISTRY_ORG/$REPO_NAME:$VERSION --scope all-layers \
-o spdx-json > sbom.json
echo $GITHUB_TOKEN | oras push -u $GIT_USERNAME --password-stdin  \
ghcr.io/$DOCKER_REGISTRY_ORG/$REPO_NAME:$VERSION-sbom sbom.json
echo $GITHUB_TOKEN | oras push -u $GIT_USERNAME --password-stdin  \
ghcr.io/$DOCKER_REGISTRY_ORG/$REPO_NAME:latest-sbom sbom.json
