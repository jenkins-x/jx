package terraform

import (
	"fmt"
	"github.com/blang/semver"
	"github.com/pkg/errors"
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
		err = os.RemoveAll(".terraform")
		if err != nil {
			return errors.Wrap(err, "unable to remove local .terraform directory")
		}
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

// CheckVersion checks the installed version of terraform to sure it is greater than 0.11.0
func CheckVersion() error {
	fmt.Println("Applying Terraform")
	cmd := util.Command{
		Name: "terraform",
		Args: []string{"-version"},
	}
	output, err := cmd.RunWithoutRetry()
	if err != nil {
		return err
	}

	version, err := extractVersionFromTerraformOutput(output)

	fmt.Printf("Determined terraform version as %s\n", util.ColorInfo(version))

	if err != nil {
		return err
	}

	v, err := semver.Make(version)

	r, err := semver.ParseRange(">= 0.11.0")
	if !r(v) {
		return errors.New("terraform version appears to be too old, please install a newer version '>= 0.11.0'")
	}

	fmt.Printf("Terraform version appears to be valid\n")

	return nil
}

func extractVersionFromTerraformOutput(output string) (string, error) {

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Terraform") {
			versionTokens := strings.Split(line, " ")
			return strings.TrimPrefix(versionTokens[1], "v"), nil
		}
	}

	return "", errors.Errorf("unable to extract version from output '%s'", output)

}
