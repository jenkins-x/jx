# JX
-	Website: https://jenkins-x.io/
-   Slack channels (part of kubernetes workspace):
    -   [#jenkins-x-user](https://app.slack.com/client/T09NY5SBT/C9MBGQJRH) for users of Jenkins X
    -   [#jenkins-x-dev](https://app.slack.com/client/T09NY5SBT/C9LTHT2BB) for developers of Jenkins X
-   Discourse forum: [Discourse](https://jenkinsx.discourse.group/)

JX is a command line tool for installing and using [Jenkins X](https://jenkins-x.io/).

[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/3237/badge)](https://bestpractices.coreinfrastructure.org/projects/3237)
[![GoDoc](https://godoc.org/github.com/jenkins-x/jx?status.svg)](https://godoc.org/github.com/jenkins-x/jx)
[![Docs](https://readthedocs.org/projects/docs/badge/?version=latest)](https://jenkins-x.io/docs/)
[![Docker Pulls](https://img.shields.io/docker/pulls/jenkinsxio/jx.svg)](https://hub.docker.com/r/jenkinsxio/jx/tags)
[![Downloads](https://img.shields.io/github/downloads/jenkins-x/jx/total.svg)](https://github.com/jenkins-x/jx/releases)
[![GoReport](https://goreportcard.com/badge/github.com/jenkins-x/jx)](https://goreportcard.com/report/github.com/jenkins-x/jx)
[![Apache](https://img.shields.io/badge/license-Apache-blue.svg)](https://github.com/jenkins-x/jx/blob/master/LICENSE)
[![Reviewed by Hound](https://img.shields.io/badge/reviewed_by-Hound-8E64B0.svg)](https://houndci.com)
![Build Status](https://img.shields.io/endpoint?url=https%3A%2F%2Fstatusbadge-jx.jenkins-x.live%2Fjx)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://slack.k8s.io/)

## Installing

Check out [how to install jx](https://jenkins-x.io/docs/getting-started/setup/install/).

## Getting Started

Please check out the [Getting Started Guide](https://jenkins-x.io/docs/getting-started/) on how to:

* [create new Kubernetes clusters with Jenkins X](https://jenkins-x.io/docs/getting-started/setup/create-cluster/)
* [install Jenkins X on existing Kubernetes clusters](https://jenkins-x.io/docs/install-setup/installing/boot/)

Then [what to do next when you have Jenkins X installed](https://jenkins-x.io/docs/create-project/).

## Welcome to the Jenkins X Community

We value respect and inclusiveness and follow the [CDF Code of Conduct](https://github.com/cdfoundation/toc/blob/master/CODE_OF_CONDUCT.md) in all interactions.

We’d love to talk with you about Jenkins X and are happy to help if you have any questions.

Find out more about our bi-weekly office hours, where we discuss all things Jenkins X, and other events [here](https://jenkins-x.io/community/).

## Getting Help

To find out the available commands type:
```
jx
```
Or to get help on a specific command, say, `create` then type:
```
jx help create
```
You can also browse the [jx command reference documentation](https://jenkins-x.io/commands/jx/).

## Reference

* [Command Line Reference](https://jenkins-x.io/commands/jx/#jx)
  
  
## Opening Consoles
To open a console for `foo`:
```sh
jx open foo
```
If you do not know the name:
```sh
jx open
```
## Tail logs

To tail the logs of anything running on Kubernetes (jenkins or your own applications) type.
```sh
jx logs
```
Which prompts you for the deployment to log then tails the logs of the newest pod for an app.

You can filter the list of deployments via:
```sh
jx logs -f cheese
```
Then if there's only one deployment with a name that contains `cheese` then it'll tail the logs of the latest pod or will prompt you to choose the exact deployment to use.

## Remote shells

You can open a remote shell inside any pods container via the `rsh` command:
```sh
jx rsh
```
Or to open a shell inside a pod named foo:
```sh
jx rsh foo
```
Pass `-c` to specify the container name. e.g. to open a shell in a maven build pod:
```sh
jx rsh -c maven maven
```
## Importing or Creating apps

To import an application from the current directory:
```sh
jx import
```
Or to create a new Spring Boot application from scratch:
```sh
jx create spring
```
e.g. to create a new WebMVC and Spring Boot Actuator microservice try this:
```sh
jx create spring -d web -d actuator
```
Or to create a new project from scratch:
```sh
jx create project
```
## Starting builds

To start a pipeline using a specific name try:
```sh
jx start pipeline myorg/myrepo
```
Or to pick the pipeline to start:
```sh
jx start pipeline
```
If you know part of the name of the pipeline to run you can filter the list via:
```sh
jx start pipeline -f thingy
```
You can start and tail the build log via:
```sh
jx start pipeline -t
```
## Viewing Apps and Environments

To view environments for a team:
```sh
jx get env
```
To view the application versions across environments:
```sh
jx get version
```
## Manual promotions

Typically we setup Environments to use _automatic_ promotion so that the CI / CD pipelines will automatically promote versions through the available Environments using the CI / CD Pipeline.

However if you wish to manually promote a version to an environment you can use the following command:
```sh
jx promote myapp -e prod
```
Or if you wish to use a custom namespace:   
```sh
jx promote myapp -n my-dummy-namespace
```
## Switching Environments

The `jx` CLI tool uses the same Kubernetes cluster and namespace context as `kubectl`. 

You can switch Environments via:
```sh
jx env
```
Or change it via:
```sh
jx env staging
jx env prod
```
    
To display the current environment without trying to change it:
```sh
jx env -b
```
To view all the environments type:
```sh
jx get env
```
You can create or edit environments too:
```sh
jx create env # Create an environment
jx edit env staging # Edit staging environment
```
You can switch namespaces in the same way via:
```sh
jx ns
```
or
```sh
jx ns awesome-staging
```
## Switching Clusters

If you have multiple Kubernetes clusters then you can switch between them via:
```sh
jx ctx
```
**Note** that changing the namespace ,environment or cluster changes the current context for **ALL** shells!

### Sub shells

So if you want to work temporarily with, say, the production cluster we highly recommend you use a sub shell for that.
```sh
jx shell my-prod-context
```
Or to pick the context to use for the sub shell:
```sh
jx shell
```
Then your bash prompt will be updated to reflect that you are in a different context and/or namespace. Any changes to the namespace, environment or context will be local to the current shell only!    

### Setting your prompt

You can use the `jx prompt` to configure your CLI prompt to display the current team and environment you are working within:          
```sh
# Enable the prompt for bash
PS1="[\u@\h \W \$(jx prompt)]\$ "

# Enable the prompt for zsh
PROMPT='$(jx prompt)'$PROMPT
```
**Note** that the prompt is updated automatically for you via the `jx shell` command too.

### Bash completion

On a Mac to enable bash completion try:
```sh
jx completion bash > ~/.jx/bash
source ~/.jx/bash
```
Or try:
```sh
source <(jx completion bash)
```
For more help try:
```sh
jx help completion bash
```
## Addons

We are adding a number of addon capabilities to Jenkins X. To add or remove addons use the `jx create addon` or `jx delete addon` commands.

For example to add the [Gitea Git server](https://gitea.io/en-US/) to your Jenkins X installation try:
```sh
jx create addon gitea
```
This will:

* install the Gitea Helm chart.
* add Gitea as a Git server (via the `jx create git server gitea` command).
* create a new user in Gitea (via the `jx create git user -n gitea` command).
* create a new Git API token in Gitea (via the `jx create git token -n gitea -p password username` command).

## Troubleshooting

We have tried to collate common issues here with work arounds. If your issue isn't listed here please [let us know](https://github.com/jenkins-x/jx/issues/new).

### Other issues

Please [let us know](https://github.com/jenkins-x/jx/issues/new) and see if we can help? Good luck!

## Contributing

We welcome your contributions.

If you're looking to build from source or get started hacking on jx, please see the [CONTRIBUTING.MD](CONTRIBUTING.MD) or our [Contributing Guide](https://jenkins-x.io/community/code/) on the Jenkins X website.


[![](https://codescene.io/projects/4772/status.svg) Get more details at **codescene.io**.](https://codescene.io/projects/4772/jobs/latest-successful/results)


[experiments]: docs/contributing/experiments.md
