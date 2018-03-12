#!/usr/bin/env bash

cd $(dirname $0)

docker run \
    -v $PWD/..:/go/src/github.com/Azure/draft \
    --workdir /go/src/github.com/Azure/draft \
    deis/go-dev:v0.22.0 \
    make $@
