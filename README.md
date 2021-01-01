# Jenkins X CLI for Version 3.x

[![Documentation](https://godoc.org/github.com/jenkins-x/jx-cli?status.svg)](https://pkg.go.dev/mod/github.com/jenkins-x/jx-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/jenkins-x/jx-cli)](https://goreportcard.com/report/github.com/jenkins-x/jx-cli)
[![Releases](https://img.shields.io/github/release-pre/jenkins-x/jx-cli.svg)](https://github.com/jenkins-x/jx-cli/releases)
[![LICENSE](https://img.shields.io/github/license/jenkins-x/jx-cli.svg)](https://github.com/jenkins-x/jx-cli/blob/master/LICENSE)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://slack.k8s.io/)

`jx-cli` is the modular command line CLI for [Jenkins X 3.x](https://jenkins-x.io/v3/about/)


## Commands

See the [jx command reference](https://github.com/jenkins-x/jx-cli/blob/master/docs/cmd/jx.md)


## Issues

To track [issues in this repository](https://github.com/jenkins-x/jx-cli/issues) and all the related [Plugins](#plugins) use this link:

* [view open issues in jx-cli and its plugins](https://github.com/issues?q=is%3Aopen+is%3Aissue+repo%3Ajenkins-x%2Fjx-cli+repo%3Ajenkins-x%2Fjx-admin+repo%3Ajenkins-x%2Fjx-application+repo%3Ajenkins-x%2Fjx-apps+repo%3Ajenkins-x%2Fjx-helpers+repo%3Ajenkins-x%2Fjx-git-operator+repo%3Ajenkins-x%2Fjx-gitops+repo%3Ajenkins-x%2Fjx-pipeline+repo%3Ajenkins-x%2Fjx-project+repo%3Ajenkins-x%2Fjx-promote+repo%3Ajenkins-x%2Fjx-secret+repo%3Ajenkins-x%2Fjx-verify+repo%3Ajenkins-x%2F%2Fjx-secret+repo%3Ajenkins-x%2Foctant-jx+)
* [view open pull requests in jx-cli and its plugins](https://github.com/pulls?q=is%3Aopen+is%3Apr+-label%3Adependencies+repo%3Ajenkins-x%2Fjx-cli+repo%3Ajenkins-x%2Fjx-admin+repo%3Ajenkins-x%2Fjx-application+repo%3Ajenkins-x%2Fjx-apps+repo%3Ajenkins-x%2Fjx-helpers+repo%3Ajenkins-x%2Fjx-git-operator+repo%3Ajenkins-x%2Fjx-gitops+repo%3Ajenkins-x%2Fjx-pipeline+repo%3Ajenkins-x%2Fjx-project+repo%3Ajenkins-x%2Fjx-promote+repo%3Ajenkins-x%2Fjx-secret+repo%3Ajenkins-x%2Fjx-verify+repo%3Ajenkins-x%2F%2Fjx-secret+repo%3Ajenkins-x%2Foctant-jx+)
* [view open pull requests in jenkins-x-plugins](https://github.com/pulls?q=is%3Aopen+is%3Apr++archived%3Afalse+user%3Ajenkins-x-plugins+)

## Plugins

* [jx admin](https://github.com/jenkins-x/jx-admin/blob/master/docs/cmd/jx-admin.md) for administration commands (creating a new environment, booting it up with the operator)
* [jx application](https://github.com/jenkins-x/jx-application/blob/master/docs/cmd/jx-application.md) for viewing applications in your environments
* [jx gitops](https://github.com/jenkins-x/jx-gitops/blob/master/docs/cmd/jx-gitops.md) a set of commands used inside pipelines for modifying helm charts, kpt files, using kustomise or modifying kubernetes resources for GitOps
* [jx health](https://github.com/jenkins-x-plugins/jx-health/blob/master/docs/cmd/jx-health.md) for visualising and reporting on cluster health
* [jx jenkins](https://github.com/jenkins-x/jx-jenkins/blob/master/docs/cmd/jx-jenkins.md) a set of commands for working with [Jenkins](https://jenkins.io/) servers in kubernetes
* [jx pipeline](https://github.com/jenkins-x/jx-pipeline/blob/master/docs/cmd/jx-pipeline.md#jx-pipeline) a command for working with Jenkins X and Tekton Pipelines
* [jx preview](https://github.com/jenkins-x/jx-preview/blob/master/docs/cmd/jx-preview.md#preview) a command for creating Preview Environments
* [jx promote](https://github.com/jenkins-x/jx-promote/blob/master/docs/cmd/jx-promote.md#jx-promote) a command for promoting a new version of an application to an Environment
* [jx project](https://github.com/jenkins-x/jx-project/blob/master/docs/cmd/project.md) a set of commands for importing projects or creating new projects from quickstarts or wizards
* [jx secret](https://github.com/jenkins-x/jx-secret/blob/master/docs/cmd/jx-secret.md) a set of commands for working with [Kubernetes External Secrets](https://github.com/godaddy/kubernetes-external-secrets)
* [jx test](https://github.com/jenkins-x/jx-test/blob/master/docs/cmd/jx-test.md) a set of commands for managing tests on kubernetes/clouds
* [jx verify](https://github.com/jenkins-x/jx-verify/blob/master/docs/cmd/jx-verify.md) a set of commands for verifying Jenkins X installations

[check out all of the other plugins available](https://github.com/jenkins-x-plugins)

## Components

* [jx-git-operator](https://github.com/jenkins-x/jx-git-operator) is an operator for triggering jobs when git commits are made
* [octant-jx](https://github.com/jenkins-x/octant-jx) an open source Jenkins X UI for  [vmware-tanzu/octant](https://github.com/vmware-tanzu/octan)

## Libraries

These are the modular libraries which have been refactored out of the main [jenkins-x/jx](https://github.com/jenkins-x/jx) repository as part of the [modularisation enhancement process](https://github.com/jenkins-x/enhancements/tree/master/proposals/5#1-overview)
       
* [go-scm](https://github.com/jenkins-x/go-scm) API for working with SCM providers
* [jx-api](https://github.com/jenkins-x/jx-api) the core JX APIs
* [jx-helpers](https://github.com/jenkins-x/jx-helpers) a bunch of utilities (mostly from the `util` package) refactored + no longer dependent on [jenkins-x/jx](https://github.com/jenkins-x/jx/) 
* [jx-kube-client](https://github.com/jenkins-x/jx-kube-client) the core library for working with kube/jx/tekton clients
* [jx-logging](https://github.com/jenkins-x/jx-logging) logging APIs
                                             


### Deprecated libraries

* [jx-apps](https://github.com/jenkins-x/jx-apps) a library for loading/saving the `jx-apps.yml` file
* [lighthouse-config](https://github.com/jenkins-x/lighthouse-config) for configuring lighthouses
