# Install Guide

Get started with Draft in three easy steps:

1. Install CLI tools for Helm, Kubectl, [Minikube][] and Draft
1. Boot Minikube and install Tiller
1. Deploy your first application

Note: This document uses a local image repository with minikube. To use Draft directly with a container registry service like https://hub.docker.com or another registry service, see the configuration steps in [Drafting in the Cloud](install-advanced.md#drafting-in-the-cloud). 

## Dependencies

In order to get started, you will need to have the following:

- the latest release of minikube
- the latest release of kubectl
- the latest release of Helm
- the latest release of Docker
- A Docker repository for storing your images

All of the dependencies (except Docker) can be installed by the following:

```shell
$ brew cask install minikube
```

Docker can be installed following the [Docker for Mac guide](https://docs.docker.com/docker-for-mac/install/).

## Install Draft

Afterwards, fetch [the latest release of Draft](https://github.com/Azure/draft/releases).

Installing Draft via Homebrew can be done using

```shell
$ brew tap azure/draft
$ brew install draft
```

Canary releases of the Draft client can be found at the following links:

- [Linux amd64](https://azuredraft.blob.core.windows.net/draft/draft-canary-linux-amd64.tar.gz)
- [macOS amd64](https://azuredraft.blob.core.windows.net/draft/draft-canary-darwin-amd64.tar.gz)
- [Linux ARM](https://azuredraft.blob.core.windows.net/draft/draft-canary-linux-arm.tar.gz)
- [Linux x86](https://azuredraft.blob.core.windows.net/draft/draft-canary-linux-386.tar.gz)

Unpack the Draft binary and add it to your PATH.

Now that Draft has been installed, we can set up Draft by running this command:

```shell
$ draft init
```

It will prepare DRAFT_HOME with a default set of packs, plugins and other directories required to get working with Draft.

## Boot Minikube

At this point, you can boot up minikube!

```shell
$ minikube start
...
Kubectl is now configured to use the cluster.
```

Now that the cluster is up and ready, minikube automatically configures kubectl, the command line tool for Kubernetes, on your machine with the appropriate authentication and endpoint information.

```shell
$ kubectl cluster-info
Kubernetes master is running at https://192.168.99.100:8443

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
```

## Install Helm

Install Helm, the Kubernetes Package Manager, in your cluster. Helm manages the lifecycle of an application in Kubernetes, and it is also how Draft deploys an application to Kubernetes.

Installing Helm and setting it up is quite simple:

```shell
$ helm init
```

Wait for Helm to come up and be in a `Ready` state. You can use `kubectl -n kube-system get deploy tiller-deploy --watch` to wait for tiller to come up.

## Configure Docker

For minikube environments, it's best to get started by telling Draft to build images directly using Minikube's Docker daemon, making the build process quick and speedy. To do this, we run

```shell
$ eval $(minikube docker-env)
$ draft config set disable-push-warning 1
```

The second command disables a warning that `draft up` outputs when no registry has been configured to push images to. Since docker builds on Minikube are immediately picked up by the Kubelet, we don't require a container registry and thus can safely disable this warning.

## Take Draft for a Spin

Once you've completed the above steps, you're ready to climb aboard and explore the [Getting Started Guide][Getting Started] - you'll soon be sailing!

## Advanced Setup

For more advanced users, [advanced setup documentation](install-advanced.md) is also provided for

- running Tiller in a Kubernetes cluster with RBAC enabled
- running Tiller in a namespace other than kube-system

[Getting Started]: getting-started.md
[minikube]: https://github.com/kubernetes/minikube
