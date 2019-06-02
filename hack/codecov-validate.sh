#!/usr/bin/env bash

CODECOV_VALIDATOR=$(curl --max-time 10 -s -X POST --data-binary @.codecov.yml https://codecov.io/validate)

if ! echo "${CODECOV_VALIDATOR}" | grep -q "Valid!"; then
  if [[ ${CODECOV_VALIDATOR} -eq "" ]]; then
    echo "cannot validate .codecov.yaml right now as request timed out"
    exit 0
  else
    echo ".codecov.yml not valid"
    echo ""
    echo "${CODECOV_VALIDATOR}"
    exit 1
  fi
else
  echo ".codecov.yml valid"
  exit 0
fi
