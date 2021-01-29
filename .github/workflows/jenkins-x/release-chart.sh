#!/usr/bin/env sh
echo "HOME is $HOME"
echo current git configuration
git config --global --get user.name
git config --global --get user.emao;

git config --global user.name jenkins-x-bot-test
git config --global user.email "jenkins-x@googlegroups.com"

jx gitops git setup 

jx gitops helm release
