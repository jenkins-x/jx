#!/bin/sh

# This script is intended to be used to match a pass attempt in a loop. Currently used in command_test.go

set -e

_file="$1"
[ $# -eq 0 ] && { echo "Usage: $0 filename"; exit 1; }
[ ! -f "$_file" ] && { echo "Error: $0 file not found."; exit 2; }
 
if [ -s "$_file" ] 
then
	read TRY < $_file
else
	TRY=1
fi

PASS_ATTEMPT=$2

if [ "$TRY" = "$PASS_ATTEMPT" ]; then
	echo "PASS"
	TRY=$(($TRY + 1))
	echo $TRY > $1
	exit 0
else
	echo "FAILURE!"
	TRY=$(($TRY + 1))
	echo $TRY > $1
	exit 1
fi
