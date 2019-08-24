package verify

import (
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/log"
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
	step.StepOptions

	Namespace string
	HelmTLS   bool
	Packages  []string
	Dir       string
}

// NewCmdStepVerifyPackages creates the `jx step verify pod` command
func NewCmdStepVerifyPackages(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepVerifyPackagesOptions{
		StepOptions: step.StepOptions{
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
	cmd.Flags().StringVarP(&options.Dir, "dir", "d", ".", "the directory to recursively look upwards for any 'jx-requirements.yml' file to determine the version stream")
	return cmd
}

// Run implements this command
func (o *StepVerifyPackagesOptions) Run() error {
	log.Logger().Infof("verifying the CLI packages\n")

	packages, table := o.GetPackageVersions(o.Namespace, o.HelmTLS)

	verifyMap := map[string]string{}
	for _, k := range o.Packages {
		verifyMap[k] = packages[k]
	}

	requirements, _, err := config.LoadRequirementsConfig(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to load boot requirements")
	}
	vs := requirements.VersionStream
	u := vs.URL
	ref := vs.Ref
	log.Logger().Infof("verifying the CLI package using version stream URL: %s and git ref: %s\n", u, vs.Ref)

	resolver, err := o.CreateVersionResolver(u, ref)
	if err != nil {
		return errors.Wrapf(err, "failed to create version resolver")
	}

	err = resolver.VerifyPackages(verifyMap)
	if err != nil {
		return err
	}

	log.Logger().Infof("the CLI packages seem to be setup correctly\n")
	table.Render()

	return nil
}
