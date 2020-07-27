module github.com/jenkins-x/jxl-promote

require (
	cloud.google.com/go v0.37.4 // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/gophercloud/gophercloud v0.1.0 // indirect
	github.com/jenkins-x/jx v1.3.981-0.20200625050556-5ccee8e660bc
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.4 // indirect
	github.com/rifflock/lfshook v0.0.0-20180920164130-b9218ef580f5 // indirect
	github.com/shurcooL/githubv4 v0.0.0-20190718010115-4ba037080260 // indirect
	github.com/shurcooL/graphql v0.0.0-20181231061246-d48a9a75455f // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/api v0.3.2 // indirect
	google.golang.org/appengine v1.5.0 // indirect
	k8s.io/api v0.0.0-20190816222004-e3a6b8045b0b // indirect
	k8s.io/apimachinery v0.0.0-20190816221834-a9f1d8a9c101 // indirect
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible // indirect
	k8s.io/metrics v0.0.0-20190704050707-780c337c9cbd // indirect
)

exclude github.com/jenkins-x/jx/pkg/prow v0.0.0-20190912224545-e8f82ee218ba

replace github.com/golang/lint => golang.org/x/lint v0.0.0-20180702182130-06c8688daad7

replace github.com/heptio/sonobuoy => github.com/jenkins-x/sonobuoy v0.11.7-0.20190318120422-253758214767

replace k8s.io/api => k8s.io/api v0.0.0-20190528110122-9ad12a4af326

replace k8s.io/metrics => k8s.io/metrics v0.0.0-20181128195641-3954d62a524d

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221084156-01f179d85dbc

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190528110200-4f3abb12cae2

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190528110544-fa58353d80f3

replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999

replace github.com/sirupsen/logrus => github.com/jtnord/logrus v1.4.2-0.20190423161236-606ffcaf8f5d

replace github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v21.1.0+incompatible

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v10.15.5+incompatible

replace github.com/banzaicloud/bank-vaults => github.com/banzaicloud/bank-vaults v0.0.0-20190508130850-5673d28c46bd

go 1.12
