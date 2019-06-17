package verify

import (
	"github.com/cloudflare/cfssl/log"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/spf13/cobra"
)

// StepVerifyInstallOptions contains the command line flags
type StepVerifyInstallOptions struct {
	opts.StepOptions
	Debug bool
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

	log.Infof("installation looks good!\n")
	return nil
}
