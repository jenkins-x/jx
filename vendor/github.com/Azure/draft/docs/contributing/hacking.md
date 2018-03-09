# Hacking on Draft

This guide is for developers who want to improve Draft. These instructions will help you set up a
development environment for working on the Draft source code.

## Prerequisites

To compile and test Draft binaries and to build Docker images, you will need:

 - a [Kubernetes][] cluster with [Draft already installed][install]. We recommend [minikube][].
 - [docker][]
 - [git][]
 - [helm][], using the same version as recommended in the [installation guide][install].
 - [Go][] 1.8 or later, with support for compiling to `linux/amd64`
 - [upx][] (optional) to compress binaries for a smaller Docker image

In most cases, install the prerequisite according to its instructions. See the next section
for a note about Go cross-compiling support.

### Configuring Go

Draft's binary executables are built on your machine and then copied into a Docker image. This
requires your Go compiler to support the `linux/amd64` target architecture. If you develop on a
machine that isn't AMD64 Linux, make sure that `go` has support for cross-compiling.

On macOS, a cross-compiling Go can be installed with [Homebrew][]:

```shell
$ brew install go --with-cc-common
```

It is also straightforward to build Go from source:

```shell
$ sudo su
$ curl -sSL https://storage.googleapis.com/golang/go1.7.5.src.tar.gz | tar -C /usr/local -xz
$ cd /usr/local/go/src
$ # compile Go for the default platform first, then add cross-compile support
$ ./make.bash --no-clean
$ GOOS=linux GOARCH=amd64 ./make.bash --no-clean
```

## Fork the Repository

Begin at Github by forking Draft, then clone your fork locally. Since Draft is a Go package, it
should be located at `$GOPATH/src/github.com/Azure/draft`.

```shell
$ mkdir -p $GOPATH/src/github.com/Azure
$ cd $GOPATH/src/github.com/Azure
$ git clone git@github.com:<username>/draft.git
$ cd draft
```

Add the conventional [upstream][] `git` remote in order to fetch changes from Draft's main master
branch and to create pull requests:

```shell
$ git remote add upstream https://github.com/Azure/draft.git
```

## Build Your Changes

With the prerequisites installed and your fork of Draft cloned, you can make changes to local Draft
source code.

Run `make` to build the `draft` and `draftd` binaries:

```shell
$ make bootstrap  # runs `dep ensure`
$ make build      # compiles `draft` and `draftd` inside bin/
```

## Test Your Changes

Draft includes a suite of tests.
- `make test-lint`: runs linter/style checks
- `make test-unit`: runs basic unit tests
- `make test`: runs all of the above

## Deploying Your Changes

To test interactively, you will likely want to deploy your changes to Draft on a Kubernetes cluster.
This requires a Docker registry where you can push your customized draftd images so Kubernetes can
pull them.

Because Draft deploys Kubernetes applications and Draft is a Kubernetes application itself, you can
use Draft to deploy Draft. How neat is that?!

To build your changes and upload it to draftd, run

```shell
$ make build docker-binary
$ draft up
--> Building Dockerfile
--> Pushing 10.0.0.237/draft/draftd:6f3b53003dcbf43821aea43208fc51455674d00e
--> Deploying to Kubernetes
--> Status: DEPLOYED
--> Notes:
     Now you can deploy an app using Draft!

        $ cd my-app
        $ draft create
        $ draft up
        --> Building Dockerfile
        --> Pushing my-app:latest
        --> Deploying to Kubernetes
        --> Deployed!

That's it! You're now running your app in a Kubernetes cluster.
```

You should see a new release of Draft available and deployed with `helm list`.

## Cleaning Up

To remove the Draft chart and local binaries:

```shell
$ make clean unserve
rm bin/*
rm rootfs/bin/*
helm delete --purge draft
```


[docker]: https://www.docker.com/
[install]: ../install.md
[git]: https://git-scm.com/
[go]: https://golang.org/
[helm]: https://github.com/kubernetes/helm
[Homebrew]: https://brew.sh/
[Kubernetes]: https://github.com/kubernetes/kubernetes
[minikube]: https://github.com/kubernetes/minikube
[upstream]: https://help.github.com/articles/fork-a-repo/
[upx]: https://upx.github.io
