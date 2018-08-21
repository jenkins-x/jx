# Hacking on JX

This guide is for developers who want to improve the jenkins-x jx CLI. These instructions will help you set up a
development environment for working on the jx source code.

## Prerequisites

To compile and test jx binaries you will need:

 - [git][]
 - [Go][] 1.9 or later, with support for compiling to `linux/amd64`
 - [dep](https://github.com/golang/dep)
 

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

$ make build      # runs dep and builds `jx`  inside the build/
```

## Testing

The jx test suite is divided into three sections:
 - The standard unit test suite
 - Slow unit tests
 - Integration tests

To run the standard test suite:
```make test```

To run the standard test suite including slow running tests:
```make test-slow```

To run all tests including integration tests (NOTE These tests are not encapsulated):
```make test-slow-integration```


To get a nice HTML report on the tests:
```make test-report-html```

### Writing tests

### Unit Tests

Unit tests should be issolated (see what is an unencapsulated test), and should contain the `t.Parallel()` directive in order to keep things nice and speedy.

If you add a slow running (more than a couple of seconds) test, it needs to be wrapped like so:
```
if testing.Short() {
	t.Skip("skipping a_long_running_test")
} else {
	// Slow test goes here...
}
```
Slows tests can (and should) still include `t.Parallel()`

Best practice for unit tests is to define the testing package appending _test to the name of your package, e.g. `mypackage_test` and then import `mypackage` inside your tests.
This encourages good package design and will enable you to define the exported package API in a composable way.

### Integration Tests

To add an integration test, create a separate file for your integration tests using the naming convention `mypackage_integration_test.go` Use the same package declaaration as your unit tests: `mypackage_test`. At the very top of the file before the package declaration add this custom build directive:

```
// +build integration

```
Note that there needs to be a blank line before you declare the package name. 

This directive will ensure that integration tests are automatically separated from unit tests, and will not be run as part of the normal test suite.
You should NOT add `t.Parallel()` to an unencapsulated test as it may cause intermittent failures.

### What is an unencapsulated test?
A test is unencapsulated (not issolated) if it cannot be run (with repeatable success) without a certain surrounding state. Relying on external binaries that may not be present, writing or reading from the filesystem without care to specifically avoid collisions, or relying on other tests to run in a specific sequence for your test to pass are all examples of a test that you should carefully consider before committing. If you would like to easily check that your test is issolated before committing simply run: `make docker-test`, or if your test is marked as slow: `make docker-test-slow`. This will mount the jx project folder into a golang docker container that does not include any of your host machines environment. If your test passes here, then you can be happy that the test is encapsulated.


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

### Debugging a unit test

You can run a single unit test via

```shell
export TEST="TestSomething"
make test1
```

You can then start a Delve debug session on a unit test via:

```shell
export TEST="TestSomething"
make debugtest1
```

Then set breakpoints and debug in your IDE like in the above debugging.

### Using a helper script

If you create a bash file called `jxDebug` as the following (replacing `SomePid` with the actual `pid`):

```bash
#!/bin/sh
echo "Debugging jx"
dlv --listen=:2345 --headless=true --api-version=2 exec `which jx` -- $*
```

Then you can change your `jx someArgs` CLI to `jxDebug someArgs` then debug it!

[git]: https://git-scm.com/
[dep]: https://github.com/golang/dep 
[go]: https://golang.org/
[Homebrew]: https://brew.sh/
[Kubernetes]: https://github.com/kubernetes/kubernetes
[minikube]: https://github.com/kubernetes/minikube
[upstream]: https://help.github.com/articles/fork-a-repo/
[upx]: https://upx.github.io
