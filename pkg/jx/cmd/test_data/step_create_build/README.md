# Jenkins X Pipeline Test Suite

This folder contains a collection of example `jenkins-x.yml` files which show how the `jenkins-x.yml` file is converted to a knative `Build` resource

## Features

### Defaulting images from the previous step

Its common to use lots of steps in a pipeline using the same image. So make things more DRY you can just omit image names and it will default to the preivous image.

* [jenkins-x.xml](default_image_from_previous_step/jenkins-x.yml#L12) generates [build.yaml](default_image_from_previous_step/expected-build-release.yml)


### Defaulting images from the build packs

Rather than maintaining the image and version for each build pileline in each git repository you can share the default for your Jenkins X installation version.

i.e. a maven build pack pipeline will default the OOTB container image from Jenkins X.

* [jenkins-x.xml](default_image_from_pod_templates/jenkins-x.yml#L12) generates [build.yaml](default_image_from_pod_templates/expected-build-release.yml)

