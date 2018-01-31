#!/usr/bin/env bash

# ensure we're not on a detached head
git checkout master

# until we switch to the new kubernetes / jenkins credential implementation use git credentials store
git config credential.helper store

export VERSION="$(jx-release-version)"
echo "Creating version: ${VERSION}"

# TODO

#jx step nexus release
#jx step tag --version ${VERSION}

updatebot push-version --kind brew jx ${VERSION}
updatebot push-version --kind docker JX_VERSION ${VERSION}

