#!/usr/bin/env bash
echo "HOME is $HOME"
echo setting the git user.name
git config --global --add user.name jenkins-x-bot-test
git config user.name jenkins-x-bot-test
jx gitops helm release
