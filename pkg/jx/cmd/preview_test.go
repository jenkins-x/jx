package cmd_test

import (
	"reflect"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/gits/mocks"
	gits_matchers "github.com/jenkins-x/jx/pkg/gits/mocks/matchers"
	"github.com/jenkins-x/jx/pkg/helm/mocks"
	"github.com/jenkins-x/jx/pkg/jx/cmd"
	cmd_mocks "github.com/jenkins-x/jx/pkg/jx/cmd/mocks"
	cmd_matchers "github.com/jenkins-x/jx/pkg/jx/cmd/mocks/matchers"
	"github.com/jenkins-x/jx/pkg/kube"
	k8s_v1 "k8s.io/api/core/v1"
	k8s_cs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8s_cs_fake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"

	jio_v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"

	cs_fake "github.com/jenkins-x/jx/pkg/client/clientset/versioned/fake"

	//"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	//"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1/fake"

	//"k8s.io/apimachinery/pkg/api/resource"
	. "github.com/petergtz/pegomock"
	"github.com/stretchr/testify/assert"

	//apiexts_mock "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"os"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube_mocks "k8s.io/client-go/kubernetes/fake"
)

func AnyPtrToString() *string {
	RegisterMatcher(NewAnyMatcher(reflect.TypeOf((*(*string))(nil)).Elem()))
	var nullValue *string
	return nullValue
}

func AnyPtrToInt() *int {
	RegisterMatcher(NewAnyMatcher(reflect.TypeOf((*(*int))(nil)).Elem()))
	var nullValue *int
	return nullValue
}

