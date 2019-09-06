package report

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	stepReportVersionLong    = templates.LongDesc(`Creates a report of a set of package versions. This command is typically used inside images to determine what tools are inside.`)
	stepReportVersionExample = templates.Examples(`
`)
)

var (
	defaultPackageVersions = []string{"jx", "kubectl", "helm", "helm3", "git", "skaffold"}
	numberRegex            = regexp.MustCompile("[0-9]")
)

// VersionReport the report
type VersionReport struct {
	Versions []Pair `json:"versions,omitempty"`
	Failures []Pair `json:"failures,omitempty"`
}

// Pair represents a name and value
type Pair struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// StepReportVersionOptions contains the command line flags and other helper objects
type StepReportVersionOptions struct {
	StepReportOptions
	FileName string
	Packages []string

	Report VersionReport
}

// NewCmdStepReportVersion Creates a new Command object
func NewCmdStepReportVersion(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepReportVersionOptions{
		StepReportOptions: StepReportOptions{
			StepOptions: step.StepOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Creates a report of a set of package versions",
		Aliases: []string{"versions"},
		Long:    stepReportVersionLong,
		Example: stepReportVersionExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	options.StepReportOptions.AddReportFlags(cmd)

	cmd.Flags().StringVarP(&options.FileName, "name", "n", "", "The name of the file to generate")
	cmd.Flags().StringArrayVarP(&options.Packages, "package", "p", defaultPackageVersions, "The name of the packages to version")
	return cmd
}

// Run generates the report
func (o *StepReportVersionOptions) Run() error {
	report := &o.Report

	err := o.generateReport()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(report)
	if err != nil {
		return errors.Wrap(err, "failed to marshal VersionReport to YAML")

	}
	if o.FileName == "" {
		log.Logger().Infof(string(data))
		return nil
	}
	if o.OutputDir == "" {
		o.OutputDir = "."
	}
	err = os.MkdirAll(o.OutputDir, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrap(err, "failed to create directories")
	}
	yamlFile := filepath.Join(o.OutputDir, o.FileName)
	err = ioutil.WriteFile(yamlFile, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save report file %s", yamlFile)
	}
	log.Logger().Infof("generated version report at %s", util.ColorInfo(yamlFile))
	return nil
}

func (o *StepReportVersionOptions) generateReport() error {
	sort.Strings(o.Packages)
	for _, p := range o.Packages {
		version, err := o.getPackageVersion(p)
		if err != nil {
			o.Report.Failures = append(o.Report.Failures, Pair{p, err.Error()})
		} else {
			o.Report.Versions = append(o.Report.Versions, Pair{p, version})
		}
	}
	return nil
}

func (o *StepReportVersionOptions) getPackageVersion(name string) (string, error) {
	args := []string{"version"}
	switch name {
	case "jx":
		args = []string{"--version"}
	case "kubectl":
		args = append(args, "--client", "--short")
	case "helm":
		args = append(args, "--client", "--short")
	case "helm3":
		args = append(args, "--short")
	}
	version, err := o.GetCommandOutput("", name, args...)

	// lets trim non-numeric prefixes such as for `git version` returning `git version 1.2.3`
	idxs := numberRegex.FindStringIndex(version)
	if len(idxs) > 0 {
		return version[idxs[0]:], err
	}
	return version, err
}
