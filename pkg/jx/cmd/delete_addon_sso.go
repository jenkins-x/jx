package cmd

import (
	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const optionReleases = "releases"

var (
	deleteAddonSSOLong = templates.LongDesc(`
		Deletes the SSO addon
`)

	deleteAddonSSOExample = templates.Examples(`
		# Deletes the SSO addon
		jx delete addon sso
	`)

	defaultSsoReleaseNames = []string{kube.DefaultSsoOperatorReleaseName, kube.DefaultSsoDexReleaseName}
)

// DeleteAddonSSOOptions the options for delete addon sso command
type DeleteAddonSSOOptions struct {
	DeleteAddonOptions

	ReleaseNames []string
}

// NewCmdDeleteAddonSSO defines the command
func NewCmdDeleteAddonSSO(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &DeleteAddonSSOOptions{
		DeleteAddonOptions: DeleteAddonOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "sso",
		Short:   "Deletes the Single Sing-On addon",
		Aliases: []string{"sso"},
		Long:    deleteAddonSSOLong,
		Example: deleteAddonSSOExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringArrayVarP(&options.ReleaseNames, optionReleases, "r", defaultSsoReleaseNames, "The relese names of sso charts")
	options.addFlags(cmd)
	return cmd
}

// Run implements the command
func (o *DeleteAddonSSOOptions) Run() error {
	if len(o.ReleaseNames) == 0 {
		return util.MissingOption(optionReleases)
	}

	for _, releaseName := range o.ReleaseNames {
		err := o.DeleteChart(releaseName, o.Purge)
		if err != nil {
			return errors.Wrapf(err, "deleteing the helm chart release '%s'", releaseName)
		}
	}

	log.Logger().Infof("%s was succesfully deleted.\n", util.ColorInfo("sso addon"))

	return nil
}