func TestRun(t *testing.T) {

	//TODO: t.Parallel()

	RegisterMockTestingT(t)

	// mock factory
	factory := cmd_mocks.NewMockFactory()

	//TODO: how are PullRequest & PullRequestName different? They get set to the same value when defaulting values; bug having both?
	// -> PullRequest is the user provided value (poss. with PR- prefix). PullRequestName is the number. (normalise in options parsing?)
	previewOpts := &cmd.PreviewOptions{
		PromoteOptions: cmd.PromoteOptions{
			CommonOptions: cmd.CommonOptions{
				Factory: factory,
				Out:     os.Stdout,
				In:      os.Stdin,
				//TODO: remove batch mode, just used to avoid stdin prompts.
				BatchMode: true,
			},
			Application: "my-app",
		},
		Namespace:    "jx",
		DevNamespace: "jx",
		Name:         "my-app-name",
		SourceURL:    "https://github.com/an-org/a-repo.git",
		PullRequest:  "1",
	}

	namespace := &k8s_v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "jx-testing",
		},
	}

	secret := &k8s_v1.Secret{
		StringData: map[string]string{
			"a": "b",
			"c": "d",
		},
	}

	ingressConfig := &k8s_v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: kube.ConfigMapIngressConfig,
		},
		Data: map[string]string{"key1": "value1", "domain": "test-domain", "config.yml": ""},
	}
	/*exposeControllerConfig := &k8s_v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta {
			Name: kube.ConfigMapExposecontroller,
		},
		Data: map[string]string{"key1":"value1"},
	}*/

	//_, err = o.KubeClient.CoreV1().ConfigMaps(o.NS).Create(cm)

	// mock Kubernetes interface
	kubernetesInterface := kube_mocks.NewSimpleClientset(namespace, secret, ingressConfig)
	kubernetesInterface.CoreV1().ConfigMaps("jx").Create(ingressConfig)

	service := &k8s_v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "my-app",
			Annotations: map[string]string{kube.ExposeURLAnnotation: "http://the-service-url/with/a/path"},
		},
	}
	kubernetesInterface.CoreV1().Services("jx").Create(service)

	//TODO: create mock ingresses.
	//kubernetesInterface.ExtensionsV1beta1().Ingresses("jx").Create(ingress x 4)

	//var _ clientset.Interface = kube_mocks.NewSimpleClientset(node)
	var apiClient k8s_cs.Interface = &k8s_cs_fake.Clientset{}
	// Override CreateClient to return mock Kubernetes interface

	//Setup K8S mocks:
	When(factory.CreateClient()).ThenReturn(kubernetesInterface, "jx-testing", nil)
	When(factory.CreateApiExtensionsClient()).ThenReturn(apiClient, nil)

	//Setup Git mocks:
	mockGitProvider := gits_test.NewMockGitProvider()
	When(factory.CreateGitProvider(AnyString(), //gitURL
		AnyString(), //message
		cmd_matchers.AnyAuthAuthConfigService(),
		AnyString(), //gitKind
		AnyBool(),   //batchMode,
		cmd_matchers.AnyGitsGitter(),
		cmd_matchers.AnyTerminalFileReader(),
		cmd_matchers.AnyTerminalFileWriter(),
		cmd_matchers.AnyIoWriter(),
	)).ThenReturn(mockGitProvider, nil)
	//TODO: fill in PR details.
	mockGitPR := &gits.GitPullRequest{Owner: "owner1"}
	When(mockGitProvider.GetPullRequest(AnyString(), //owner
		gits_matchers.AnyPtrToGitsGitRepositoryInfo(), //repo
		AnyInt(), // number
	)).ThenReturn(mockGitPR, nil)

	env := &jio_v1.Environment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-app-name",
		},
	}

	cs := cs_fake.NewSimpleClientset(env)
	When(factory.CreateJXClient()).ThenReturn(cs, "jx", nil)

	//TODO: check environment was created in environmentsResource (cs).

	//TODO: assert CRD registrations?

	mockHelmer := helm_test.NewMockHelmer()
	When(factory.GetHelm(AnyBool(), AnyString(), AnyBool(), AnyBool())).ThenReturn(mockHelmer)
	When(mockHelmer.UpgradeChart(AnyString(),
		AnyString(),
		AnyString(),
		AnyPtrToString(),
		AnyBool(),
		AnyPtrToInt(),
		AnyBool(),
		AnyBool(),
		AnyStringSlice(),
		AnyStringSlice())).ThenReturn(nil) //err=nil

	//TODO: work out how to fake out github environment (queried to get PR details). Use file url? return pre-built API responses (JSON?)?

	//TODO: do we really _need_ to get the author email? i.e. it _must_ be set?

	//TODO: set git username credential info GIT_USERNAME, GIT_API_TOKEN, GIT_BEARER_TOKEN
	os.Setenv("GITHUB_USERNAME", "markawm")
	os.Setenv("GITHUB_BEARER_TOKEN", "abc123def")
	os.Setenv(cmd.JENKINS_X_DOCKER_REGISTRY_SERVICE_HOST, "MyOrganisation")
	os.Setenv(cmd.JENKINS_X_DOCKER_REGISTRY_SERVICE_PORT, "5000")
	os.Setenv("BUILD_NUMBER", "1")
	os.Setenv(cmd.ORG, "MyOrganisation")
	os.Setenv(cmd.APP_NAME, "MyApp")
	os.Setenv(cmd.PREVIEW_VERSION, "v0.1.2")

	err := previewOpts.Run()

	assert.NoError(t, err, "Should not error")

	// Setup options
	/*options := &cmd.StatusOptions{
		CommonOptions: cmd.CommonOptions{
			Factory: factory,
			Out:     os.Stdout,
			Err:     os.Stderr,
		},
	}

	err := options.Run()

	assert.NoError(t, err, "Should not error")*/
}

/*
func TestNewCmdPreview(t *testing.T) {
	type args struct {
		f   cmd.Factory
		in  terminal.FileReader
		out terminal.FileWriter
	}
	tests := []struct {
		name       string
		args       args
		want       *cobra.Command
		wantErrOut string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errOut := &bytes.Buffer{}
			if got := NewCmdPreview(tt.args.f, tt.args.in, tt.args.out, errOut); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCmdPreview() = %v, want %v", got, tt.want)
			}
			if gotErrOut := errOut.String(); gotErrOut != tt.wantErrOut {
				t.Errorf("NewCmdPreview() = %v, want %v", gotErrOut, tt.wantErrOut)
			}
		})
	}
}*/

