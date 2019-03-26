#!/bin/bash

files=$(find . -name "*.go" | grep -v vendor/ | grep -v Dockerfile | xargs gofmt -l -s)
if [[ $files ]]; then
    echo "Gofmt errors in files:"
    echo "$files"
    diff=$(find . -name "*.go" | grep -v vendor/ | grep -v Dockerfile | xargs gofmt -d -s)
    echo "$diff"
    exit 1
fi
