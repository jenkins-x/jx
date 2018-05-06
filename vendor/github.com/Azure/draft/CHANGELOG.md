# Changelog

## v0.14.0

### Features

* introduced `draft connect --detach` (see DEP 007)
* implemented Draft tasks (see DEP 008)
* introduced ACR Build support, with support for writing alternative container builders (see DEP 009)
* introduced Windows support
* introduced Powershell support for plugin install hooks
* introduced `draft pack list`
* introduced support for custom image tags on `draft up`
* introduced `draft init --config` to specify a default set of plugins and pack repositories to bootstrap Draft

### Bugs

* fixed an issue where `draft create --pack` required the full path to the pack, rather than just `--pack=python`
* fixed an issue where `set` fields in draft.toml were not being respected
* fixed an issue where `draft init` would always install the canary release of `draft pack-repo`

### Housekeeping

* switched the default number of replicas spawned from the default Draft packs from 2 to 1
* switched the default value of `wait` in draft.toml to `true`

## v0.13.0

### Features

* introduced `draft logs --tail`
* introduced `draft connect --dry-run`
* `draft up` now writes logs directly to the file as it happens
  * this allows users to run `draft logs` in another terminal as soon as they call `draft up`
* added a more helpful error to run `draft pack-repo update` when a Draft Pack cannot be loaded
* all files within the root directory of a Draft Pack is now loaded into the application's directory on `draft create`
* an image pull secret is injected into the namespace on `draft up` when pushing an image to a container registry

### Bugs

* fixed up an issue where output from a `docker push` and `helm install` wasn't available in `draft logs`
* fixed an issue where `draft pack-repo list` wouldn't work on Windows
* fixed an issue where `draft config unset` wasn't truncating config.toml

## v0.12.0

### Features

* Removed draftd
* New packs added:
   * Rust (thanks to @FGRibreau)
* Introduced `draft history`
* Introduced `draft config`
* `draft connect`
   * Introduced the `--override-port` flag to specify a local:remote port mapping for tunnelling
* `draft logs`
   * Command has been simplified to `draft logs <build-id>`, or `draft logs` to get the latest build's logs
* `draft up`
   * Introduced the `--auto-connect` flag to automatically connect to your app once it's deployed

### Bugs

* fixed an ipv6 lookup error when connecting to draftd (before removing it)
* fixed a rebase issue with the Swift pack that caused it to not work on `draft create`

### Housekeeping

* switched from SHA1 to SHA256 for app context shasums (thanks @thedrow for the heads up)

## v0.11.0

### Features

* Improved and more granular `draft logs` functionality
   * Each instance of `draft up` results in a Build ID and you can now get logs by build ID
* `draft connect`
   * Now connects to every containerPort in application pod by default
   * Added `--container/-c` flag to support connecting to a specific container in the pod

### Bugs

* Corrected readiness/liveness probe port in draftd chart

### Housekeeping

* CI upkeep, docs, and go format improvements
* Updates to Dockerfiles in Java/Gradle packs
   * Notably, switched from Alpine to Debian based Docker image


## v0.10.0

### Features

* Introduced `draft init --upgrade`
* TLS support added via `draft init`:
   * --draftd-tls
   * --draftd-tls-cert string
   * --draftd-tls-key string
   * --draftd-tls-verify
* Reverted back to using a docker-in-docker container for draft builds for cross-cloud support
* Added ability to save application state information in Kubernetes as ConfigMaps
* New packs added:
   * Clojure (thanks to @kstrempel)
* New example-spring-boot application added (thanks to @jstrachan)
* Introduced `draft connect --environment`
* When `draft create` fails on the first language, it now attempts all other detected languages for packs

### Bugs

* `draft up -e` and `draft connect -e` will now return an error if the environment is not found in draft.toml

### Housekeeping

