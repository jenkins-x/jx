#!/usr/bin/env bash

make skaffold-run
reflex -r "\.go$" -R "vendor.*" make skaffold-run
