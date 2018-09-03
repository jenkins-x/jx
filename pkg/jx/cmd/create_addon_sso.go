package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	survey "gopkg.in/AlecAivazis/survey.v1"

	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
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

const (
	defaultSSONamesapce   = "jx"
	defaultSSOReleaseName = "jx"
	repoName              = "jenkinsxio"
	repoURL               = "https://chartmuseum.jx.cd.jenkins-x.io"
	dexChart              = "jenkinsxio/dex"
	dexServiceName        = "dex"
	dexChartVersion       = ""
	operatorChart         = "jenkinsxio/sso-operator"
	operatorChartVersion  = ""
	operatorServiceName   = "sso-operator"
	githubNewOAuthAppURL  = "https://github.com/settings/applications/new"
)

// CreateAddonSSOptions the options for the create sso addon
type CreateAddonSSOOptions struct {
	CreateAddonOptions
	UpgradeIngressOptions UpgradeIngressOptions
}

// NewCmdCreateAddonSSO creates a command object for the "create sso" command
func NewCmdCreateAddonSSO(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	commonOptions := CommonOptions{
		Factory: f,
		Out:     out,
		Err:     errOut,
	}
	options := &CreateAddonSSOOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: commonOptions,
			},
		},
		UpgradeIngressOptions: UpgradeIngressOptions{
			CreateOptions: CreateOptions{
				CommonOptions: commonOptions,
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
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, defaultSSONamesapce, defaultSSOReleaseName)
	options.UpgradeIngressOptions.addFlags(cmd)
	return cmd
}

// Run implements the command
func (o *CreateAddonSSOOptions) Run() error {
	_, _, err := o.KubeClient()
	if err != nil {
		return fmt.Errorf("cannot connect to kubernetes cluster: %v", err)
	}
	o.devNamespace, _, err = kube.GetDevNamespace(o.KubeClientCached, o.currentNamespace)
	if err != nil {
		return errors.Wrap(err, "retrieving the development namesapce")
	}

	err = o.ensureCertmanager()
	if err != nil {
		return errors.Wrap(err, "ensuring cert-manager is installed")
	}

	log.Infof("Installing %s...\n", util.ColorInfo("dex identity provider"))

	ingressConfig, err := kube.GetIngressConfig(o.KubeClientCached, o.devNamespace)
	if err != nil {
		return errors.Wrap(err, "retrieving existing ingress configuration")
	}
	domain, err := util.PickValue("Domain:", ingressConfig.Domain, true)
	if err != nil {
		return err
	}

	log.Infof("Configuring %s connector\n", util.ColorInfo("GitHub"))

	log.Infof("Please go to %s and create a new OAuth application with %s callback\n",
		util.ColorInfo(githubNewOAuthAppURL), util.ColorInfo(o.dexCallback(domain)))
	log.Infof("Then copy the %s and %s so that you can pate them into the form bellow:\n",
		util.ColorInfo("Client ID"), util.ColorInfo("Client Secret"))

	clientID, err := util.PickValue("Client ID:", "", true)
	if err != nil {
		return err
	}
	clientSecret, err := util.PickPassword("Client Secret:")
	if err != nil {
		return err
	}
	authorizedOrgs, err := o.getAuthorizedOrgs()
	if err != nil {
		return err
	}

	err = o.ensureHelm()
	if err != nil {
		return errors.Wrap(err, "checking if helm is installed")
	}

	err = o.addHelmRepoIfMissing(repoURL, repoName)
	if err != nil {
		return errors.Wrap(err, "adding dex chart helm repository")
	}

	err = o.installDex(o.dexDomain(domain), clientID, clientSecret, authorizedOrgs)
	if err != nil {
		return errors.Wrap(err, "installing dex")
	}

	log.Infof("Installing %s...\n", util.ColorInfo("sso-operator"))
	dexGrpcService := fmt.Sprintf("%s.%s", dexServiceName, o.Namespace)
	err = o.installSSOOperator(dexGrpcService)
	if err != nil {
		return errors.Wrap(err, "installing sso-operator")
	}

	log.Infof("Exposing services with %s enabled...\n", util.ColorInfo("TLS"))
	return o.exposeSSO()
}

func (o *CreateAddonSSOOptions) dexDomain(domain string) string {
	return fmt.Sprintf("%s.%s.%s", dexServiceName, o.Namespace, domain)
}

func (o *CreateAddonSSOOptions) dexCallback(domain string) string {
	return fmt.Sprintf("https://%s/callback", o.dexDomain(domain))
}

func (o *CreateAddonSSOOptions) getAuthorizedOrgs() ([]string, error) {
	authConfigSvc, err := o.CreateGitAuthConfigService()
	if err != nil {
		return nil, err
	}
	config := authConfigSvc.Config()
	server := config.GetOrCreateServer(gits.GitHubURL)
	userAuth, err := config.PickServerUserAuth(server, "git user name", true)
	if err != nil {
		return nil, err
	}
	provider, err := gits.CreateProvider(server, userAuth, o.Git())
	if err != nil {
		return nil, err
	}

	orgs := gits.GetOrganizations(provider, userAuth.Username)
	sort.Strings(orgs)
	promt := &survey.MultiSelect{
		Message: "Authorize access to all users from GitHub organizations:",
		Options: orgs,
	}

	authorizedOrgs := []string{}
	err = survey.AskOne(promt, &authorizedOrgs, nil)
	return authorizedOrgs, err
}

func (o *CreateAddonSSOOptions) installDex(domain string, clientID string, clientSecret string, authorizedOrgs []string) error {
	values := []string{
		"connectors.github.config.clientID=" + clientID,
		"connectors.github.config.clientSecret=" + clientSecret,
		fmt.Sprintf("connectors.github.config.orgs={%s}", strings.Join(authorizedOrgs, ",")),
		"domain=" + domain,
		"certs.grpc.ca.namespace=" + CertManagerNamespace,
	}
	setValues := strings.Split(o.SetValues, ",")
	values = append(values, setValues...)
	releaseName := o.ReleaseName + "-sso-" + dexServiceName
	return o.installChart(releaseName, dexChart, dexChartVersion, o.Namespace, true, values)
}

func (o *CreateAddonSSOOptions) installSSOOperator(dexGrpcService string) error {
	values := []string{
		"dex.grpcHost=" + dexGrpcService,
	}
	setValues := strings.Split(o.SetValues, ",")
	values = append(values, setValues...)
	releaseName := o.ReleaseName + "-" + operatorServiceName
	return o.installChart(releaseName, operatorChart, operatorChartVersion, o.Namespace, true, values)
}

func (o *CreateAddonSSOOptions) exposeSSO() error {
	options := &o.UpgradeIngressOptions
	options.Namespaces = []string{o.Namespace}
	options.SkipCertManager = true
	return options.Run()
}
