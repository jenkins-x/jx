package step

import (
	"fmt"
	"sort"

	"github.com/jenkins-x/jx/pkg/cmd/opts/step"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StepCredentialOptions contains the command line arguments for this command
type StepCredentialOptions struct {
	step.StepOptions

	Namespace string
	Secret    string
	Key       string
	Optional  bool
}

var (
	stepCredentialLong = templates.LongDesc(`
		Returns a credential from a Secret for easy scripting in pipeline steps.

		Supports the [Jenkins Credentials Provider labels on the Secrets](https://jenkinsci.github.io/kubernetes-credentials-provider-plugin/examples/)

		If you specify --optional then if the key or secret doesn't exist then the command will only print a warning and will not error.
`)

	stepCredentialExample = templates.Examples(`
		# get the password of a secret 'foo' which uses the Jenkins Credentials Provider labels
		export MY_PWD="$(jx step credential -s foo)"

		# get the password entry of a secret 'foo' as an environment variable
		export MY_PWD="$(jx step credential -s foo -k passwordj)"

		# create a local file from a file based secret using the Jenkins Credentials Provider labels
        export MY_KEY_FILE="$(jx step credential -s foo)"
`)
)

// NewCmdStepCredential creates the command
func NewCmdStepCredential(commonOpts *opts.CommonOptions) *cobra.Command {
	options := StepCredentialOptions{
		StepOptions: step.StepOptions{
			CommonOptions: commonOpts,
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
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "the namespace to look for a Secret")
	cmd.Flags().StringVarP(&options.Secret, "name", "s", "", "the name of the Secret")
	cmd.Flags().StringVarP(&options.Key, "key", "k", "", "the key in the Secret to output")
	cmd.Flags().BoolVarP(&options.Optional, "optional", "", false, "if true, then the command will only warn (not error) if the secret or the key doesn't exist")

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
		if o.Optional {
			log.Logger().Warnf("failed to find Secret %s in namespace %s", name, ns)
			return nil
		}
		return errors.Wrapf(err, "failed to find Secret %s in namespace %s", name, ns)
	}
	data := secret.Data
	if data == nil {
		if o.Optional {
			log.Logger().Warnf("Secret %s in namespace %s has no data", name, ns)
			return nil
		}
		return errors.Wrapf(err, "Secret %s in namespace %s has no data", name, ns)
	}
	keys := []string{}
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	key := o.Key

	labels := secret.Labels
	if labels != nil {
		kind := labels[kube.LabelCredentialsType]
		if key == "" && kind == kube.ValueCredentialTypeUsernamePassword {
			key = "password"
		}
	}

	if key == "" {
		return util.MissingOptionWithOptions("key", keys)
	}

	value, ok := data[key]
	if !ok {
		log.Logger().Warnf("Secret %s in namespace %s does not have key %s", name, ns, key)
		if o.Optional {
			return nil
		}
		return util.InvalidOption("key", key, keys)
	}
	fmt.Fprintf(o.Out, "%s\n", value)
	return nil
}
