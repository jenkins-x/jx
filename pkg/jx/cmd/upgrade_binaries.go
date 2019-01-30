package cmd

import (
	"io"
	"io/ioutil"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

var (
	upgradeBinariesLong = templates.LongDesc(`
		Upgrades the Jenkins X command line binaries (like helm or eksctl) if there is a newer release
`)

	upgradeBInariesExample = templates.Examples(`
		# Upgrades the Jenkins X binaries (like helm or eksctl) 
		jx upgrade binaries
	`)
)

type UpgradeBinariesOptions struct {
	CreateOptions
}

func NewCmdUpgradeBinaries(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &UpgradeBinariesOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "binaries",
		Short:   "Upgrades the command line binaries (like helm or eksctl) - if there are new versions available",
		Long:    upgradeBinariesLong,
		Example: upgradeBInariesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Verbose, "verbose", "", false, "Enable verbose logging")
	return cmd
}

func (o *UpgradeBinariesOptions) Run() error {
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	binaries, err := ioutil.ReadDir(binDir)
	if err != nil {
		return err
	}

	for _, binary := range binaries {
		if binary.Name() == "eksctl" {
			err = o.installEksCtl(true)
			if err != nil {
				return err
			}
		} else if binary.Name() == "heptio-authenticator-aws" {
			err = o.installHeptioAuthenticatorAws(true)
			if err != nil {
				return err
			}
		} else if binary.Name() == "kubectl" {
			err = o.installKubectl(true)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
