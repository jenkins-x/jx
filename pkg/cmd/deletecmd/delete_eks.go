package deletecmd

import (
	"errors"

	"github.com/jenkins-x/jx/pkg/cloud/amazon/session"

	"github.com/jenkins-x/jx/pkg/cmd/get"
	"github.com/jenkins-x/jx/pkg/packages"

	"os"
	"os/exec"

	"github.com/jenkins-x/jx/pkg/cmd/helper"

	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
)

type deleteEksOptions struct {
	get.GetOptions
	Profile string
	Region  string
}

var (
	deleteEksLong = templates.LongDesc(`
		Deletes EKS cluster resource. Under the hood this command delegated removal operation to eksctl.
`)

	deleteEksExample = templates.Examples(`
		# Delete EKS cluster
		jx delete eks
	`)
)

func newCmdDeleteEks(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &deleteEksOptions{
		GetOptions: get.GetOptions{
			CommonOptions: commonOpts,
		},
	}
	cmd := &cobra.Command{
		Use:     "eks",
		Short:   "Deletes EKS cluster.",
		Long:    deleteEksLong,
		Example: deleteEksExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Profile, "profile", "", "", "AWS profile to use.")
	cmd.Flags().StringVarP(&options.Region, "region", "", "", "AWS region to use. Default: "+session.DefaultRegion)

	options.AddGetFlags(cmd)
	return cmd
}

func (o *deleteEksOptions) Run() error {
	if len(o.Args) == 0 {
		return errors.New("cluster name expected")
	}
	cluster := o.Args[0]

	var deps []string
	d := packages.BinaryShouldBeInstalled("eksctl")
	if d != "" {
		deps = append(deps, d)
	}
	d = packages.BinaryShouldBeInstalled("aws-iam-authenticator")
	if d != "" {
		deps = append(deps, d)
	}
	err := o.InstallMissingDependencies(deps)
	if err != nil {
		log.Logger().Errorf("%v\nPlease fix the error or install manually then try again", err)
		os.Exit(-1)
	}

	region, err := session.ResolveRegion(o.Profile, o.Region)
	if err != nil {
		return err
	}
	cmd := exec.Command("eksctl", "delete", "cluster", cluster, "--region", region)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}
	log.Logger().Infof(string(output))
	return nil
}
