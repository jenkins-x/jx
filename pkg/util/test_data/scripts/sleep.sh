#!/bin/sh

# This script is intended to be used to simulate a long running execution. Currently used in command_test.go

set -e

SLEEP="$1"
[ $# -eq 0 ] && { echo "Usage: $0 10"; exit 1; }
sleep $1
echo $1
