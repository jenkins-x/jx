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

var (
	defaultChartReportName = "chart-images"
)

// ChartReport the report
type ChartReport struct {
	Charts []*ChartData `json:"charts,omitempty"`
}

// ChartData the image information
type ChartData struct {
	Prefix  string   `json:"prefix,omitempty"`
	Name    string   `json:"name,omitempty"`
	RepoURL string   `json:"url,omitempty"`
	Version string   `json:"version,omitempty"`
	Images  []string `json:"images,omitempty"`
}

// StepReportChartOptions contains the command line flags and other helper objects
type StepReportChartOptions struct {
	StepReportOptions
	ChartsDir       string
	ReportName      string
	FailOnDuplicate bool

	Report ChartReport
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
	cmd.Flags().StringVarP(&options.ReportName, "name", "n", defaultChartReportName, "The name of the files to generate")
	cmd.Flags().BoolVarP(&options.FailOnDuplicate, "fail-on-duplicate", "f", false, "If true lets fail the step if we have any duplicate")
	return cmd
}

func (o *StepReportChartOptions) Run() error {

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

	report := &o.Report

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

		chartInfo := &ChartData{
			RepoURL: chartRepo,
			Prefix:  chartPrefix,
			Name:    chartName,
			Version: version,
		}
		report.Charts = append(report.Charts, chartInfo)
		err = o.processChart(chartInfo)
		if err != nil {
			return err
		}
		sort.Strings(chartInfo.Images)

		cl := util.ColorInfo
		log.Logger().Infof("chart %s/%s has images: %s", cl(chartInfo.Prefix), cl(chartInfo.Name), cl(strings.Join(chartInfo.Images, ", ")))
		return nil
	})

	SortChartData(report.Charts)
	report.CalculateDuplicates()

	if o.OutputDir == "" {
		o.OutputDir = "."
	}
	if o.ReportName == "" {
		o.ReportName = defaultChartReportName
	}
	err = os.MkdirAll(o.OutputDir, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrap(err, "failed to create directories")
	}

	data, err := yaml.Marshal(report)
	if err != nil {
		return errors.Wrap(err, "failed to marshal ChartReport to YAML")

	}
	yamlFile := filepath.Join(o.OutputDir, o.ReportName+".yml")
	err = ioutil.WriteFile(yamlFile, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save report file %s", yamlFile)
	}
	log.Logger().Infof("generated chart images report at %s", util.ColorInfo(yamlFile))

	markdownFile := filepath.Join(o.OutputDir, o.ReportName+".yml")
	return report.GenerateMarkdown(markdownFile)
}

func (o *StepReportChartOptions) processChart(info *ChartData) error {
	tmpDir, err := ioutil.TempDir("", "helm-"+info.Name+"-")
	if err != nil {
		return errors.Wrapf(err, "failed to create tempo dir")
	}

	chartDir := filepath.Join(tmpDir, "chart")
	fullChart := fmt.Sprintf("%s/%s", info.Prefix, info.Name)
	version := info.Version

	// lets fetch the chart
	err = o.RunCommandVerbose("helm", "fetch", "-d", chartDir, "--version", version, "--untar", "--repo", info.RepoURL, info.Name)
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

	output, err := o.GetCommandOutput(chartDir, "helm", "template", info.Name)
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

func (o *StepReportChartOptions) processTemplate(info *ChartData, text string) error {
	data := map[string]interface{}{}

	err := yaml.Unmarshal([]byte(text), &data)
	if err != nil {
		return errors.Wrapf(err, "failed to parse template for chart %s/%s for template: %s", info.Prefix, info.Name, text)
	}

	if util.GetMapValueAsStringViaPath(data, "kind") == "Deployment" {
		d := v1beta1.Deployment{}
		err := yaml.Unmarshal([]byte(text), &d)
		if err != nil {
			return errors.Wrapf(err, "failed to parse Deployment template for chart %s/%s for template: %s", info.Prefix, info.Name, text)
		}

		containers := d.Spec.Template.Spec.Containers
		if len(containers) == 0 {
			log.Logger().Warnf("Deployment template %s for chart %s/%s has no containers", d.Name, info.Prefix, info.Name)
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

// CalculateDuplicates figures out if there are any duplicate image verisons
func (r *ChartReport) CalculateDuplicates() {
}

// GenerateMarkdown generates the markdown version of the report
func (r *ChartReport) GenerateMarkdown(s string) error {
	return fmt.Errorf("TODO")
}

func (i *ChartData) addImage(image string) {
	if util.StringArrayIndex(i.Images, image) < 0 {
		i.Images = append(i.Images, image)
		sort.Strings(i.Images)
	}
}

type chartDataOrder []*ChartData

func (a chartDataOrder) Len() int      { return len(a) }
func (a chartDataOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a chartDataOrder) Less(i, j int) bool {
	r1 := a[i]
	r2 := a[j]

	n1 := r1.Prefix
	n2 := r2.Prefix
	if n1 != n2 {
		return n1 < n2
	}
	v1 := r1.Name
	v2 := r2.Name
	return v1 < v2
}

// SortChartData sorts the charts in order
func SortChartData(items []*ChartData) {
	sort.Sort(chartDataOrder(items))
}
