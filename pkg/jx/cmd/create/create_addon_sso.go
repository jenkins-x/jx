package create

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/jx/cmd/upgrade"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	"github.com/jenkins-x/jx/pkg/helm"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	survey "gopkg.in/AlecAivazis/survey.v1"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/kube/pki"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	defaultSSONamesapce         = "sso"
	defaultSSOReleaseNamePrefix = "jx-sso"
	repoName                    = "jenkins-x"
	dexServiceName              = "dex"
	operatorServiceName         = "operator"
	githubNewOAuthAppURL        = "https://github.com/settings/applications/new"
	defaultDexVersion           = ""
	defaultOperatorVersion      = ""
)

var (
	CreateAddonSSOLong = templates.LongDesc(`
		Creates the Single Sign-On addon

		This addon will install and configure the dex identity provider, sso-operator and cert-manager.
`)

	CreateAddonSSOExample = templates.Examples(`
		# Create the sso addon
		jx create addon sso
	`)
)

// CreateAddonSSOptions the options for the create sso addon
type CreateAddonSSOOptions struct {
	CreateAddonOptions
	DexVersion string
}

// NewCmdCreateAddonSSO creates a command object for the "create addon sso" command
func NewCmdCreateAddonSSO(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &CreateAddonSSOOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "sso",
		Short:   "Create a SSO addon for Single Sign-On",
		Long:    CreateAddonSSOLong,
		Example: CreateAddonSSOExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.DexVersion, "dex-version", "", defaultDexVersion, "The dex chart version to install)")
	options.addFlags(cmd, defaultSSONamesapce, defaultSSOReleaseNamePrefix, defaultOperatorVersion)
	return cmd
}

// Run implements the command
func (o *CreateAddonSSOOptions) Run() error {
	client, devNamespace, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return errors.Wrap(err, "retrieving the development namespace")
	}

	err = o.EnsureCertManager()
	if err != nil {
		return errors.Wrap(err, "ensuring cert-manager is installed")
	}

	log.Logger().Infof("Installing %s...", util.ColorInfo("dex identity provider"))

	ingressConfig, err := kube.GetIngressConfig(client, devNamespace)
	if err != nil {
		return errors.Wrap(err, "retrieving existing ingress configuration")
	}
	domain, err := util.PickValue("Domain:", ingressConfig.Domain, true, "", o.In, o.Out, o.Err)
	if err != nil {
		return err
	}

	log.Logger().Infof("Configuring %s connector", util.ColorInfo("GitHub"))

	log.Logger().Infof("Please go to %s and create a new OAuth application with an Authorization Callback URL of %s.\nChoose a suitable Application name and Homepage URL.",
		util.ColorInfo(githubNewOAuthAppURL), util.ColorInfo(o.dexCallback(domain)))
	log.Logger().Infof("Copy the %s and the %s and paste them into the form below:",
		util.ColorInfo("Client ID"), util.ColorInfo("Client Secret"))

	clientID, err := util.PickValue("Client ID:", "", true, "", o.In, o.Out, o.Err)
	if err != nil {
		return err
	}
	clientSecret, err := util.PickPassword("Client Secret:", "", o.In, o.Out, o.Err)
	if err != nil {
		return err
	}
	authorizedOrgs, err := o.getAuthorizedOrgs()
	if err != nil {
		return err
	}

	err = o.EnsureHelm()
	if err != nil {
		return errors.Wrap(err, "checking if helm is installed")
	}

	_, err = o.AddHelmBinaryRepoIfMissing(kube.DefaultChartMuseumURL, repoName, "", "")
	if err != nil {
		return errors.Wrap(err, "adding dex chart helm repository")
	}

	err = o.installDex(o.dexDomain(domain), clientID, clientSecret, authorizedOrgs)
	if err != nil {
		return errors.Wrap(err, "installing dex")
	}

	log.Logger().Infof("Installing %s...", util.ColorInfo("sso-operator"))
	dexGrpcService := fmt.Sprintf("%s.%s", dexServiceName, o.Namespace)
	err = o.installSSOOperator(dexGrpcService)
	if err != nil {
		return errors.Wrap(err, "installing sso-operator")
	}

	log.Logger().Infof("Exposing services with %s enabled...", util.ColorInfo("TLS"))
	return o.exposeSSO()
}

