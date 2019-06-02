#!/usr/bin/env bash

CODECOV_VALIDATOR=$(curl -s -X POST --data-binary @.codecov.yml https://codecov.io/validate)

if ! echo "${CODECOV_VALIDATOR}" | grep -q "Valid!"; then
  echo ".codecov.yml not valid"
  echo ""
  echo "${CODECOV_VALIDATOR}"
  exit 1
else
  echo ".codecov.yml valid"
  exit 0
fi
