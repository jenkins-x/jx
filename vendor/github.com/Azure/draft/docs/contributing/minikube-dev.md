When developing locally with minkube (after `minikube addons enable registry`):

In one terminal window:
```console
$ helm init
$ tiller_deploy=$(kubectl get po -n kube-system -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}' | grep "tiller")
$ kubectl port-forward $tiller_deploy 44134:44134 -n kube-system
```

In another terminal window:
```console
$ registry_pod=$(kubectl get po -n kube-system -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}' | grep "registry")
$ kubectl port-forward $registry_pod 5000:5000 -n kube-system
```

In another terminal window:
```console
$ eval $(minikube docker-env)
$ registry_ip=$(kubectl get svc registry -n kube-system -o go-template --template '{{.spec.clusterIP'}})
$ registry_port=$(kubectl get svc registry -n kube-system -o go-template --template '{{range .spec.ports }}{{.port}}{{end}}')
$ draftd start --listen-addr="127.0.0.1:44135" --registry-auth="e30K" --tiller-uri=":44134" --basedomain=k8s.local --registry-url $registry_ip:$registry_port --local
```

In another terminal you can run draft commands after the following step:
```console
export DRAFT_HOST=127.0.0.1:44135
```

