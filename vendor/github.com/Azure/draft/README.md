![Draft Logo](./docs/img/draft-logo.png)
[![Build Status](https://circleci.com/gh/Azure/draft.svg?style=svg)](https://circleci.com/gh/Azure/draft)

# Draft: Streamlined Kubernetes Development
Draft makes it easy to build applications that run on Kubernetes.  Draft targets the "inner loop" of a developer's workflow: as they hack on code, but before code is committed to version control.


## _NOTE: Draft is experimental and does not have a stable release yet._
As an example, starting with the 0.12 release Draft moves the containment and deployment engine to your local machine. This changes quite a bit about how Draft works compared to prior releases; for details, see [the changelog](CHANGELOG.md).

For previous users, this change means that you now have more control over where your container images are stored, and in the local Minikube environment you can skip pushing the images entirely, which speeds up the developer iteration speed _dramatically_. (Short version without jargon: you can build and run your app really fast.) However, when you want to push your images to an image registry service like Docker hub (or ACR, GCR, Quay.io, and so on) you must inform draft which registry to use and your cluster must have access to the secrets for that registry.

## Overview

[![asciicast](https://asciinema.org/a/WGVE7JNodpBEOautl105tdc97.png)](https://asciinema.org/a/WGVE7JNodpBEOautl105tdc97)

Using Draft is as simple as:

1. `draft init` to set up draft (after prerequisites are installed)
2. `draft create` to containerize your application based on Draft [packs](docs/reference/dep-003.md)
3. `draft up` to deploy your application to a Kubernetes dev sandbox, accessible using `draft connect` over a secured tunnel.
4. Use a local editor to modify the application, with changes deployed to Kubernetes in seconds.

Once the developer is happy with changes made via Draft, they commit and push to version control, after which a continuous integration (CI) system takes over.  Draft builds upon [Kubernetes Helm][helm] and the [Kubernetes Chart format](https://github.com/kubernetes/helm/blob/master/docs/charts.md), making it easy to construct CI pipelines from Draft-enabled applications.

[Join the conversation in Slack!](https://kubernetes.slack.com/messages/draft-users) If you don't already have a Kubernetes slack account, sign up [here](http://slack.k8s.io/).

## Installation

Use the [Minikube install guide][Installation Guide -- Minikube] to install and use Draft locally with Minikube. The default installation skips pushing the image, deploying directly from the local repository, which makes developer work very fast.

Use the [Cloud provider install guide][Installation Guide -- Cloud] to install and use Draft to build locally but push images to a public (or private) repository and deploy into a Kubernetes cluster offered by a cloud provider such as Azure, Google, AWS, or any other provider.

Remember, however, take note of the major changes in releases greater than 0.12 if you have already used Draft. It's different. No, really.

### Take Draft for a Spin

Climb aboard and explore the [Getting Started Guide][Getting Started] - you'll soon be sailing!

## Contributing

If you're looking to build from source or get started hacking on Draft, please see the
[hacking guide][hacking] for more information.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/). For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.

## License

This software is covered under the MIT license. You can read the license [here][license].

This software contains code from [Helm][], which is covered by the Apache v2.0 license.

You can read third-party software licenses [here][Third-Party Licenses].


[Installation Guide -- Minikube]: docs/install-minikube.md
[Installation Guide -- Cloud]: docs/install-cloud.md 
[Getting Started]: docs/getting-started.md
[hacking]: docs/contributing/hacking.md
[helm]: https://github.com/kubernetes/helm
[Installing Helm]: https://github.com/kubernetes/helm/blob/master/docs/install.md
[Kubernetes]: https://kubernetes.io/
[license]: LICENSE
[Third-Party Licenses]: NOTICE
