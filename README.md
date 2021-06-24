# Jenkins X CLI

[![Documentation](https://godoc.org/github.com/jenkins-x/jx?status.svg)](https://pkg.go.dev/mod/github.com/jenkins-x/jx)
[![Go Report Card](https://goreportcard.com/badge/github.com/jenkins-x/jx)](https://goreportcard.com/report/github.com/jenkins-x/jx)
[![Releases](https://img.shields.io/github/release-pre/jenkins-x/jx.svg)](https://github.com/jenkins-x/jx/releases)
[![LICENSE](https://img.shields.io/github/license/jenkins-x/jx.svg)](https://github.com/jenkins-x/jx/blob/master/LICENSE)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://slack.k8s.io/)

`jx` is the modular command line CLI for [Jenkins X 3.x](https://jenkins-x.io/v3/about/)

## Commands

See the [jx command reference](https://jenkins-x.io/v3/develop/reference/jx/)

## Issues

To track [issues in this repository](https://github.com/jenkins-x/jx/issues) and all the related [Plugins](#plugins) use these links:

* [view open issues in jenkins-x-plugins](https://github.com/issues?q=is%3Aopen+is%3Aissue+author%3Ajstrachan+archived%3Afalse+user%3Ajenkins-x-plugins)
* [view open pull requests in jenkins-x-plugins](https://github.com/pulls?q=is%3Aopen+is%3Apr+archived%3Afalse+user%3Ajenkins-x-plugins+-label%3Adependencies)

## Plugins

You can browse the documentation for all of the `jx`  plugins at:

* [Plugin CLI Reference](https://jenkins-x.io/v3/develop/reference/jx/)
* [Plugin Source](https://github.com/jenkins-x-plugins)


## Components

* [jx-git-operator](https://github.com/jenkins-x/jx-git-operator) is an operator for triggering jobs when git commits are made
* [octant-jx](https://github.com/jenkins-x/octant-jx) an open source Jenkins X UI for  [vmware-tanzu/octant](https://github.com/vmware-tanzu/octant)

## Libraries

These are the modular libraries which have been refactored out of the main [jenkins-x/jx](https://github.com/jenkins-x/jx) repository as part of the [modularisation enhancement process](https://github.com/jenkins-x/enhancements/tree/master/proposals/5#1-overview)
       
* [go-scm](https://github.com/jenkins-x/go-scm) API for working with SCM providers
* [jx-api](https://github.com/jenkins-x/jx-api) the core JX APIs
* [jx-helpers](https://github.com/jenkins-x/jx-helpers) a bunch of utilities (mostly from the `util` package) refactored + no longer dependent on [jenkins-x/jx](https://github.com/jenkins-x/jx/) 
* [jx-kube-client](https://github.com/jenkins-x/jx-kube-client) the core library for working with kube/jx/tekton clients
* [jx-logging](https://github.com/jenkins-x/jx-logging) logging APIs
* [lighthouse-client](https://github.com/jenkins-x/lighthouse-client) client library for working with [lighthouse](https://github.com/jenkins-x/lighthouse)
     
                                        
