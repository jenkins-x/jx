#!/usr/bin/env bash
ORG_REPOS=("jenkins-x/jx-ts-client")
for org_repo in "${ORG_REPOS[@]}"; do
  OUTDIR="$(jx step git fork-and-clone -b --print-out-dir https://github.com/$org_repo)"
  echo "Forked repo to $OUTDIR"
  VERSION="$(cat VERSION)"
  pushd $OUTDIR
  echo "Running make all in $ORG_REPOS"
  make all
  echo "make all complete in $ORG_REPOS"
  git add -N .
  git diff --exit-code
  if [ $? -ne 0 ]; then
    jx create pullrequest -b --push=true --body="upgrade $org_repo client to jx $VERSION" --title="upgrade to jx $VERSION" --label="updatebot"
  fi
  popd
done