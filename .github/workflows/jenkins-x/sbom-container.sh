#!/bin/sh

# Install syft in this script
apk add --no-cache curl unzip
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | \
sh -s -- -b /usr/local/bin v0.55.0
chmod +x /usr/local/bin/syft

# Generate SBOM
syft ghcr.io/$DOCKER_REGISTRY_ORG/$REPO_NAME:$VERSION --scope all-layers \
-o spdx-json > sbom.json

#Push SBOM with oras
echo $GITHUB_TOKEN | oras push -u $GIT_USERNAME --password-stdin  \
ghcr.io/$DOCKER_REGISTRY_ORG/$REPO_NAME:$VERSION-sbom sbom.json
echo $GITHUB_TOKEN | oras push -u $GIT_USERNAME --password-stdin  \
ghcr.io/$DOCKER_REGISTRY_ORG/$REPO_NAME:latest-sbom sbom.json
