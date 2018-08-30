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
	defaultSSOReleaseName = "sso"
	repoName              = "jenkinsxio"
	repoURL               = "https://chartmuseum.jx.cd.jenkins-x.io"
	dexChart              = "jenkinsxio/dex"
	dexServiceName        = "dex"
	dexChartVersion       = ""
	githubNewOAuthAppURL  = "https://github.com/settings/applications/new"
)

// CreateAddonSSOptions the options for the create sso addon
type CreateAddonSSOOptions struct {
	CreateAddonOptions
}

// NewCmdCreateAddonSSO creates a command object for the "create sso" command
func NewCmdCreateAddonSSO(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateAddonSSOOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					Out:     out,
					Err:     errOut,
				},
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
		return errors.Wrap(err, "retrieveing the development namesapce")
	}

	err = o.ensureCertmanager()
	if err != nil {
		return errors.Wrap(err, "ensureing cert-manager is installed")
	}

	ingressConfig, err := kube.GetIngressConfig(o.KubeClientCached, o.devNamespace)
	if err != nil {
		return errors.Wrap(err, "retrieveing existing ingress configuration")
	}
	domain, err := util.PickValue("Domain:", ingressConfig.Domain, true)
	if err != nil {
		return err
	}

	log.Infof("Cofiguring %s connector\n", util.ColorInfo("GitHub"))

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

	return o.installDex(o.dexDomain(domain), clientID, clientSecret, authorizedOrgs)
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
		Message: "Authorized GitHub Organizations:",
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
	return o.installChart(o.ReleaseName, dexChart, dexChartVersion, o.Namespace, true, values)
}
