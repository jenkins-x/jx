package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
)

const (
	defaultEnvCtrlReleaseName = "jxet"
)

var (
	createAddonEnvironmentControllerLong = templates.LongDesc(`
		Create an Environment Controller to handle webhooks and promote changes from GitOps 
`)

	createAddonEnvironmentControllerExample = templates.Examples(`
		# Creates the environment controller using a specific environment git repository
		jx create addon envctl -s https://github.com/myorg/environment-production.git
	`)
)

// CreateAddonEnvironmentControllerOptions the options for the create spring command
type CreateAddonEnvironmentControllerOptions struct {
	CreateAddonOptions

	Namespace   string
	Version     string
	ReleaseName string
	SetValues   string
	Timeout     int

	// chart parameters
	WebHookURL        string
	GitSourceURL      string
	GitKind           string
	GitUser           string
	GitToken          string
	BuildPackURL      string
	BuildPackRef      string
	ClusterRBAC       bool
	ProjectID         string
	DockerRegistry    string
	DockerRegistryOrg string
}

// NewCmdCreateAddonEnvironmentController creates a command object for the "create" command
func NewCmdCreateAddonEnvironmentController(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CreateAddonEnvironmentControllerOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "environment controller",
		Short:   "Create an Environment Controller to handle webhooks and promote changes from GitOps",
		Aliases: []string{"envctl"},
		Long:    createAddonEnvironmentControllerLong,
		Example: createAddonEnvironmentControllerExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The namespace to install the controller")
	cmd.Flags().StringVarP(&options.ReleaseName, optionRelease, "r", defaultEnvCtrlReleaseName, "The chart release name")
	cmd.Flags().StringVarP(&options.SetValues, "set", "", "", "The chart set values (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	cmd.Flags().StringVarP(&options.Version, "version", "", "", "The version of the chart to use - otherwise the latest version is used")
	cmd.Flags().IntVarP(&options.Timeout, "timeout", "", 600000, "The timeout value for how long to wait for the install to succeed")
	cmd.Flags().StringVarP(&options.GitSourceURL, "source-url", "s", "", "The git URL of the environment repository to promote from")
	cmd.Flags().StringVarP(&options.GitKind, "git-kind", "", "", "The kind of git repository. Should be one of: "+strings.Join(gits.KindGits, ", "))
	cmd.Flags().StringVarP(&options.GitUser, "user", "u", "", "The git user to use to clone and tag the git repository")
	cmd.Flags().StringVarP(&options.GitToken, "token", "t", "", "The git token to clone and tag the git repository")
	cmd.Flags().StringVarP(&options.WebHookURL, "webhook-url", "w", "", "The webhook URL used to expose the exposecontroller and register with the git provider's webhooks")
	cmd.Flags().StringVarP(&options.BuildPackURL, "buildpack-url", "", "", "The URL for the build pack Git repository")
	cmd.Flags().StringVarP(&options.BuildPackRef, "buildpack-ref", "", "", "The Git reference (branch,tag,sha) in the Git repository to use")
	cmd.Flags().StringVarP(&options.ProjectID, "project-id", "", "", "The cloud project ID")
	cmd.Flags().BoolVarP(&options.ClusterRBAC, "cluster-rbac", "", false, "Whether to enable cluster level RBAC on Tekton")
	cmd.Flags().StringVarP(&options.DockerRegistry, "docker-registry", "", "", "The Docker Registry host name to use which is added as a prefix to docker images")
	cmd.Flags().StringVarP(&options.DockerRegistryOrg, "docker-registry-org", "", "", "The Docker registry organisation. If blank the git repository owner is used")
	return cmd
}

// Run implements the command
func (o *CreateAddonEnvironmentControllerOptions) Run() error {
	o.EnableRemoteKubeCluster()

	_, ns, err := o.KubeClientAndNamespace()
	if err != nil {
		return err
	}
	if o.Namespace == "" {
		o.Namespace = ns
	}

	if o.GitSourceURL == "" {
		if o.BatchMode {
			return util.MissingOption("source-url")
		}
		o.GitSourceURL, err = util.PickValue("git repository to promote from: ", "", true, "please specify the GitOps repository used to store the kubernetes applications to deploy to this cluster", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
	}
	gitInfo, err := gits.ParseGitURL(o.GitSourceURL)
	if err != nil {
		return err
	}
	serverUrl := gitInfo.ProviderURL()
	if o.GitKind == "" {
		o.GitKind = gits.SaasGitKind(serverUrl)
	}
	if o.GitKind == "" {
		if o.BatchMode {
			return util.MissingOption("git-kind")
		}
		o.GitKind, err = util.PickName(gits.KindGits, "kind of git repository: ", "please specify the GitOps repository used to store the kubernetes applications to deploy to this cluster", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
	}

	apisClient, err := o.ApiExtensionsClient()
	if err != nil {
		return errors.Wrap(err, "failed to create the API extensions client")
	}
	err = kube.RegisterPipelineCRDs(apisClient)
	if err != nil {
		return errors.Wrap(err, "failed to register the Jenkins X Pipeline CRDs")
	}

	authSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return err
	}
	config, err := authSvc.LoadConfig()
	if err != nil {
		return err
	}
	server := config.GetOrCreateServer(serverUrl)

	if o.GitUser == "" {
		auth, err := o.PickPipelineUserAuth(config, server)
		if err != nil {
			return err
		}
		if auth == nil {
			return fmt.Errorf("no user found for git server %s", serverUrl)
		}
		o.GitUser = auth.Username
		if o.GitToken == "" {
			o.GitToken = auth.ApiToken
		}
	}
	if o.GitToken == "" {
		auth := server.GetUserAuth(o.GitUser)
		if auth != nil {
			o.GitToken = auth.ApiToken
		} else {
			return util.MissingOption("token")
		}
	}
	if o.GitUser == "" {
		return util.MissingOption("user")
	}
	if o.GitToken == "" {
		return util.MissingOption("token")
	}

	setValues := []string{}
	if o.SetValues != "" {
		setValues = strings.Split(o.SetValues, ",")
	}
	if o.WebHookURL != "" {
		setValues = append(setValues, "webhookUrl="+o.WebHookURL)
	}
	setValues = append(setValues, "source.owner="+gitInfo.Organisation)
	setValues = append(setValues, "source.repo="+gitInfo.Name)
	setValues = append(setValues, "source.serverUrl="+serverUrl)
	setValues = append(setValues, "source.gitKind="+o.GitKind)
	setValues = append(setValues, "source.user="+o.GitUser)
	setValues = append(setValues, "source.token="+o.GitToken)
	if o.ProjectID != "" {
		setValues = append(setValues, "projectId="+o.ProjectID)
	}
	if o.BuildPackURL != "" {
		setValues = append(setValues, "buildPackURL="+o.BuildPackURL)
	}
	if o.BuildPackRef != "" {
		setValues = append(setValues, "buildPackRef="+o.BuildPackRef)
	}
	if o.DockerRegistry != "" {
		setValues = append(setValues, "dockerRegistry="+o.DockerRegistry)
	}
	if o.DockerRegistryOrg != "" {
		setValues = append(setValues, "dockerRegistryOrg="+o.DockerRegistryOrg)
	}
	setValues = append(setValues, "tekton.rbac.cluster="+strconv.FormatBool(o.ClusterRBAC))

	log.Infof("installing the Environment Controller with values: %s\n", util.ColorInfo(strings.Join(setValues, ",")))
	helmOptions := helm.InstallChartOptions{
		Chart:          "environment-controller",
		ReleaseName:    o.ReleaseName,
		Version:        o.Version,
		Ns:             ns,
		SetValues:      setValues,
		HelmUpdate:     true,
		Repository:     kube.DefaultChartMuseumURL,
		VersionsGitURL: opts.DefaultVersionsURL,
	}
	err = o.InstallChartWithOptions(helmOptions)
	if err != nil {
		return err
	}
	log.Infof("installed the Environment Controller!\n")
	return nil
}
