#!/usr/bin/env bash

mkdir vendor/protobuf && cd vendor/protobuf
curl -LO https://github.com/google/protobuf/releases/download/v3.5.1/protoc-3.5.1-linux-x86_64.zip
unzip protoc-3.5.1-linux-x86_64.zip
cd .. && mkdir protobuf-include
cp -R protobuf/include protobuf-include/
rm -rf protobuf