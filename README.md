# jx alpha

[![Documentation](https://godoc.org/github.com/jenkins-x/jx-cli?status.svg)](https://pkg.go.dev/mod/github.com/jenkins-x/jx-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/jenkins-x/jx-cli)](https://goreportcard.com/report/github.com/jenkins-x/jx-cli)
[![Releases](https://img.shields.io/github/release-pre/jenkins-x/jx-cli.svg)](https://github.com/jenkins-x/jx-cli/releases)
[![LICENSE](https://img.shields.io/github/license/jenkins-x/jx-cli.svg)](https://github.com/jenkins-x/jx-cli/blob/master/LICENSE)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://slack.k8s.io/)

`jx-cli` is an experimental new small modular CLI for Jenkins X

## Commands

See the [jx command reference](https://github.com/jenkins-x/jx-cli/blob/master/docs/cmd/jx.md)


## Issues

To track [issues in this repository](https://github.com/jenkins-x/jx-cli/issues) and all the related [Plugins](#plugins) use this link:

* [view open issues in jx-cli and its plugins](https://github.com/pulls?q=+++is%3Aopen+repo%3Ajenkins-x%2Fjx-cli+repo%3Ajenkins-x%2Fjx-admin+repo%3Ajenkins-x%2Fjx-application+repo%3Ajenkins-x%2Fjx-gitops+repo%3Ajenkins-x%2Fjx-pipeline+repo%3Ajenkins-x%2Fjx-project+repo%3Ajenkins-x%2Fjx-promote+repo%3Ajenkins-x%2Fjx-secret++repo%3Ajenkins-x%2Fjx-verify+)

## Plugins

* [jx admin](https://github.com/jenkins-x/jx-admin/blob/master/docs/cmd/jx-admin.md) for administration commands (creating a new environment, booting it up with the operator)
* [jx application](https://github.com/jenkins-x/jx-application/blob/master/docs/cmd/jx-application.md) for viewing applications in your environments
* [jx gitops](https://github.com/jenkins-x/jx-gitops/blob/master/docs/cmd/jx-gitops.md) a set of commands used inside pipelines for modifying helm charts, kpt files, using kustomise or modifying kubernetes resources for GitOps
* [jx pipeline](https://github.com/jenkins-x/jx-pipeline/blob/master/docs/cmd/jx-pipeline.md#jx-pipeline) a command for working with Jenkins X and Tekton Pipelines
* [jx promote](https://github.com/jenkins-x/jx-promote/blob/master/docs/cmd/jx-promote.md#jx-promote) a command for promoting a new version of an application to an Environment
* [jx project](https://github.com/jenkins-x/jx-project/blob/master/docs/cmd/project.md) a set of commands for importing projects or creating new projects from quickstarts or wizards
* [jx secret](https://github.com/jenkins-x/jx-secret/blob/master/docs/cmd/jx-secret.md) a set of commands for working with [Kubernetes External Secrets](https://github.com/godaddy/kubernetes-external-secrets)
* [jx verify](https://github.com/jenkins-x/jx-verify/blob/master/docs/cmd/jx-verify.md) a set of commands for verifying Jenkins X installations

  
                                                            