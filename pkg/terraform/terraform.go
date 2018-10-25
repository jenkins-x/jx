package terraform

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"io"
)

func Init(terraformDir string, serviceAccountPath string) error {
	fmt.Println("Initialising Terraform")

	if _, err := os.Stat(".terraform"); !os.IsNotExist(err) {
		fmt.Println("Discovered local .terraform directory, removing...")
		os.RemoveAll(".terraform")
	}

	os.Setenv("GOOGLE_CREDENTIALS", serviceAccountPath)
	cmd := util.Command{
		Name: "terraform",
		Args: []string{"init", terraformDir},
	}
	_, err := cmd.RunWithoutRetry()
	if err != nil {
		return err
	}
	return nil
}

func Plan(terraformDir string, terraformVars string, serviceAccountPath string) (string, error) {
	fmt.Println("Showing Terraform Plan")
	cmd := util.Command{
		Name: "terraform",
		Args: []string{"plan",
			fmt.Sprintf("-var-file=%s", terraformVars),
			"-var",
			fmt.Sprintf("credentials=%s", serviceAccountPath),
			terraformDir},
	}
	out, err := cmd.RunWithoutRetry()
	if err != nil {
		return out, err
	}
	return out, nil
}

func Apply(terraformDir string, terraformVars string, serviceAccountPath string, stdout io.Writer, stderr io.Writer) error {
	fmt.Println("Applying Terraform")
	cmd := util.Command{
		Name: "terraform",
		Args: []string{"apply", "-auto-approve",
			fmt.Sprintf("-var-file=%s", terraformVars),
			"-var",
			fmt.Sprintf("credentials=%s", serviceAccountPath),
			terraformDir},
		Out: stdout,
		Err: stderr,
	}
	_, err := cmd.RunWithoutRetry()
	if err != nil {
		return err
	}
	return nil
}

func WriteKeyValueToFileIfNotExists(path string, key string, value string) error {
	// file exists
	if _, err := os.Stat(path); err == nil {
		buffer, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		contents := string(buffer)

		if strings.Contains(contents, key) {
			return nil
		}
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	line := fmt.Sprintf("%s = \"%s\"\n", key, value)

	_, err = file.WriteString(line)
	if err != nil {
		return err
	}

	return nil
}

func ReadValueFromFile(path string, key string) (string, error) {
	if _, err := os.Stat(path); err == nil {
		buffer, err := ioutil.ReadFile(path)
		if err != nil {
			return "", err
		}
		contents := string(buffer)
		lines := strings.Split(contents, "\n")
		for _, line := range lines {
			if strings.Contains(line, key) {
				tokens := strings.Split(line, "=")
				trimmedValue := strings.Trim(strings.TrimSpace(tokens[1]), "\"")
				return trimmedValue, nil
			}
		}

	}
	return "", nil
}
