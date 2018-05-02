## Drafting with ACR Build

To use [ACR Build](https://aka.ms/acr/build) as your preferred container image builder, you must first have an account on Microsoft Azure.

Once you have an account there, there are two steps for preparing Draft to use ACR Build as a target for building images.

1. Log in with `az login` on the command line.
2. Create an Azure Container Registry. Currently ACR Build is only supported in Container Registries deployed in EastUS and EastUS2.
3. Configure Draft to use the `acrbuild` container image builder. To do this, you need to know your Container Registry name and the Resource Group it was deployed to.

```console
$ draft config set container-builder acrbuild
$ draft config set registry myregistry.azurecr.io
$ draft config set resource-group-name the-resource-group-i-deployed-acr-to
```

Once that's done, invoke `draft up` and you'll be using the ACR Build container image builder!
