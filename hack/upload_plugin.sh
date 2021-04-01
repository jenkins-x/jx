#!/bin/bash

echo "uploading the plugin for version ${VERSION}"

echo "creating the plugin.gz"
cd dist

echo "apiVersion: jenkins.io/v1
kind: Plugin
metadata:
  labels:
    jenkins.io/pluginCommand: jx
  name: remote
spec:
  description: CloudBees plugin for remote environments
  name: remote
  subCommand: remote
  version: ${VERSION}" > plugin.yaml
tar -czvf ../plugin.gz plugin.* *.zip *.gz *.txt *.md
cd ..

echo "created plugin.gz:"
pwd
ls -al *.gz

echo "uploading the plugin distro to github"
github-release upload \
    --user jenkins-x \
    --repo jx \
    --tag v${VERSION} \
    --name "plugin.gz" \
    --file plugin.gz

    

