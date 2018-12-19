package cmd

import (
	"errors"
	"fmt"
	"github.com/jenkins-x/jx/pkg/cloud/amazon"
	"io"
	"os"
	"os/exec"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

type DeleteEksOptions struct {
	GetOptions
	Profile string
	Region  string
}

var (
	DeleteEksLong = templates.LongDesc(`
		Deletes EKS cluster resource
`)

	DeleteEksExample = templates.Examples(`
		# Delete EKS cluster
		jx delete eks
	`)
)

func NewCmdDeleteEks(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &DeleteEksOptions{
		GetOptions: GetOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "eks",
		Short:   "Deletes EKS cluster.",
		Long:    DeleteEksLong,
		Example: DeleteEksExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&options.Profile, "profile", "", "", "AWS profile to use.")
	cmd.Flags().StringVarP(&options.Region, "region", "", "", "AWS region to use. Default: "+amazon.DefaultRegion)

	options.addGetFlags(cmd)
	return cmd
}

func (o *DeleteEksOptions) Run() error {
	if len(o.Args) == 0 {
		return errors.New("Cluster name expected.")
	}
	cluster := o.Args[0]

	var deps []string
	d := binaryShouldBeInstalled("eksctl")
	if d != "" {
		deps = append(deps, d)
	}
	d = binaryShouldBeInstalled("heptio-authenticator-aws")
	if d != "" {
		deps = append(deps, d)
	}
	err := o.installMissingDependencies(deps)
	if err != nil {
		log.Errorf("%v\nPlease fix the error or install manually then try again", err)
		os.Exit(-1)
	}

	region, err := amazon.ResolveRegion(o.Profile, o.Region)
	if err != nil {
		return err
	}
	cmd := exec.Command("eksctl", "delete", "cluster", cluster, "--region", region)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}
	fmt.Print(string(output))
	return nil
}
