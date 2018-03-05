# Hacking on JX

This guide is for developers who want to improve the jenkins-x jx CLI. These instructions will help you set up a
development environment for working on the jx source code.

## Prerequisites

To compile and test jx binaries you will need:

 - [git][]
 - [Go][] 1.9 or later, with support for compiling to `linux/amd64`
 - [glide][]
 

In most cases, install the prerequisite according to its instructions. See the next section
for a note about Go cross-compiling support.

### Configuring Go

The jx's binary eCLI is  built on your machine in your GO Path. 

On macOS, Go can be installed with [Homebrew][]:

```shell

$ brew install go 
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

Begin at Github by forking jx, then clone your fork locally. Since jx is a Go package, it
should be located at `$GOPATH/src/github.com/jenkins-x/jx`.

```shell
$ mkdir -p $GOPATH/src/github.com/jenkins-x
$ cd $GOPATH/src/github.com/jenkins-x
$ git clone git@github.com:<username>/jx.git
$ cd jx
```

Add the conventional [upstream][] `git` remote in order to fetch changes from jx's main master
branch and to create pull requests:

```shell
$ git remote add upstream https://github.com/jenkins-x/jx.git
```

## Build Your Changes

With the prerequisites installed and your fork of jx cloned, you can make changes to local jx
source code.

Run `make` to build the `jx`  binaries:

```shell

$ make build      # runs glide and builds `jx`  inside the build/
```

## Debugging

First you need to [install Delve](https://github.com/derekparker/delve/blob/master/Documentation/installation/README.md)

Then you should be able to run a debug version of a jx command:

```
dlv --listen=:2345 --headless=true --api-version=2 exec ./build/jx -- some arguments
```

Then in you IDE you should be able to then set a breakpoint and connect to `2345`.

e.g. in IntellJ you create a new `Go Remote` execution and then hit `Debug`

### Using a helper script

If you create a bash file called `jxDebug` as the following

```bash
#!/bin/sh
echo "Debugging jx"
dlv --listen=:2345 --headless=true --api-version=2 exec `which jx` -- $*
```

Then you can change your `jx someArgs` CLI to `jxDebug someArgs` then debug it!

## Cleaning Up

To remove the Draft chart and local binaries:

```shell
$ make clean 
rm build/*
```


[git]: https://git-scm.com/
[glide]: https://github.com/Masterminds/glide
[go]: https://golang.org/
[Homebrew]: https://brew.sh/
[Kubernetes]: https://github.com/kubernetes/kubernetes
[minikube]: https://github.com/kubernetes/minikube
[upstream]: https://help.github.com/articles/fork-a-repo/
[upx]: https://upx.github.io
