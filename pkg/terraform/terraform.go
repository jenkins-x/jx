package terraform

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/util"
)

func Init(terraformDir string) error {
	err := util.RunCommand("", "terraform", "init", terraformDir)
	if err != nil {
		return err
	}
	return nil
}

func Plan(terraformDir string, terraformVars string, credentials string) error {
	err := util.RunCommand("", "terraform", "plan",
		fmt.Sprintf("-var-file=%s", terraformVars),
		"-var",
		fmt.Sprintf("credentials=%s", credentials),
		terraformDir)
	if err != nil {
		return err
	}
	return nil
}

func Apply(terraformDir string, terraformVars string, credentials string) error {
	err := util.RunCommand("", "terraform", "apply", "-auto-approve",
		fmt.Sprintf("-var-file=%s", terraformVars),
		"-var",
		fmt.Sprintf("credentials=%s", credentials),
		terraformDir)
	if err != nil {
		return err
	}
	return nil
}
