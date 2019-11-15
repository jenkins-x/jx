#!/bin/sh
echo "Debugging jx"
dlv --listen=:2345 --headless=true --api-version=2 exec ./build/jx -- $*
