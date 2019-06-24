package verify

import (
	"github.com/cloudflare/cfssl/log"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StepVerifyInstallOptions contains the command line flags
type StepVerifyInstallOptions struct {
	opts.StepOptions
	Debug bool
	Dir   string
}

// NewCmdStepVerifyInstall creates the `jx step verify pod` command
func NewCmdStepVerifyInstall(commonOpts *opts.CommonOptions) *cobra.Command {

	options := &StepVerifyInstallOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use: "install",
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Debug, "debug", "", false, "Output logs of any failed pod")
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", ".", "the directory to look for the install requirements file")
	return cmd
}

// Run implements this command
func (o *StepVerifyInstallOptions) Run() error {
	log.Infof("verifying the Jenkins X installation\n")

	po := &StepVerifyPodReadyOptions{}
	po.StepOptions = o.StepOptions
	po.Debug = o.Debug

	log.Info("verifying pods\n")
	err := po.Run()
	if err != nil {
		return err
	}

	gto := &StepVerifyGitOptions{}
	gto.StepOptions = o.StepOptions
	err = gto.Run()
	if err != nil {
		return err
	}

	requirements, _, err := config.LoadRequirementsConfig(o.Dir)
	if err != nil {
		return err
	}
	if requirements.Kaniko {
		err = o.validateKaniko()
		if err != nil {
			return err
		}
	}
	log.Infof("installation looks good!\n")
	return nil
}

func (o *StepVerifyInstallOptions) validateKaniko() error {
	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return err
	}
	secret, err := kubeClient.CoreV1().Secrets(ns).Get(kube.SecretKaniko, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "could not find the Secret %s in the namespace %s", kube.SecretKaniko, ns)
	}
	if secret.Data == nil || len(secret.Data["kube.SecretKaniko"]) == 0 {
		return errors.Wrapf(err, "the Secret %s in the namespace %s does not have a key %s", kube.SecretKaniko, ns, kube.SecretKaniko)
	}
	log.Infof("kaniko is valid: there is a Secret %s in namespace %s\n", util.ColorInfo(kube.SecretKaniko), util.ColorInfo(ns))
	return nil
}
