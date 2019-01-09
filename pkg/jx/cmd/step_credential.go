package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
)

// StepCredentialOptions contains the command line arguments for this command
type StepCredentialOptions struct {
	StepOptions

	Namespace string
	Secret    string
	Key       string
}

var (
	stepCredentialLong = templates.LongDesc(`
		This pipeline step lets you run the BDD tests in the current team in a current cluster or create a new cluster/team run tests there then tear things down again.

`)

	stepCredentialExample = templates.Examples(`
		# run the BDD tests in the current team
		jx step bdd --use-current-team --git-provider-url=https://my.git.server.com

        # create a new team for the tests, run the tests then tear everything down again 
		jx step bdd -b --provider=gke --git-provider=ghe --git-provider-url=https://my.git.server.com --default-admin-password=myadminpwd --git-username myuser --git-api-token mygittoken
`)
)

// NewCmdStepCredential creates the command
func NewCmdStepCredential(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := StepCredentialOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "credential",
		Short:   "Returns a secret entry for easy scripting in pipeline steps",
		Long:    stepCredentialLong,
		Example: stepCredentialExample,
		Aliases: []string{"secret", "cred"},
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	options.addCommonFlags(cmd)
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "the namespace to look for a Secret")
	cmd.Flags().StringVarP(&options.Secret, "name", "s", "", "the name of the Secret")
	cmd.Flags().StringVarP(&options.Key, "key", "k", "", "the key in the Secret to output")

	return cmd
}

// Run runs the command
func (o *StepCredentialOptions) Run() error {
	kubeClient, devNs, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}

	ns := o.Namespace
	if ns == "" {
		ns = devNs
	}


	name := o.Secret
	if name == "" {
		return util.MissingOption("name")
	}
	secret, err := kubeClient.CoreV1().Secrets(ns).Get(name, metav1.GetOptions{})
	if err != nil {
	  return errors.Wrapf(err, "failed to find Secret %s in namespace %s", name, ns)
	}
	if secret.Data == nil {
		return errors.Wrapf(err, "Secret %s in namespace %s has no data", name, ns)
	}
	keys := []string{}
	for k := range secret.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	key := o.Key
	if key == "" {
		return util.MissingOptionWithOptions("key", keys)
	}

	value := secret.Data[key]
	if value == nil {
		log.Warnf("Secret %s in namespace %s does not have key %s\n", name, ns, key)
		return util.InvalidOption("key", key, keys)
	}
	log.Infof("%s\n", string(value))
	return nil
}