// Testcases:
//   Run:
//    Happy path - preview created.
//    PreviewJobPollDuration
//	    Valid - exceed
//      Invalid
//    Discoverappname (don't provide in the options). ((belongs in CommonOptions test))
//    Poll time & duration
//      Valid & invalid.
//    Git username missing & prompted from stdin.
//    Git parameters (URL, username, ...) are:
//      Provided in PreviewOptions
//        As source-ref
//        As source-url
//      Determined from current git directory.
//      Authentication from environment variables.
//      Authentication from secrets.
//      ...
//    Domain sources (see configmap.go & domain, err := kube.GetCurrentDomain(kubeClient, ns))
//      From ConfigMapIngressConfig
//      From ConfigMapExposeController
//      From CM.Data.domain
//      From CM.Data.config.yml
//
// Refactorings:
//   Git detection stuff (currently in preview.go).

/*
func TestPreviewOptions_addPreviewOptions(t *testing.T) {
	type fields struct {
		PromoteOptions                PromoteOptions
		Name                          string
		Label                         string
		Namespace                     string
		DevNamespace                  string
		Cluster                       string
		PullRequestURL                string
		PullRequest                   string
		SourceURL                     string
		SourceRef                     string
		Dir                           string
		PostPreviewJobTimeout         string
		PostPreviewJobPollTime        string
		PullRequestName               string
		GitConfDir                    string
		GitProvider                   gits.GitProvider
		GitInfo                       *gits.GitRepositoryInfo
		PostPreviewJobTimeoutDuration time.Duration
		PostPreviewJobPollDuration    time.Duration
		HelmValuesConfig              config.HelmValuesConfig
	}
	type args struct {
		cmd *cobra.Command
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &PreviewOptions{
				PromoteOptions:                tt.fields.PromoteOptions,
				Name:                          tt.fields.Name,
				Label:                         tt.fields.Label,
				Namespace:                     tt.fields.Namespace,
				DevNamespace:                  tt.fields.DevNamespace,
				Cluster:                       tt.fields.Cluster,
				PullRequestURL:                tt.fields.PullRequestURL,
				PullRequest:                   tt.fields.PullRequest,
				SourceURL:                     tt.fields.SourceURL,
				SourceRef:                     tt.fields.SourceRef,
				Dir:                           tt.fields.Dir,
				PostPreviewJobTimeout:         tt.fields.PostPreviewJobTimeout,
				PostPreviewJobPollTime:        tt.fields.PostPreviewJobPollTime,
				PullRequestName:               tt.fields.PullRequestName,
				GitConfDir:                    tt.fields.GitConfDir,
				GitProvider:                   tt.fields.GitProvider,
				GitInfo:                       tt.fields.GitInfo,
				PostPreviewJobTimeoutDuration: tt.fields.PostPreviewJobTimeoutDuration,
				PostPreviewJobPollDuration:    tt.fields.PostPreviewJobPollDuration,
				HelmValuesConfig:              tt.fields.HelmValuesConfig,
			}
			options.addPreviewOptions(tt.args.cmd)
		})
	}
}

func TestPreviewOptions_Run(t *testing.T) {
	type fields struct {
		PromoteOptions                PromoteOptions
		Name                          string
		Label                         string
		Namespace                     string
		DevNamespace                  string
		Cluster                       string
		PullRequestURL                string
		PullRequest                   string
		SourceURL                     string
		SourceRef                     string
		Dir                           string
		PostPreviewJobTimeout         string
		PostPreviewJobPollTime        string
		PullRequestName               string
		GitConfDir                    string
		GitProvider                   gits.GitProvider
		GitInfo                       *gits.GitRepositoryInfo
		PostPreviewJobTimeoutDuration time.Duration
		PostPreviewJobPollDuration    time.Duration
		HelmValuesConfig              config.HelmValuesConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &PreviewOptions{
				PromoteOptions:                tt.fields.PromoteOptions,
				Name:                          tt.fields.Name,
				Label:                         tt.fields.Label,
				Namespace:                     tt.fields.Namespace,
				DevNamespace:                  tt.fields.DevNamespace,
				Cluster:                       tt.fields.Cluster,
				PullRequestURL:                tt.fields.PullRequestURL,
				PullRequest:                   tt.fields.PullRequest,
				SourceURL:                     tt.fields.SourceURL,
				SourceRef:                     tt.fields.SourceRef,
				Dir:                           tt.fields.Dir,
				PostPreviewJobTimeout:         tt.fields.PostPreviewJobTimeout,
				PostPreviewJobPollTime:        tt.fields.PostPreviewJobPollTime,
				PullRequestName:               tt.fields.PullRequestName,
				GitConfDir:                    tt.fields.GitConfDir,
				GitProvider:                   tt.fields.GitProvider,
				GitInfo:                       tt.fields.GitInfo,
				PostPreviewJobTimeoutDuration: tt.fields.PostPreviewJobTimeoutDuration,
				PostPreviewJobPollDuration:    tt.fields.PostPreviewJobPollDuration,
				HelmValuesConfig:              tt.fields.HelmValuesConfig,
			}
			if err := o.Run(); (err != nil) != tt.wantErr {
				t.Errorf("PreviewOptions.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPreviewOptions_RunPostPreviewSteps(t *testing.T) {
	type fields struct {
		PromoteOptions                PromoteOptions
		Name                          string
		Label                         string
		Namespace                     string
		DevNamespace                  string
		Cluster                       string
		PullRequestURL                string
		PullRequest                   string
		SourceURL                     string
		SourceRef                     string
		Dir                           string
		PostPreviewJobTimeout         string
		PostPreviewJobPollTime        string
		PullRequestName               string
		GitConfDir                    string
		GitProvider                   gits.GitProvider
		GitInfo                       *gits.GitRepositoryInfo
		PostPreviewJobTimeoutDuration time.Duration
		PostPreviewJobPollDuration    time.Duration
		HelmValuesConfig              config.HelmValuesConfig
	}
	type args struct {
		kubeClient kubernetes.Interface
		ns         string
		url        string
		pipeline   string
		build      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &PreviewOptions{
				PromoteOptions:                tt.fields.PromoteOptions,
				Name:                          tt.fields.Name,
				Label:                         tt.fields.Label,
				Namespace:                     tt.fields.Namespace,
				DevNamespace:                  tt.fields.DevNamespace,
				Cluster:                       tt.fields.Cluster,
				PullRequestURL:                tt.fields.PullRequestURL,
				PullRequest:                   tt.fields.PullRequest,
				SourceURL:                     tt.fields.SourceURL,
				SourceRef:                     tt.fields.SourceRef,
				Dir:                           tt.fields.Dir,
				PostPreviewJobTimeout:         tt.fields.PostPreviewJobTimeout,
				PostPreviewJobPollTime:        tt.fields.PostPreviewJobPollTime,
				PullRequestName:               tt.fields.PullRequestName,
				GitConfDir:                    tt.fields.GitConfDir,
				GitProvider:                   tt.fields.GitProvider,
				GitInfo:                       tt.fields.GitInfo,
				PostPreviewJobTimeoutDuration: tt.fields.PostPreviewJobTimeoutDuration,
				PostPreviewJobPollDuration:    tt.fields.PostPreviewJobPollDuration,
				HelmValuesConfig:              tt.fields.HelmValuesConfig,
			}
			if err := o.RunPostPreviewSteps(tt.args.kubeClient, tt.args.ns, tt.args.url, tt.args.pipeline, tt.args.build); (err != nil) != tt.wantErr {
				t.Errorf("PreviewOptions.RunPostPreviewSteps() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPreviewOptions_waitForJobsToComplete(t *testing.T) {
	type fields struct {
		PromoteOptions                PromoteOptions
		Name                          string
		Label                         string
		Namespace                     string
		DevNamespace                  string
		Cluster                       string
		PullRequestURL                string
		PullRequest                   string
		SourceURL                     string
		SourceRef                     string
		Dir                           string
		PostPreviewJobTimeout         string
		PostPreviewJobPollTime        string
		PullRequestName               string
		GitConfDir                    string
		GitProvider                   gits.GitProvider
		GitInfo                       *gits.GitRepositoryInfo
		PostPreviewJobTimeoutDuration time.Duration
		PostPreviewJobPollDuration    time.Duration
		HelmValuesConfig              config.HelmValuesConfig
	}
	type args struct {
		kubeClient kubernetes.Interface
		jobs       []*batchv1.Job
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &PreviewOptions{
				PromoteOptions:                tt.fields.PromoteOptions,
				Name:                          tt.fields.Name,
				Label:                         tt.fields.Label,
				Namespace:                     tt.fields.Namespace,
				DevNamespace:                  tt.fields.DevNamespace,
				Cluster:                       tt.fields.Cluster,
				PullRequestURL:                tt.fields.PullRequestURL,
				PullRequest:                   tt.fields.PullRequest,
				SourceURL:                     tt.fields.SourceURL,
				SourceRef:                     tt.fields.SourceRef,
				Dir:                           tt.fields.Dir,
				PostPreviewJobTimeout:         tt.fields.PostPreviewJobTimeout,
				PostPreviewJobPollTime:        tt.fields.PostPreviewJobPollTime,
				PullRequestName:               tt.fields.PullRequestName,
				GitConfDir:                    tt.fields.GitConfDir,
				GitProvider:                   tt.fields.GitProvider,
				GitInfo:                       tt.fields.GitInfo,
				PostPreviewJobTimeoutDuration: tt.fields.PostPreviewJobTimeoutDuration,
				PostPreviewJobPollDuration:    tt.fields.PostPreviewJobPollDuration,
				HelmValuesConfig:              tt.fields.HelmValuesConfig,
			}
			if err := o.waitForJobsToComplete(tt.args.kubeClient, tt.args.jobs); (err != nil) != tt.wantErr {
				t.Errorf("PreviewOptions.waitForJobsToComplete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPreviewOptions_waitForJob(t *testing.T) {
	type fields struct {
		PromoteOptions                PromoteOptions
		Name                          string
		Label                         string
		Namespace                     string
		DevNamespace                  string
		Cluster                       string
		PullRequestURL                string
		PullRequest                   string
		SourceURL                     string
		SourceRef                     string
		Dir                           string
		PostPreviewJobTimeout         string
		PostPreviewJobPollTime        string
		PullRequestName               string
		GitConfDir                    string
		GitProvider                   gits.GitProvider
		GitInfo                       *gits.GitRepositoryInfo
		PostPreviewJobTimeoutDuration time.Duration
		PostPreviewJobPollDuration    time.Duration
		HelmValuesConfig              config.HelmValuesConfig
	}
	type args struct {
		kubeClient kubernetes.Interface
		job        *batchv1.Job
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &PreviewOptions{
				PromoteOptions:                tt.fields.PromoteOptions,
				Name:                          tt.fields.Name,
				Label:                         tt.fields.Label,
				Namespace:                     tt.fields.Namespace,
				DevNamespace:                  tt.fields.DevNamespace,
				Cluster:                       tt.fields.Cluster,
				PullRequestURL:                tt.fields.PullRequestURL,
				PullRequest:                   tt.fields.PullRequest,
				SourceURL:                     tt.fields.SourceURL,
				SourceRef:                     tt.fields.SourceRef,
				Dir:                           tt.fields.Dir,
				PostPreviewJobTimeout:         tt.fields.PostPreviewJobTimeout,
				PostPreviewJobPollTime:        tt.fields.PostPreviewJobPollTime,
				PullRequestName:               tt.fields.PullRequestName,
				GitConfDir:                    tt.fields.GitConfDir,
				GitProvider:                   tt.fields.GitProvider,
				GitInfo:                       tt.fields.GitInfo,
				PostPreviewJobTimeoutDuration: tt.fields.PostPreviewJobTimeoutDuration,
				PostPreviewJobPollDuration:    tt.fields.PostPreviewJobPollDuration,
				HelmValuesConfig:              tt.fields.HelmValuesConfig,
			}
			if err := o.waitForJob(tt.args.kubeClient, tt.args.job); (err != nil) != tt.wantErr {
				t.Errorf("PreviewOptions.waitForJob() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPreviewOptions_modifyJob(t *testing.T) {
	type fields struct {
		PromoteOptions                PromoteOptions
		Name                          string
		Label                         string
		Namespace                     string
		DevNamespace                  string
		Cluster                       string
		PullRequestURL                string
		PullRequest                   string
		SourceURL                     string
		SourceRef                     string
		Dir                           string
		PostPreviewJobTimeout         string
		PostPreviewJobPollTime        string
		PullRequestName               string
		GitConfDir                    string
		GitProvider                   gits.GitProvider
		GitInfo                       *gits.GitRepositoryInfo
		PostPreviewJobTimeoutDuration time.Duration
		PostPreviewJobPollDuration    time.Duration
		HelmValuesConfig              config.HelmValuesConfig
	}
	type args struct {
		originalJob *batchv1.Job
		envVars     map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *batchv1.Job
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &PreviewOptions{
				PromoteOptions:                tt.fields.PromoteOptions,
				Name:                          tt.fields.Name,
				Label:                         tt.fields.Label,
				Namespace:                     tt.fields.Namespace,
				DevNamespace:                  tt.fields.DevNamespace,
				Cluster:                       tt.fields.Cluster,
				PullRequestURL:                tt.fields.PullRequestURL,
				PullRequest:                   tt.fields.PullRequest,
				SourceURL:                     tt.fields.SourceURL,
				SourceRef:                     tt.fields.SourceRef,
				Dir:                           tt.fields.Dir,
				PostPreviewJobTimeout:         tt.fields.PostPreviewJobTimeout,
				PostPreviewJobPollTime:        tt.fields.PostPreviewJobPollTime,
				PullRequestName:               tt.fields.PullRequestName,
				GitConfDir:                    tt.fields.GitConfDir,
				GitProvider:                   tt.fields.GitProvider,
				GitInfo:                       tt.fields.GitInfo,
				PostPreviewJobTimeoutDuration: tt.fields.PostPreviewJobTimeoutDuration,
				PostPreviewJobPollDuration:    tt.fields.PostPreviewJobPollDuration,
				HelmValuesConfig:              tt.fields.HelmValuesConfig,
			}
			if got := o.modifyJob(tt.args.originalJob, tt.args.envVars); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PreviewOptions.modifyJob() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPreviewOptions_defaultValues(t *testing.T) {
	type fields struct {
		PromoteOptions                PromoteOptions
		Name                          string
		Label                         string
		Namespace                     string
		DevNamespace                  string
		Cluster                       string
		PullRequestURL                string
		PullRequest                   string
		SourceURL                     string
		SourceRef                     string
		Dir                           string
		PostPreviewJobTimeout         string
		PostPreviewJobPollTime        string
		PullRequestName               string
		GitConfDir                    string
		GitProvider                   gits.GitProvider
		GitInfo                       *gits.GitRepositoryInfo
		PostPreviewJobTimeoutDuration time.Duration
		PostPreviewJobPollDuration    time.Duration
		HelmValuesConfig              config.HelmValuesConfig
	}
	type args struct {
		ns              string
		warnMissingName bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &PreviewOptions{
				PromoteOptions:                tt.fields.PromoteOptions,
				Name:                          tt.fields.Name,
				Label:                         tt.fields.Label,
				Namespace:                     tt.fields.Namespace,
				DevNamespace:                  tt.fields.DevNamespace,
				Cluster:                       tt.fields.Cluster,
				PullRequestURL:                tt.fields.PullRequestURL,
				PullRequest:                   tt.fields.PullRequest,
				SourceURL:                     tt.fields.SourceURL,
				SourceRef:                     tt.fields.SourceRef,
				Dir:                           tt.fields.Dir,
				PostPreviewJobTimeout:         tt.fields.PostPreviewJobTimeout,
				PostPreviewJobPollTime:        tt.fields.PostPreviewJobPollTime,
				PullRequestName:               tt.fields.PullRequestName,
				GitConfDir:                    tt.fields.GitConfDir,
				GitProvider:                   tt.fields.GitProvider,
				GitInfo:                       tt.fields.GitInfo,
				PostPreviewJobTimeoutDuration: tt.fields.PostPreviewJobTimeoutDuration,
				PostPreviewJobPollDuration:    tt.fields.PostPreviewJobPollDuration,
				HelmValuesConfig:              tt.fields.HelmValuesConfig,
			}
			if err := o.defaultValues(tt.args.ns, tt.args.warnMissingName); (err != nil) != tt.wantErr {
				t.Errorf("PreviewOptions.defaultValues() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_writePreviewURL(t *testing.T) {
	type args struct {
		o   *PreviewOptions
		url string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writePreviewURL(tt.args.o, tt.args.url)
		})
	}
}

func Test_getContainerRegistry(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getContainerRegistry()
			if (err != nil) != tt.wantErr {
				t.Errorf("getContainerRegistry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getContainerRegistry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getImageName(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getImageName()
			if (err != nil) != tt.wantErr {
				t.Errorf("getImageName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getImageName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getImageTag(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getImageTag()
			if (err != nil) != tt.wantErr {
				t.Errorf("getImageTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getImageTag() = %v, want %v", got, tt.want)
			}
		})
	}
}
*/
