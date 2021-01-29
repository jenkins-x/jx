#!/usr/bin/env sh
echo "HOME is $HOME"
echo current git configuration

git config --global --get user.name
git config --global --get user.email

echo "setting git user"

git config --global user.name jenkins-x-bot-test
git config --global user.email "jenkins-x@googlegroups.com"

jx gitops helm release
