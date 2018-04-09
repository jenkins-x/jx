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

## Testing

There's a handy script to output nice syntax highlighted output of test results via:

```shell 
./test.sh
```

Or you can use `make`

```shell 
make test
```

### Debug logging

Lots of the test have debug output to try figure out when things fail. You can enable verbose debug logging for tests via

```shell 
export JX_TEST_DEBUG=true
```

## Debugging

First you need to [install Delve](https://github.com/derekparker/delve/blob/master/Documentation/installation/README.md)

Then you should be able to run a debug version of a jx command:

```
dlv --listen=:2345 --headless=true --api-version=2 exec ./build/jx -- some arguments
```

Then in you IDE you should be able to then set a breakpoint and connect to `2345`.

e.g. in IntellJ you create a new `Go Remote` execution and then hit `Debug`

### Debugging jx with stdin

If you want to debug using `jx` with `stdin` to test out terminal interaction, you can start `jx` as usual from the command line then:

* find the `pid` of the jx command via something like `ps -elaf | grep jx`
* start Delve attaching to the pid:

```shell

dlv --listen=:2345 --headless=true --api-version=2 attach SomePID
```

### Using a helper script

If you create a bash file called `jxDebug` as the following (replacing `SomePid` with the actual `pid`):

```bash
#!/bin/sh
echo "Debugging jx"
dlv --listen=:2345 --headless=true --api-version=2 exec `which jx` -- $*
```

Then you can change your `jx someArgs` CLI to `jxDebug someArgs` then debug it!

[git]: https://git-scm.com/
[glide]: https://github.com/Masterminds/glide
[go]: https://golang.org/
[Homebrew]: https://brew.sh/
[Kubernetes]: https://github.com/kubernetes/kubernetes
[minikube]: https://github.com/kubernetes/minikube
[upstream]: https://help.github.com/articles/fork-a-repo/
[upx]: https://upx.github.io