* switched from [glide](https://github.com/Masterminds/glide) to [dep](https://github.com/golang/dep)
* removed unused/flaky end-to-end tests; to be refactored in #486


## v0.9.0

### Client

* ported [linguist's .gitattributes support](https://github.com/Azure/draft/blob/v0.9.0/docs/troubleshooting.md#my-repository-is-detected-as-the-wrong-language)
* `draft create` now bootstraps with a `charts/` directory, as opposed to `chart/`
* application releases are purged from the Kubernetes cluster on `draft delete`
* added `--dry-run` flag to `draft init`
* the name of the directory is now used as the application name for `draft create`
* added a [Swift](https://swift.org/) pack
* ASP.NET pack was bumped to 2.0

### Server

* the docker-in-docker container was removed in favour of mounting the host's docker socket

### Documentation

* design documentation has been re-organized into [Draft Enhancement Proposals](https://github.com/Azure/draft/blob/v0.9.0/docs/reference/index.md), aka DEPs

## v0.8.0

### Client

* implemented `draft delete` to remove applications from Kubernetes
* implemented `draft logs` to view build logs after `draft up`
* added --tail flag to `draft connect` (as well as `draft logs`)
* added -i/--image flag to `draft init` to override the draftd image * added "upgrade" workflow to `draft init`
* installed the pack-repo plugin by default on `draft init`
* switched default listening port to 3000 for apps deployed with the default Ruby pack
* added global flag `--draft-namespace` for talking to draftd in another namespace

### Server

* bumped max RPC message size to 40MB

### Documentation

* clarified how to use ingress with the basedomain field
* added an asciicast on draft's workflow
* added documentation on creating and maintaining a pack repository

## v0.7.0

### Client

* introduced `draft connect`
* added a new UI for `draft up`
* introduced language aliases to linguist

### Server

* removed requirement for ingress setup, making it optional
* bumped GRPC max message size from 4MB to 20MB
* removed registry org and image name from generated charts, simplifying templates

### Community

* introduced a draft Homebrew formula. Use `brew tap azure/draft && brew install draft` to try it out

## v0.6.0

### Client

* introduced a new plugin manager! See `draft plugins`
* introduced smarter language detection for apps through `draft create`
* `draft init --yes` has been renamed to `draft init --auto-accept`
* STDIN is now attached when running Draft plugins
* the project file watcher feature has been disabled by default
* fixed a regression where values in draft.toml were not being pushed to the server
* rewrote `draft create` to make charts generated by draft helm-compatible

### Server

* the websocket framework for Draft has been completely re-written to communicate via the gRPC protocol!
* bumped to helm v2.5.1 compatibility
* added `ondraft=true` as an injected value into charts deployed via Draft

### Documentation

* alter project governance to a more team-based model
* add documentation on the new `draft plugins` feature

## v0.5.1

### Client

* fix up --yes being ignored on `draft init`

### Server

* use overlayfs as the selected storage driver

## v0.5.0

### Client

* added .draftignore file support
* added .NET pack support
* added gradle pack support
* renamed java pack as maven
* refactored the PHP and maven packs to utilize multi-stage Dockerfile builds
* re-wrote `draft init` for a smoother installation experience

### Server

* image pull secrets are now updated on changes
* fixed some bugs with running draft on Windows, specifically around `draft home` and `draft create`
* the draft server now runs a docker-in-docker sidecar container instead of mounting the host socket
* bumped to helm v2.5 compatibility

### Documentation

* install documentation has been overhauled with the new `draft init` behaviour
* added project scope
* added project archutecture

### Test Infrastructure

* added codecov integration to new pull requests

## v0.4.0

### Client

* Added -o/--dest flag to `draft create`
* Fixed unused --app flag on `draft create`
* go-bindata is now used to package the default packs, making it easier to contribute new packs

### Server

* Bumped to Helm v2.4 compatibility
* Refactored and cleaned up package `api`

### Documentation

* Added a Draft logo
* Added a nice video tutorial for Draft on Azure Container Services
* Documented `basedomain` and basic ingress setups
* Added descriptions to some of the fields in `draft init`
* Added documentation on getting started with Minikube

### Test Infrastructure

* CI will now publish binaries on tagged releases of Draft

## v0.3.0

### Client

* Added default draft packs for 6 different languages
* Ignore temporary files from file watcher
* Switched to `draft.toml`
* Draft auto-generates the application name on `draft create`

### Server

* Connect to tiller via kubernetes service

### Documentation

* Added example applications for 6 different languages
* Switched getting-started documentation over to use python example app
* Added basedomain logic to ingress hosts
* Added Governance Model

### Test Infrastructure

* Switched to Jenkins
* Upload build artifacts to Azure Blob Storage
* Improved code coverage

## v0.2.0

### Client

* New command: `draft home`
* New command: `draft init`
* Introduced pack detection into `draft create`
* New option flags on `draft up`: `-f`, `--set`, and `--values`
* Introduced a default Ingress resource with the default nginx pack
* Introduced `draft.yaml`

### Server

* Initialized connection to Helm on startup rather than at build time
* Bumped Helm to commit 1aee50f

### Documentation

* Introduced the --watch flag in the Getting Started Guide
* Documented the release process 

### Test Infrastructure

* Introduced Drone CI!
  * Canary images are uploaded to docker registry
  * Canary clients are uploaded to S3 for linux-arm, linux-i386, linux-amd64, darwin-amd64, and windows-amd64
  * Release images and clients are uploaded, too!
* Unit tests for the client and server were improved over this release
* Introduced `hack/docker-make.sh` to run the test suite inside a container

## v0.1.0

Initial release! :tada:
