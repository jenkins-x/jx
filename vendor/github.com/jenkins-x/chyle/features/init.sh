#!/bin/bash

gitRepositoryPath=testing-repository

if [ ! -z "${TRAVIS+x}" ];
then
    git config --global user.name "whatever";
    git config --global user.email "whatever@example.com";
fi

# Configure name

# Init
rm -rf $gitRepositoryPath > /dev/null;
git init --quiet $gitRepositoryPath;

cd $gitRepositoryPath || exit 1;

git config --local user.name "whatever";
git config --local user.email "whatever@example.com";
