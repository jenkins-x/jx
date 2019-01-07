package cmd

import (
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/kube/services"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"fmt"

	"time"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
)

const (
	defaultAnchoreName        = "anchore"
	defaultAnchoreNamespace   = "anchore"
	defaultAnchoreReleaseName = "anchore"
	defaultAnchoreVersion     = "0.2.3"
	defaultAnchorePassword    = "anchore"
	defaultAnchoreConfigDir   = "/anchore_service_dir"
	anchoreDeploymentName     = "anchore-anchore-engine-core"
)

var (
	createAddonAnchoreLong = templates.LongDesc(`
		Creates the anchore addon for serverless on kubernetes
`)

	createAddonAnchoreExample = templates.Examples(`
		# Create the anchore addon 
		jx create addon anchore

		# Create the anchore addon in a custom namespace
		jx create addon anchore -n mynamespace
	`)
)

// CreateAddonAnchoreOptions the options for the create spring command
type CreateAddonAnchoreOptions struct {
	CreateAddonOptions

	Chart     string
	Password  string
	ConfigDir string
}

// NewCmdCreateAddonAnchore creates a command object for the "create" command
func NewCmdCreateAddonAnchore(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateAddonAnchoreOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					In:      in,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "anchore",
		Short:   "Create the Anchore addon for verifying container images",
		Aliases: []string{"env"},
		Long:    createAddonAnchoreLong,
		Example: createAddonAnchoreExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	options.addFlags(cmd, defaultAnchoreNamespace, defaultAnchoreReleaseName, defaultAnchoreVersion)

	cmd.Flags().StringVarP(&options.Password, "password", "p", defaultAnchorePassword, "The default password to use for Anchore")
	cmd.Flags().StringVarP(&options.ConfigDir, "config-dir", "d", defaultAnchoreConfigDir, "The config directory to use")
	cmd.Flags().StringVarP(&options.Chart, optionChart, "c", kube.ChartAnchore, "The name of the chart to use")
	return cmd
}

// Run implements the command
func (o *CreateAddonAnchoreOptions) Run() error {
	err := o.ensureHelm()
	if err != nil {
		return errors.Wrap(err, "failed to ensure that helm is present")
	}

	if o.ReleaseName == "" {
		return util.MissingOption(optionRelease)
	}
	if o.Chart == "" {
		return util.MissingOption(optionChart)
	}
	client, err := o.KubeClient()
	if err != nil {
		return err
	}

	devNamespace, _, err := kube.GetDevNamespace(client, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}

	log.Infof("found dev namespace %s\n", devNamespace)

	values := []string{"globalConfig.users.admin.password=" + o.Password, "globalConfig.configDir=/anchore_service_dir"}
	setValues := strings.Split(o.SetValues, ",")
	values = append(values, setValues...)
	err = o.installChart(o.ReleaseName, o.Chart, o.Version, o.Namespace, true, values, nil, "")
	if err != nil {
		return fmt.Errorf("anchore deployment failed: %v", err)
	}

	log.Info("waiting for anchore deployment to be ready, this can take a few minutes\n")

	err = kube.WaitForDeploymentToBeReady(client, anchoreDeploymentName, o.Namespace, 10*time.Minute)
	if err != nil {
		return err
	}

	anchoreServiceName, ok := kube.AddonServices[defaultAnchoreName]
	if !ok {
		return errors.New("no service name defined for anchore chart")
	}

	err = o.CreateAddonOptions.ExposeAddon(defaultAnchoreName)
	if err != nil {
		return err
	}

	// get the external anchore services URL
	ing, err := services.GetServiceURLFromName(client, anchoreServiceName, o.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get external URL for service %s: %v", anchoreServiceName, err)
	}

	// create the local addonAuth.yaml file so `jx get cve` commands work
	tokenOptions := CreateTokenAddonOptions{
		Password: o.Password,
		Username: "admin",
		ServerFlags: ServerFlags{
			ServerURL:  ing,
			ServerName: anchoreDeploymentName,
		},
		Kind: kube.ValueKindCVE,
		CreateOptions: CreateOptions{
			CommonOptions: o.CommonOptions,
		},
	}
	err = tokenOptions.Run()
	if err != nil {
		return fmt.Errorf("failed to create addonAuth.yaml error: %v", err)
	}

	_, err = client.CoreV1().Services(o.currentNamespace).Get(anchoreServiceName, meta_v1.GetOptions{})
	if err != nil {
		// create a service link
		err = services.CreateServiceLink(client, o.currentNamespace, o.Namespace, anchoreServiceName, ing)
		if err != nil {
			return fmt.Errorf("failed creating a service link for %s in target namespace %s", anchoreServiceName, o.Namespace)
		}

	}
	return nil
}
