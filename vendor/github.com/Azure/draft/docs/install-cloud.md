# Cloud installations and Advanced Configuration

In certain situations, users are given a Kubernetes cluster with security constraints such as

- needing to push and pull container images to and from a remote or otherwise secured container registry
- only being able to deploy applications to a certain namespace
- running applications in an RBAC-enabled cluster

This document explains some of these situations as well as how it can be handled using draft.

## Drafting in the Cloud

When using Draft with cloud-provided Kubernetes solutions like [Azure Container Service (AKS)](https://azure.microsoft.com/services/container-service/), [Google Kubernetes Engine (GKE)](https://cloud.google.com/kubernetes-engine/), or [Amazon EKS](https://aws.amazon.com/eks/), we need a way to distribute the built image across the cluster. A container registry allows all nodes in the Kubernetes cluster to pull the images we build using Draft, and we have a way for our local Docker daemon to distribute the built image.

For local use with Minikube, only the `eval $(minikube docker-env)` command is required to inform Draft to use the local registry.

For cloud registry services, like Azure Container Registry (ACR), Docker Hub, or other container registry services, two things are needed. 
1. You need to tell Draft where the registry resides using the `draft config set registry` command, passing the registry's server URL (without the protocol scheme). 
2. Unless there is a trust relationship between the cluster provider and the registry -- {as you can configure between the Azure Kubernetes Service (AKS) and the Azure Container Registry (ACR)](https://docs.microsoft.com/azure/container-registry/container-registry-auth-aks#grant-aks-access-to-acr) -- you'll need to either [add a container registry secret to your chart](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry) to pull the private image, or you can configure Draft to inject a secret for you for deployment using the login feature of your registry provider.

## Drafting with ACR and AKS

Let's use Azure Container Registry (ACR) as an example registry, and use AKS as our Kubernetes service for this example. 

REMEMBER: Kubernetes services like Azure Kubernetes Service (AKS) are automatically authorized to pull from container registries in the same resource group. If this is not the case, then please reread the [Drafting in the Cloud](#drafting-in-the-cloud) section. 

For this example, we want to push images to our registry sitting at `myregistry.azurecr.io`, and pull those images down to the Kubernetes cluster from that same registry. To do that, we run

```shell
$ draft config set registry myregistry.azurecr.io
```

This command tells Draft to push images to this container registry as well as to tell Kubernetes to pull images from this container registry. If you were using Docker Hub, for example, you would specify `docker.io/myregistry`. 

We'll also need to log into the cluster to push images from our local docker daemon to the container registry:

```
$ az acr login -n myregistry -g myresourcegroup
```

If you were using Docker Hub, for example, this command would be the `docker login` command.

NOTE: Once configured, Draft will inject a registry auth secret into the destination namespace deploying the chart to the cluster so the image can be pulled from the registry. 

## Running Tiller with RBAC enabled

To install Tiller in a cluster with RBAC enabled, a few additional steps are required to grant tiller access to deploy in namespaces.

[Helm's documentation](https://docs.helm.sh/using_helm/#role-based-access-control) on setting up Tiller in RBAC-enabled clusters is the best document on the subject, but for new users who just want to get started, let's create a new service account for tiller with the right roles to deploy to the `default` namespace.

```shell
$ kubectl create serviceaccount tiller --namespace kube-system
serviceaccount "tiller" created
```

Let's define a role that allows tiller to manage all resources in the `default` namespace. In `role-tiller.yaml`:

```yaml
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: tiller-manager
  namespace: default
rules:
- apiGroups: ["", "extensions", "apps"]
  resources: ["*"]
  verbs: ["*"]
```

```shell
$ kubectl create -f role-tiller.yaml
role "tiller-manager" created
```

Now let's bind the service account to that role. In `rolebinding-tiller.yaml`:

```yaml
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: tiller-binding
  namespace: default
subjects:
- kind: ServiceAccount
  name: tiller
  namespace: kube-system
roleRef:
  kind: Role
  name: tiller-manager
  apiGroup: rbac.authorization.k8s.io
```

```shell
$ kubectl create -f rolebinding-tiller.yaml
rolebinding "tiller-binding" created
```

We'll also need to grant tiller access to read configmaps in kube-system so it can store release information. In `role-tiller-kube-system.yaml`:

```yaml
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  namespace: kube-system
  name: tiller-manager
rules:
- apiGroups: ["", "extensions", "apps"]
  resources: ["configmaps"]
  verbs: ["*"]
```

```shell
$ kubectl create -f role-tiller-kube-system.yaml
role "tiller-manager" created
```

And the respective role binding. In `rolebinding-tiller-kube-system.yaml`:

```yaml
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: tiller-binding
  namespace: kube-system
subjects:
- kind: ServiceAccount
  name: tiller
  namespace: kube-system
roleRef:
  kind: Role
  name: tiller-manager
  apiGroup: rbac.authorization.k8s.io
```

```shell
$ kubectl create -f rolebinding-tiller-kube-system.yaml
rolebinding "tiller-binding" created
```

Then, install tiller.

```shell
$ helm init --service-account=tiller
```

## Running Tiller in a namespace other than kube-system

To install Tiller in a namespace other than kube-system, a user can run `helm init` with the `--tiller-namespace` feature flag. This feature flag will deploy Tiller in that namespace.

There are a few prerequisites to make this work, however:

For minikube users, the setup documentation starts to look like this when deploying Tiller to the `draft` namespace:

```shell
$ minikube start
$ kubectl create namespace draft
$ helm init --tiller-namespace draft
```

From then on, you can interact with Tiller either with the `--tiller-namespace` flag or by setting `$TILLER_NAMESPACE` in your client's environment.

```
$ draft up --tiller-namespace draft
$ export TILLER_NAMESPACE=draft
$ draft up
```
