# jx alpha

[![Documentation](https://godoc.org/github.com/jenkins-x/jx-cli?status.svg)](https://pkg.go.dev/mod/github.com/jenkins-x/jx-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/jenkins-x/jx-cli)](https://goreportcard.com/report/github.com/jenkins-x/jx-cli)
[![Releases](https://img.shields.io/github/release-pre/jenkins-x-labs/helmboot.svg)](https://github.com/jenkins-x/jx-cli/releases)
[![LICENSE](https://img.shields.io/github/license/jenkins-x-labs/helmboot.svg)](https://github.com/jenkins-x/jx-cli/blob/master/LICENSE)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://slack.k8s.io/)

`jx-cli` is an experimental new small modular CLI for Jenkins X

## Commands

See the [jx command reference](https://github.com/jenkins-x/jx-cli/blob/master/docs/cmd/jx.md)

## Plugins

* [jx admin](https://github.com/jenkins-x/jx-admin/blob/master/docs/cmd/jx-admin.md) for administration commands (creating a new environment, booting it up with the operator)
* [jx gitops](https://github.com/jenkins-x/jx-gitops/blob/master/docs/cmd/jx-gitops.md) a set of commands used inside pipelines for modifying helm charts, kpt files, using kustomise or modifying kubernetes resources for GitOps
* [jx promote](https://github.com/jenkins-x/jx-promote/blob/master/docs/cmd/jx-promote.md#jx-promote) a command for promoting a new version of an application to an Environment
* [jx project](https://github.com/jenkins-x/jx-project/blob/master/docs/cmd/project.md) a set of commands for importing projects or creating new projects from quickstarts or wizards
* [jx secret](https://github.com/jenkins-x/jx-secret/blob/master/docs/cmd/jx-secret.md) a set of commands for working with [Kubernetes External Secrets](https://github.com/godaddy/kubernetes-external-secrets)

  
                                                            