func (o *CreateAddonSSOOptions) dexDomain(domain string) string {
	return fmt.Sprintf("%s.%s.%s", dexServiceName, o.Namespace, domain)
}

func (o *CreateAddonSSOOptions) dexCallback(domain string) string {
	return fmt.Sprintf("https://%s/callback", o.dexDomain(domain))
}

func (o *CreateAddonSSOOptions) getAuthorizedOrgs() ([]string, error) {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return nil, err
	}
	config := authConfigSvc.Config()
	server := config.GetOrCreateServer(gits.GitHubURL)
	userAuth, err := config.PickServerUserAuth(server, "Git user name", true, "", o.In, o.Out, o.Err)
	if err != nil {
		return nil, err
	}
	provider, err := gits.CreateProvider(server, userAuth, o.Git())
	if err != nil {
		return nil, err
	}

	orgs := gits.GetOrganizations(provider, userAuth.Username)
	if len(orgs) == 0 {
		return nil, fmt.Errorf("user '%s' does not have any GitHub organizations", userAuth.Username)
	}

	orgChecker, ok := provider.(gits.OrganisationChecker)
	if !ok || orgChecker == nil {
		return nil, errors.New("failed to create the GitHub organisation checker")
	}
	orgsWithMembers := []string{}
	for _, org := range orgs {
		member, err := orgChecker.IsUserInOrganisation(userAuth.Username, org)
		if err != nil {
			continue
		}
		if member {
			orgsWithMembers = append(orgsWithMembers, org)
		}
	}

	if len(orgsWithMembers) == 0 {
		return nil, fmt.Errorf("user '%s' is not member of any GitHub organizations", userAuth.Username)
	}

	sort.Strings(orgsWithMembers)

	promt := &survey.MultiSelect{
		Message: "Select GitHub organizations to authorize users from:",
		Options: orgsWithMembers,
	}

	authorizedOrgs := []string{}
	err = survey.AskOne(promt, &authorizedOrgs, nil, surveyOpts)
	return authorizedOrgs, err
}

func (o *CreateAddonSSOOptions) installDex(domain string, clientID string, clientSecret string, authorizedOrgs []string) error {
	values := []string{
		"connectors.github.config.clientID=" + clientID,
		"connectors.github.config.clientSecret=" + clientSecret,
		fmt.Sprintf("connectors.github.config.orgs={%s}", strings.Join(authorizedOrgs, ",")),
		"domain=" + domain,
		"certs.grpc.ca.namespace=" + pki.CertManagerNamespace,
	}
	setValues := strings.Split(o.SetValues, ",")
	values = append(values, setValues...)
	releaseName := o.ReleaseName + "-" + dexServiceName
	helmOptions := helm.InstallChartOptions{
		Chart:       kube.ChartSsoDex,
		ReleaseName: releaseName,
		Version:     o.DexVersion,
		Ns:          o.Namespace,
		SetValues:   values,
	}
	return o.InstallChartWithOptions(helmOptions)
}

func (o *CreateAddonSSOOptions) installSSOOperator(dexGrpcService string) error {
	values := []string{
		"dex.grpcHost=" + dexGrpcService,
	}
	setValues := strings.Split(o.SetValues, ",")
	values = append(values, setValues...)
	releaseName := o.ReleaseName + "-" + operatorServiceName
	helmOptions := helm.InstallChartOptions{
		Chart:       kube.ChartSsoOperator,
		ReleaseName: releaseName,
		Version:     o.DexVersion,
		Ns:          o.Namespace,
		SetValues:   values,
	}
	return o.InstallChartWithOptions(helmOptions)
}

func (o *CreateAddonSSOOptions) exposeSSO() error {
	upgradeIngOpts := &upgrade.UpgradeIngressOptions{
		CommonOptions: o.CommonOptions,
		Namespaces:    []string{o.Namespace},
		WaitForCerts:  true,
	}
	return upgradeIngOpts.Run()
}
