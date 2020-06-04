module github.com/jenkins-x/jx/v2

require (
	code.gitea.io/sdk/gitea v0.12.0
	contrib.go.opencensus.io/exporter/prometheus v0.1.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.12.9 // indirect
	github.com/Azure/draft v0.15.0
	github.com/Comcast/kuberhealthy v1.0.2
	github.com/IBM-Cloud/bluemix-go v0.0.0-20181008063305-d718d474c7c2
	github.com/Jeffail/gabs v1.1.1
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e
	github.com/Masterminds/semver v1.4.2
	github.com/Netflix/go-expect v0.0.0-20180814212900-124a37274874
	github.com/Pallinder/go-randomdata v1.1.0
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6 // indirect
	github.com/TV4/logrus-stackdriver-formatter v0.1.0
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/alecthomas/jsonschema v0.0.0-20190504002508-159cbd5dba26
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f
	github.com/andygrunwald/go-gerrit v0.0.0-20181026193842-43cfd7a94eb4
	github.com/andygrunwald/go-jira v1.5.0
	github.com/antham/chyle v1.6.0
	github.com/aws/aws-sdk-go v1.24.0
	github.com/awslabs/goformation v0.0.0-20190320125420-ac0a17860cf1
	github.com/banzaicloud/bank-vaults v0.0.0-20190508130850-5673d28c46bd
	github.com/beevik/etree v1.0.1
	github.com/blang/semver v3.5.1+incompatible
	github.com/briandowns/spinner v1.7.0 // indirect
	github.com/c2h5oh/datasize v0.0.0-20171227191756-4eba002a5eae // indirect
	github.com/cenkalti/backoff v2.1.1+incompatible
	github.com/chromedp/cdproto v0.0.0-20180720050708-57cf4773008d
	github.com/chromedp/chromedp v0.1.1
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/denormal/go-gitignore v0.0.0-20180713143441-75ce8f3e513c
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/dsnet/compress v0.0.0-20171208185109-cc9eb1d7ad76 // indirect
	github.com/elazarl/goproxy v0.0.0-20181111060418-2ce16c963a8a // indirect
	github.com/emicklei/go-restful v2.12.0+incompatible // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/fatih/color v1.7.0
	github.com/fatih/structs v1.1.0
	github.com/gfleury/go-bitbucket-v1 v0.0.0-20200320173742-022f4bab9090
	github.com/ghodss/yaml v1.0.0
	github.com/go-ole/go-ole v1.2.1 // indirect
	github.com/go-openapi/jsonreference v0.19.3
	github.com/go-openapi/spec v0.19.7
	github.com/go-openapi/swag v0.19.9 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.2
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/go-cmp v0.3.1
	github.com/google/go-containerregistry v0.0.0-20190317040536-ebbba8469d06 // indirect
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/uuid v1.1.1
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/vault v1.1.2
	github.com/heptio/sonobuoy v0.16.0
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c
	github.com/iancoleman/orderedmap v0.0.0-20181121102841-22c6ecc9fe13
	github.com/imdario/mergo v0.3.8
	github.com/jbrukh/bayesian v0.0.0-20161210175230-bf3f261f9a9c // indirect
	github.com/jenkins-x/draft-repo v0.0.0-20180417100212-2f66cc518135
	github.com/jenkins-x/golang-jenkins v0.0.0-20180919102630-65b83ad42314
	github.com/jenkins-x/lighthouse-config v0.0.3
	github.com/jetstack/cert-manager v0.5.2
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/knative/pkg v0.0.0-20190624141606-d82505e6c5b4
	github.com/knative/serving v0.7.0
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lusis/go-slackbot v0.0.0-20180109053408-401027ccfef5 // indirect
	github.com/lusis/slack-test v0.0.0-20180109053238-3c758769bfa6 // indirect
	github.com/magiconair/properties v1.8.0
	github.com/mailru/easyjson v0.7.1 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/mitchellh/mapstructure v1.2.2
	github.com/nlopes/slack v0.0.0-20180721202243-347a74b1ea30
	github.com/nwaples/rardecode v1.0.0 // indirect
	github.com/onsi/ginkgo v1.7.0
	github.com/onsi/gomega v1.4.3
	github.com/pborman/uuid v1.2.0
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/petergtz/pegomock v2.7.0+incompatible
	github.com/pierrec/lz4 v2.0.5+incompatible // indirect
	github.com/pkg/browser v0.0.0-20170505125900-c90ca0c84f15
	github.com/pkg/errors v0.8.1
	github.com/rickar/props v0.0.0-20170718221555-0b06aeb2f037
	github.com/rodaine/hclencoder v0.0.0-20180926060551-0680c4321930
	github.com/rollout/rox-go v0.0.0-20181220111955-29ddae74a8c4
	github.com/russross/blackfriday v1.5.2
	github.com/satori/go.uuid v1.2.1-0.20180103174451-36e9d2ebbde5
	github.com/sethvargo/go-password v0.1.2
	github.com/shirou/gopsutil v0.0.0-20180901134234-eb1f1ab16f2e
	github.com/shirou/w32 v0.0.0-20160930032740-bb4de0191aa4 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.3.2
	github.com/stoewer/go-strcase v1.0.1
	github.com/stretchr/testify v1.6.0
	github.com/tektoncd/pipeline v0.8.0
	github.com/trivago/tgo v1.0.1 // indirect
	github.com/ulikunitz/xz v0.5.6 // indirect
	github.com/viniciuschiele/tarx v0.0.0-20151205142357-6e3da540444d // indirect
	github.com/vrischmann/envconfig v1.2.0
	github.com/wbrefvem/go-bitbucket v0.0.0-20190128183802-fc08fd046abb
	github.com/xanzy/go-gitlab v0.22.1
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.1.0
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.opencensus.io v0.22.2 // indirect
	gocloud.dev v0.9.0
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20200323222414-85ca7c5b95cd
	golang.org/x/tools v0.0.0-20200415034506-5d8e1897c761
	gopkg.in/AlecAivazis/survey.v1 v1.8.3
	gopkg.in/alecthomas/kingpin.v2 v2.2.6 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.0.0-20190816222004-e3a6b8045b0b
	k8s.io/apiextensions-apiserver v0.0.0-20190718185103-d1ef975d28ce
	k8s.io/apimachinery v0.0.0-20190816221834-a9f1d8a9c101
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/helm v2.7.2+incompatible
	k8s.io/kube-openapi v0.0.0-20190228160746-b3a7cee44a30
	k8s.io/kubernetes v1.11.3
	k8s.io/metrics v0.0.0-20180620010437-b11cf31b380b
	k8s.io/test-infra v0.0.0-20190131093439-a22cef183a8f
	knative.dev/pkg v0.0.0-20191217184203-cf220a867b3d
	sigs.k8s.io/yaml v1.1.0

)

replace k8s.io/api => k8s.io/api v0.0.0-20190528110122-9ad12a4af326

replace k8s.io/metrics => k8s.io/metrics v0.0.0-20181128195641-3954d62a524d

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221084156-01f179d85dbc

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190528110200-4f3abb12cae2

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190528110544-fa58353d80f3

replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999

replace github.com/sirupsen/logrus => github.com/jtnord/logrus v1.4.2-0.20190423161236-606ffcaf8f5d

replace github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v21.1.0+incompatible

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v10.14.0+incompatible

go 1.13
