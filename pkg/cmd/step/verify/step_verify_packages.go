package verify

import (
	"github.com/cloudflare/cfssl/log"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	verifyPackagesLong = templates.LongDesc(`
		Verifies the versions of the required command line packages


` + opts.SeeAlsoText("jx create project"))

	verifyPackagesExample = templates.Examples(`
		Verifies the versions of the required command line packages

		# verify packages and fail if any are not valid:
		jx step verify packages

		# override the error if the 'jx' binary is out of range (e.g. for development)
        export JX_DISABLE_VERIFY_JX="true"
		jx step verify packages
	`)
)

// StepVerifyPackagesOptions contains the command line flags
type StepVerifyPackagesOptions struct {
	opts.StepOptions

	Namespace string
	HelmTLS   bool
	Packages  []string
}

// NewCmdStepVerifyPackages creates the `jx step verify pod` command
func NewCmdStepVerifyPackages(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepVerifyPackagesOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "packages",
		Aliases: []string{"package"},
		Short:   "Verifies the versions of the required command line packages",
		Long:    verifyPackagesLong,
		Example: verifyPackagesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.HelmTLS, "helm-tls", "", false, "Whether to use TLS with helm")
	cmd.Flags().StringVarP(&options.Namespace, "namespace", "n", "", "The namespace to use to look for helm's tiller")
	cmd.Flags().StringArrayVarP(&options.Packages, "packages", "p", []string{"jx", "kubectl", "git", "helm", "kaniko"}, "The packages to verify")

	return cmd
}

// Run implements this command
func (o *StepVerifyPackagesOptions) Run() error {
	log.Infof("verifying the CLI packages\n")

	packages, table := o.GetPackageVersions(o.Namespace, o.HelmTLS)

	verifyMap := map[string]string{}
	for _, k := range o.Packages {
		verifyMap[k] = packages[k]
	}

	resolver, err := o.CreateVersionResolver("", "")
	if err != nil {
		return errors.Wrapf(err, "failed to create version resolver")
	}

	err = resolver.VerifyPackages(verifyMap)
	if err != nil {
		return err
	}

	log.Infof("the CLI packages seem to be setup correctly\n")
	table.Render()

	return nil
}
