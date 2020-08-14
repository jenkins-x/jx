#!/bin/bash

echo "promoting the new version ${VERSION} to downstream repositories"

git clone https://github.com/jenkins-x/jx3-gitops-template.git
cd jx3-gitops-template

sed -i -e "s/jx-cli:.*/jx-cli:${VERSION}/" .jx/git-operator/job.yaml

git commit -a -m "fix: upgrade jx-cli"
git push

./promote.sh

cd ..

echo "running the promote binary"
./promote/main