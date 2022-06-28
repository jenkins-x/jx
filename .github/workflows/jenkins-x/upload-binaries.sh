#!/bin/bash

export BRANCH=$(git rev-parse --abbrev-ref HEAD)
export BUILDDATE=$(date)
export REV=$(git rev-parse HEAD)
export GOVERSION="1.17.9"
export ROOTPACKAGE="github.com/$REPOSITORY"

goreleaser release
