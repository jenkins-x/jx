#!/usr/bin/env bash

read -r -d '' JSON << EOJSON
{
  "labels": {
    "created-by-prow": "true",
    "prowJobName": "cdf89f04-98ec-11e9-a846-4ad95a1bb3ab"
  },
  "prowJobSpec": {
    "type": "presubmit",
    "agent": "tekton",
    "cluster": "default",
    "namespace": "jx",
    "job": "serverless-jenkins",
    "refs": {
      "org": "hf-bee",
      "repo": "demo",
      "repo_link": "https://github.com/hf-bee/demo",
      "base_ref": "master",
      "base_sha": "1b1f1ad000a4ef4087a8dd8f61f0fb94e9302fc5",
      "base_link": "https://github.com/hf-bee/demo/commit/1b1f1ad000a4ef4087a8dd8f61f0fb94e9302fc5",
      "pulls": [
        {
          "number": 1,
          "author": "hferentschik",
          "sha": "a2e02647568eab192ab8bfd21091820356aa916a",
          "link": "https://github.com/hf-bee/demo/pull/1",
          "commit_link": "https://github.com/hf-bee/demo/pull/1/commits/a2e02647568eab192ab8bfd21091820356aa916a",
          "author_link": "https://github.com/hferentschik"
        }
      ]
    },
    "report": true,
    "context": "serverless-jenkins",
    "rerun_command": "/test this"
  }
}
EOJSON

curl -v http://localhost:8080 --data "$JSON" \
--header "Host: pipelinerunner" \
--header "Content-Type: application/json" \
--header "Accept-Encoding: gzip"





