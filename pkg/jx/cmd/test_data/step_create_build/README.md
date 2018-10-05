# Jenkins X Pipeline Test Suite

This folder contains a collection of example `jenkins-x.yml` files which show how the `jenkins-x.yml` file is converted to a Knative `build` resources for different branch patterns.

## Features

### Defaulting environment variables and volumes from pod templates

Its very common to require lots of common environment variables or volumes (e.g. for the Docker socket or binding `Secrets`)

By default if you specify a build pack Jenkins X will default the build pack environment variables and volumes from the [Pod Template](https://jenkins-x.io/architecture/pod-templates/) for your build pack:

* [jenkins-x.xml](inherit_pod_template_env_volumes/jenkins-x.yml) generates [build.yaml](inherit_pod_template_env_volumes/expected-build-release.yml)

You can opt out of this by [excluding env vars or volumes in the pod atemplate](add_common_envvars/jenkins-x.yml#L4-L5)
                                             

### Defaulting images from the previous step

Its common to use lots of steps in a pipeline using the same image. So make things more DRY you can just omit image names and it will default to the preivous image.

* [jenkins-x.xml](default_image_from_previous_step/jenkins-x.yml#L12) generates [build.yaml](default_image_from_previous_step/expected-build-release.yml)


### Defaulting images from the build packs

Rather than maintaining the image and version for each build pipeline in each Git repository you can share the default for your Jenkins X installation version.

i.e. a maven build pack pipeline will default the OOTB container image from Jenkins X.

* [jenkins-x.xml](default_image_from_pod_templates/jenkins-x.yml#L12) generates [build.yaml](default_image_from_pod_templates/expected-build-release.yml)

### Define common environment variables

Its very handy to be able to define common environment variables and have them added to each stem:

* [jenkins-x.xml](add_common_envvars/jenkins-x.yml#L5-L7) generates [build.yaml](add_common_envvars/expected-build-release.yml)

