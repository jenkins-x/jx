module github.com/jenkins-x/jx

require (
	code.gitea.io/sdk v0.0.0-20180702024448-79a281c4e34a
	contrib.go.opencensus.io/exporter/prometheus v0.1.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.12.5
	github.com/Azure/draft v0.15.0
	github.com/Comcast/kuberhealthy v1.0.2
	github.com/IBM-Cloud/bluemix-go v0.0.0-20181008063305-d718d474c7c2
	github.com/Jeffail/gabs v1.1.1
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e
	github.com/Masterminds/semver v1.4.2
	github.com/Netflix/go-expect v0.0.0-20180814212900-124a37274874
	github.com/Pallinder/go-randomdata v1.1.0
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6 // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/alcortesm/tgz v0.0.0-20161220082320-9c5fe88206d7 // indirect
	github.com/alecthomas/jsonschema v0.0.0-20190504002508-159cbd5dba26
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/alexflint/go-filemutex v0.0.0-20171028004239-d358565f3c3f
	github.com/andygrunwald/go-gerrit v0.0.0-20181026193842-43cfd7a94eb4
	github.com/andygrunwald/go-jira v1.5.0
	github.com/antham/chyle v1.4.0
	github.com/aws/aws-sdk-go v1.24.1
	github.com/awslabs/goformation v0.0.0-20190320125420-ac0a17860cf1
	github.com/banzaicloud/bank-vaults v0.0.0-20191212164220-b327d7f2b681
	github.com/banzaicloud/bank-vaults/pkg/sdk v0.2.1
	github.com/beevik/etree v1.0.1
	github.com/blang/semver v3.5.1+incompatible
	github.com/briandowns/spinner v1.7.0 // indirect
	github.com/c2h5oh/datasize v0.0.0-20171227191756-4eba002a5eae // indirect
	github.com/cenkalti/backoff v2.1.1+incompatible
	github.com/chromedp/cdproto v0.0.0-20180720050708-57cf4773008d
	github.com/chromedp/chromedp v0.1.1
	github.com/codeship/codeship-go v0.0.0-20180717142545-7793ca823354
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/denormal/go-gitignore v0.0.0-20180713143441-75ce8f3e513c
	github.com/docker/docker v1.13.1
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/dsnet/compress v0.0.0-20171208185109-cc9eb1d7ad76 // indirect
	github.com/elazarl/goproxy v0.0.0-20181111060418-2ce16c963a8a // indirect
	github.com/emirpasic/gods v1.9.0 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/fatih/color v1.7.0
	github.com/fatih/structs v1.1.0
	github.com/gfleury/go-bitbucket-v1 v0.0.0-20190216152406-3a732135aa4d
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-ole/go-ole v1.2.1 // indirect
	github.com/go-openapi/jsonreference v0.19.2
	github.com/go-openapi/spec v0.19.2
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.2
	github.com/google/go-cmp v0.3.1
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/uuid v1.1.1
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/vault v1.2.0-beta2.0.20190725165751-afd9e759ca82
	github.com/hashicorp/vault/api v1.0.4
	github.com/heptio/sonobuoy v0.16.0
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c
	github.com/iancoleman/orderedmap v0.0.0-20181121102841-22c6ecc9fe13
	github.com/imdario/mergo v0.3.8
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jbrukh/bayesian v0.0.0-20161210175230-bf3f261f9a9c // indirect
	github.com/jenkins-x/draft-repo v0.0.0-20180417100212-2f66cc518135
	github.com/jenkins-x/golang-jenkins v0.0.0-20180919102630-65b83ad42314
	github.com/jetstack/cert-manager v0.9.1
	github.com/kevinburke/ssh_config v0.0.0-20180317175531-9fc7bb800b55 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lusis/go-slackbot v0.0.0-20180109053408-401027ccfef5 // indirect
	github.com/lusis/slack-test v0.0.0-20180109053238-3c758769bfa6 // indirect
	github.com/magiconair/properties v1.8.0
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a
	github.com/mholt/archiver v3.1.1+incompatible
	github.com/mitchellh/mapstructure v1.1.2
	github.com/nlopes/slack v0.0.0-20180721202243-347a74b1ea30
	github.com/nwaples/rardecode v1.0.0 // indirect
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pborman/uuid v1.2.0
	github.com/pelletier/go-buffruneio v0.2.0 // indirect
	github.com/petergtz/pegomock v2.7.0+incompatible
	github.com/pkg/browser v0.0.0-20170505125900-c90ca0c84f15
	github.com/pkg/errors v0.8.1
	github.com/rickar/props v0.0.0-20170718221555-0b06aeb2f037
	github.com/rodaine/hclencoder v0.0.0-20180926060551-0680c4321930
	github.com/rollout/rox-go v0.0.0-20181220111955-29ddae74a8c4
	github.com/russross/blackfriday v1.5.2
	github.com/satori/go.uuid v1.2.1-0.20180103174451-36e9d2ebbde5
	github.com/sergi/go-diff v1.0.0 // indirect
	github.com/sethvargo/go-password v0.1.2
	github.com/shirou/gopsutil v0.0.0-20180901134234-eb1f1ab16f2e
	github.com/shirou/w32 v0.0.0-20160930032740-bb4de0191aa4 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.4.0
	github.com/src-d/gcfg v1.3.0 // indirect
	github.com/stoewer/go-strcase v1.0.1
	github.com/stretchr/testify v1.4.0
	github.com/tektoncd/pipeline v0.9.2
	github.com/trivago/tgo v1.0.1 // indirect
	github.com/ulikunitz/xz v0.5.6 // indirect
	github.com/viniciuschiele/tarx v0.0.0-20151205142357-6e3da540444d // indirect
	github.com/vrischmann/envconfig v1.2.0
	github.com/wbrefvem/go-bitbucket v0.0.0-20190128183802-fc08fd046abb
	github.com/xanzy/go-gitlab v0.22.1
	github.com/xanzy/ssh-agent v0.2.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20180127040702-4e3ac2762d5f // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.1.0
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8 // indirect
	go.opencensus.io v0.22.2 // indirect
	gocloud.dev v0.9.0
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20190927073244-c990c680b611
	gopkg.in/AlecAivazis/survey.v1 v1.8.3
	gopkg.in/src-d/go-billy.v4 v4.2.0 // indirect
	gopkg.in/src-d/go-git-fixtures.v3 v3.3.0 // indirect
	gopkg.in/src-d/go-git.v4 v4.5.0
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.0.0-20191114100352-16d7abae0d2a
	k8s.io/apiextensions-apiserver v0.0.0-20191114105449-027877536833
	k8s.io/apimachinery v0.0.0-20191028221656-72ed19daf4bb
	k8s.io/client-go v11.0.1-0.20190516230509-ae8359b20417+incompatible
	k8s.io/helm v2.7.2+incompatible
	k8s.io/kube-openapi v0.0.0-20190816220812-743ec37842bf
	k8s.io/kubernetes v1.14.0
	k8s.io/metrics v0.0.0-20180620010437-b11cf31b380b
	k8s.io/test-infra v0.0.0-20190131093439-a22cef183a8f
	knative.dev/pkg v0.0.0-20191107185656-884d50f09454
	knative.dev/serving v0.10.1
	sigs.k8s.io/yaml v1.1.0

)

replace k8s.io/api => k8s.io/api v0.0.0-20190819141258-3544db3b9e44

replace k8s.io/metrics => k8s.io/metrics v0.0.0-20190819143841-305e1cef1ab1

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190819141724-e14f31a72a77

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190819143637-0dbe462fe92d

replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999

replace github.com/sirupsen/logrus => github.com/jtnord/logrus v1.4.2-0.20190423161236-606ffcaf8f5d

replace github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v23.2.0+incompatible

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.1+incompatible

replace github.com/banzaicloud/bank-vaults => github.com/banzaicloud/bank-vaults v0.0.0-20191212164220-b327d7f2b681

replace github.com/banzaicloud/bank-vaults/pkg/sdk => github.com/banzaicloud/bank-vaults/pkg/sdk v0.0.0-20191212164220-b327d7f2b681

replace k8s.io/test-infra => github.com/abayer/test-infra v0.0.0-20200105180038-f2f6e1a9da0f

replace launchpad.net/gocheck => github.com/go-check/check v0.0.0-20140225173054-eb6ee6f84d0a
