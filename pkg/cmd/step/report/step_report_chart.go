package report

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/opts/step"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/versionstream"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/api/apps/v1beta1"
)

var (
	stepReportChartLong    = templates.LongDesc(`This step is used to generate an HTML report from *.junit.xml files created from running BDD tests.`)
	stepReportChartExample = templates.Examples(`
	# Collect every *.junit.xml file from --in-dir, merge them, and store them in --out-dir with a file name --output-name and provide an HTML report title
	jx step report --in-dir /randomdir --out-dir /outdir --merge --output-name resulting_report.html --suite-name This_is_the_report_title

	# Collect every *.junit.xml file without defining --in-dir and use the value of $REPORTS_DIR , merge them, and store them in --out-dir with a file name --output-name
	jx step report --out-dir /outdir --merge --output-name resulting_report.html

	# Select a single *.junit.xml file and create a report form it
	jx step report --in-dir /randomdir --out-dir /outdir --target-report test.junit.xml --output-name resulting_report.html
`)
)

// StepReportChartOptions contains the command line flags and other helper objects
type StepReportChartOptions struct {
	StepReportOptions
	ChartsDir string
}

// NewCmdStepReportChart Creates a new Command object
func NewCmdStepReportChart(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepReportChartOptions{
		StepReportOptions: StepReportOptions{
			StepOptions: step.StepOptions{
				CommonOptions: commonOpts,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "chart",
		Short:   "Creates a report of all the images used in the charts in a version stream",
		Long:    stepReportChartLong,
		Example: stepReportChartExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	options.StepReportOptions.AddReportFlags(cmd)

	cmd.Flags().StringVarP(&options.ChartsDir, "dir", "d", "", "The dir to look for charts. If not specified it defaults to the version stream")

	return cmd
}

func (o *StepReportChartOptions) Run() error {
	/*	if o.OutputDir != "" {
			o.OutputReportName = filepath.Join(o.OutputDir, o.OutputReportName)
		}
	*/

	if o.ChartsDir == "" {
		resolver, err := o.GetVersionResolver()
		if err != nil {
			return err
		}
		o.ChartsDir = filepath.Join(resolver.VersionsDir, "charts")
	}

	dir := o.ChartsDir
	exists, err := util.DirExists(dir)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("directory does not exist %s", dir)
	}

	repoPrefixes, err := versionstream.GetRepositoryPrefixes(filepath.Join(dir, ".."))
	if err != nil {
		return err
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		name := info.Name()
		chartsDir := filepath.Dir(path)
		chartName := strings.TrimSuffix(name, filepath.Ext(name))
		_, chartPrefix := filepath.Split(chartsDir)

		if info.IsDir() || name == "repositories.yml" || !strings.HasSuffix(name, ".yml") {
			return nil
		}

		v, err := versionstream.LoadStableVersionFile(path)
		if err != nil {
			return err
		}
		prefixInfos := repoPrefixes.URLsForPrefix(chartPrefix)
		version := v.Version
		if len(prefixInfos) == 0 {
			log.Logger().Warnf("chart %s/%s does not have a repository URL for prefix in registries.yml", chartPrefix, chartName)
			return nil
		}
		if version == "" {
			log.Logger().Warnf("chart %s/%s does not have a version", chartPrefix, chartName)
			return nil
		}
		chartRepo := prefixInfos[0]
		log.Logger().Infof("found chart %s/%s in repo %s with version %s", chartPrefix, chartName, chartRepo, version)

		chartInfo := &chartInfo{
			repoURL:     chartRepo,
			chartPrefix: chartPrefix,
			chartName:   chartName,
			version:     version,
		}
		err = o.processChart(chartInfo)
		if err != nil {
			return err
		}
		sort.Strings(chartInfo.images)

		cl := util.ColorInfo
		log.Logger().Infof("chart %s/%s has images: %s", cl(chartInfo.chartPrefix), cl(chartInfo.chartName), cl(strings.Join(chartInfo.images, ", ")))
		return nil
	})
	return err
}

func (o *StepReportChartOptions) processChart(info *chartInfo) error {
	tmpDir, err := ioutil.TempDir("", "helm-"+info.chartName+"-")
	if err != nil {
		return errors.Wrapf(err, "failed to create tempo dir")
	}

	chartDir := filepath.Join(tmpDir, "chart")
	fullChart := fmt.Sprintf("%s/%s", info.chartPrefix, info.chartName)
	version := info.version

	// lets fetch the chart
	err = o.RunCommandVerbose("helm", "fetch", "-d", chartDir, "--version", version, "--untar", "--repo", info.repoURL, info.chartName)
	if err != nil {
		return err
	}

	exists, err := util.DirExists(chartDir)
	if err != nil {
		return errors.Wrapf(err, "failed to check chart dir exists %s", chartDir)
	}
	if !exists {
		return fmt.Errorf("should have fetched the chart %s version %s", fullChart, version)
	}

	output, err := o.GetCommandOutput(chartDir, "helm", "template", info.chartName)
	if err != nil {
		return err
	}

	files := strings.Split(output, "---\n")
	for _, text := range files {
		if isEmptyTemplate(text) {
			continue
		}
		err = o.processTemplate(info, text)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *StepReportChartOptions) processTemplate(info *chartInfo, text string) error {
	data := map[string]interface{}{}

	err := yaml.Unmarshal([]byte(text), &data)
	if err != nil {
		return errors.Wrapf(err, "failed to parse template for chart %s/%s for template: %s", info.chartPrefix, info.chartName, text)
	}

	if util.GetMapValueAsStringViaPath(data, "kind") == "Deployment" {
		d := v1beta1.Deployment{}
		err := yaml.Unmarshal([]byte(text), &d)
		if err != nil {
			return errors.Wrapf(err, "failed to parse Deployment template for chart %s/%s for template: %s", info.chartPrefix, info.chartName, text)
		}

		containers := d.Spec.Template.Spec.Containers
		if len(containers) == 0 {
			log.Logger().Warnf("Deployment template %s for chart %s/%s has no containers", d.Name, info.chartPrefix, info.chartName)
			return nil
		}

		for _, c := range containers {
			if c.Image != "" {
				info.addImage(c.Image)
			}
		}
	}
	return nil
}

func isEmptyTemplate(text string) bool {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		return false
	}
	return true
}

type chartInfo struct {
	repoURL, chartPrefix, chartName, version string
	images                                   []string
}

func (i *chartInfo) addImage(image string) {
	if util.StringArrayIndex(i.images, image) < 0 {
		i.images = append(i.images, image)
	}
}
