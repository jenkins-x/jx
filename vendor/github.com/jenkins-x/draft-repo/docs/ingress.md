# What is Ingress

Ingress is a way to route traffic from the internet to services within your Kubernetes cluster, without creating a load-balancer for each service. For more information, review the [Kubernetes Ingress Documentation][Kubernetes Ingress Documentation].

This documentation is only relevant to users who have explicitly enabled ingress routes with Draft. If you did not supply the `--ingress-enabled` flag with `draft init`, then please skip this!

## Installing an Ingress Controller

### Cloud providers

While there are many ingress controllers available within the Kubernetes Community, for simplicity, this guide will use the nginx-ingress from the stable helm charts, but you are welcome to use any ingress controller.

These documents assume you are connected to a Kubernetes cluster running in a cloud-provider.

```shell
$ helm install stable/nginx-ingress --namespace=kube-system --name=nginx-ingress
```

After you've installed the nginx-ingress controller, wait for a Load Balancer to be created with:

```shell
$ kubectl --namespace kube-system get services -w nginx-ingress-nginx-ingress-controller
```

### Minikube

On minikube, you can simply enable the ingress controller add-on

```shell
$ minikube addons enable ingress
```

The ingress IP address is minikube's IP:

```shell
$ minikube ip
```

## Point a wildcard domain

When ingress is enabled, Draft uses a wildcard domain to make accessing draft-created applications easier. To do so, it specifies a custom host in the ingress from which tells the backing load balancer to route requests based on the Host header.

Using a domain that you manage, create a DNS wildcard `A Record` pointing to the ingress IP address.

**NOTE:** you are welcome to use `*.draft.example.com` or any other wildcard domain.

Remember the domain you use, it will be needed in the next step of installation as the `basedomain` passed to `draft init `.
Note that to configure the ingress, you must initialize draft using `draft init --ingress-enabled`!

| Name          | Type | Data                      |
|---------------|------|---------------------------|
| *.example.com | A    | `<ip address from above>` |

### I don't manage a domain

If you don't manage a domain you can't directly use the domain in your request. To fulfill the load balancer request to have the Host header provided, you can explicitly provide it:

```shell
$ curl --header Host:<application domain> <ip address from above>
```

#### dnsmasq

For wildcard support, you can use a DNS server like dnsmasq. Installing a local DNS server like dnsmasq and configuring your system to use that server can make `/etc/hosts` configuration changes a thing of the past.

There are plenty of ways to install dnsmasq for MacOS users, but the easiest by far is to use Homebrew.

```shell
$ brew install dnsmasq
```

Once it's installed, you will want to point all outgoing requests to `k8s.local` to your minikube instance.

```shell
$ echo 'address=/.k8s.local/`minikube ip`' > $(brew --prefix)/etc/dnsmasq.conf
$ sudo brew services start dnsmasq
```

This will start dnsmasq and make it resolve requests from `k8s.local` to your minikube instance's IP address (usually some form of 192.168.99.10x), but now we need to point the operating system's DNS resolver at dnsmasq to resolve addresses.

```shell
$ sudo mkdir /etc/resolver
$ echo nameserver 127.0.0.1 | sudo tee /etc/resolver/k8s.local
```

Afterwards, you will need to clear the DNS resolver cache so any new requests will go through dnsmasq instead of hitting the cached results from your operating system.

```shell
$ sudo killall -HUP mDNSResponder
```

To verify that your operating system is now pointing all `k8s.local` requests at dnsmasq:

```shell
$ scutil --dns | grep k8s.local -B 1 -A 3
resolver #8
  domain   : k8s.local
  nameserver[0] : 127.0.0.1
  flags    : Request A records, Request AAAA records
  reach    : Reachable, Local Address, Directly Reachable Address
```

If you're on Linux, refer to [Arch Linux's fantastic wiki on dnsmasq](https://wiki.archlinux.org/index.php/dnsmasq).

If you're on Windows, refer to [Acrylic's documentation][acrylic], which is another local DNS proxy
specifically for Windows. Just make sure that Acrylic is pointing at minikube through `k8s.local`.
You can use the above steps as a general guideline on how to set up Acrylic.

#### /etc/hosts

You could also edit your `/etc/hosts` file to point to the ingressed out application domain to your cluster.

The following snippet would allow you to access an application:

```shell
$ sudo echo <ip address from above> <application domain> >> /etc/hosts
```

The draw back is that `/etc/hosts` does not support wildcards, so you would need to add an entry for each application deployed by Draft.

## Next steps

Once you have an ingress controller installed and configured on your cluster, you're ready to install Draft.

Continue with the [Installation Guide][Installation Guide]!

[acrylic]: http://mayakron.altervista.org/wikibase/show.php?id=AcrylicHome
[Installation Guide]: install.md#install-draft
[Kubernetes Ingress Documentation]: https://kubernetes.io/docs/concepts/services-networking/ingress/
