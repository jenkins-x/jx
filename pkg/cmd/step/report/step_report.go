package report

import (
	"encoding/xml"
	"github.com/google/uuid"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/reportingtools"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	stepReportLong    = templates.LongDesc(`This step is used to generate an HTML report from *.junit.xml files created from running BDD tests.`)
	stepReportExample = templates.Examples(`
	# Collect every *.junit.xml file from --in-dir, merge them, and store them in --out-dir with a file name --output-name and provide an HTML report title
	jx step report --in-dir /randomdir --out-dir /outdir --merge --output-name resulting_report.html --suite-name This_is_the_report_title

	# Collect every *.junit.xml file without defining --in-dir and use the value of $REPORTS_DIR , merge them, and store them in --out-dir with a file name --output-name
	jx step report --out-dir /outdir --merge --output-name resulting_report.html

	# Select a single *.junit.xml file and create a report form it
	jx step report --in-dir /randomdir --out-dir /outdir --target-report test.junit.xml --output-name resulting_report.html
`)
)

// StepReportOptions contains the command line flags and other helper objects
type StepReportOptions struct {
	opts.StepOptions
	reportingtools.XUnitClient
	MergeReports     bool
	ReportsDir       string
	TargetReport     string
	OutputReportName string
	OutputDir        string
	SuiteName        string
	DeleteReportFn   func(reportName string) error
}

// TestSuites is the representation of the root of a *.junit.xml xml file
type TestSuites struct {
	XMLName    xml.Name    `xml:"testsuites"`
	Text       string      `xml:",chardata"`
	TestSuites []TestSuite `xml:"testsuite"`
}

// TestSuite is the representation of a <testsuite> of a *.junit.xml xml file
type TestSuite struct {
	Text     string     `xml:",chardata"`
	Name     string     `xml:"name,attr"`
	Tests    string     `xml:"tests,attr"`
	Failures string     `xml:"failures,attr"`
	Errors   string     `xml:"errors,attr"`
	Time     string     `xml:"time,attr"`
	TestCase []TestCase `xml:"testcase"`
}

// TestCase is the representation of an individual test case within a TestSuite in a *.junit.xml xml file
type TestCase struct {
	Text      string   `xml:",chardata"`
	Name      string   `xml:"name,attr"`
	Classname string   `xml:"classname,attr"`
	Time      string   `xml:"time,attr"`
	Failure   *Failure `xml:"failure,omitempty"`
	SystemOut string   `xml:"system-out"`
}

// Failure is the representation of a Failure that can be present in a TestCase within a TestSuite in a *.junit.xml xml file
type Failure struct {
	Text string `xml:",chardata"`
	Type string `xml:"type,attr"`
}

// NewCmdStepReport Creates a new Command object
func NewCmdStepReport(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &StepReportOptions{
		StepOptions: opts.StepOptions{
			CommonOptions: commonOpts,
		},
	}

	cmd := &cobra.Command{
		Use:     "report",
		Short:   "Creates a HTML report from junit files",
		Long:    stepReportLong,
		Example: stepReportExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.ReportsDir, "in-dir", "f", "", "The directory to get the reports from")
	cmd.Flags().StringVarP(&options.OutputDir, "out-dir", "o", "", "The directory to store the resulting reports in")
	cmd.Flags().StringVarP(&options.TargetReport, "target-report", "t", "", "The name of a single report file to parse")
	cmd.Flags().StringVarP(&options.OutputReportName, "output-name", "n", "", "The result of parsing the report(s) in HTML format")
	cmd.Flags().StringVarP(&options.SuiteName, "suite-name", "s", "", "The name of the tests suite to be shown in the HTML report")
	cmd.Flags().BoolVarP(&options.MergeReports, "merge", "m", false, "Whether or not to merge the report files in the \"in-folder\" to parse them and show it as a single test run")

	return cmd
}

func (o *StepReportOptions) Run() error {

	if o.XUnitClient == nil {
		o.XUnitClient = reportingtools.XUnitViewer{}
	}

	if o.DeleteReportFn == nil {
		o.DeleteReportFn = util.DeleteFile
	}

	//We want to finish gracefully, otherwise the pipeline would fail
	err := o.XUnitClient.EnsureXUnitViewer(o.CommonOptions)
	if err != nil {
		return logErrorAndExitGracefully("there was a problem ensuring the presence of xunit-viewer", err)
	}

	// check $REPORTS_DIR is set, overridden by "in-folder"
	if o.ReportsDir == "" {
		o.ReportsDir = os.Getenv("REPORTS_DIR")
	}

	matchingReportFiles, err := o.obtainingMatchingReportFiles()
	if err != nil {
		return logErrorAndExitGracefully("there was a problem obtaining the matching report files", err)
	}

	targetFileName, err := generateTargetParsableReportName()
	defer o.DeleteReportFn(targetFileName)
	if err != nil {
		return logErrorAndExitGracefully("there was a problem generating a parsable temporary file", err)
	}

	// if "merge" is true, merge every xml file into one, called <targetFileName>, if there's only one file, rename to <targetFileName>
	if o.MergeReports {
		err = o.mergeJUnitReportFiles(matchingReportFiles, targetFileName)
		if err != nil {
			return logErrorAndExitGracefully("there was a problem merging the junit reports: %+v", err)
		}
	} else {
		err = o.prepareSingleFileForParsing(targetFileName)
		if err != nil {
			return logErrorAndExitGracefully("there was a problem preparing a single junit report: %+v", err)
		}
	}

	if o.OutputDir != "" {
		o.OutputReportName = filepath.Join(o.OutputDir, o.OutputReportName)
	}

	// Generate report with xunit-viewer from <targetFileName>
	err = o.XUnitClient.CreateHTMLReport(o.OutputReportName, o.SuiteName, targetFileName)
	if err != nil {
		return logErrorAndExitGracefully("error creating the HTML report", err)
	}
	return nil
}

func generateTargetParsableReportName() (string, error) {
	fileName := uuid.New().String() + ".xml"
	xunitReportsPath := filepath.Join(os.TempDir(), "xunit-reports")
	err := os.MkdirAll(xunitReportsPath, os.ModePerm)
	if err != nil {
		return "", errors.Wrap(err, "there was a problem creating the xunit-reports dir in the temp dir")
	}
	fileName = filepath.Join(xunitReportsPath, fileName)
	return fileName, nil
}

func (o *StepReportOptions) obtainingMatchingReportFiles() ([]string, error) {
	matchingReportFiles, err := filepath.Glob(filepath.Join(o.ReportsDir, "*.junit.xml"))
	if err != nil {
		return nil, errors.Wrapf(err, "There was an error reading the report files in directory %s", o.ReportsDir)
	}

	if matchingReportFiles == nil {
		return nil, errors.Errorf("no report files to parse in %s, skipping", o.ReportsDir)
	}
	return matchingReportFiles, nil
}

func (o *StepReportOptions) prepareSingleFileForParsing(resultFileName string) error {
	if o.TargetReport == "" {
		return errors.New("the TargetReport name is empty, parsing will ber skipped")
	}
	err := util.CopyFile(filepath.Join(o.ReportsDir, o.TargetReport), resultFileName)
	if err != nil {
		return errors.Wrap(err, "there was a problem copying the report file to temp directory, skipping")
	}
	return nil
}

func (o *StepReportOptions) mergeJUnitReportFiles(jUnitReportFiles []string, resultFileName string) error {
	log.Logger().Infof(util.ColorInfo("Performing merge of *.junit.xml files in %s"), o.ReportsDir)

	testSuites := TestSuites{}
	for _, v := range jUnitReportFiles {
		bytes, err := ioutil.ReadFile(v)
		if err != nil {
			return err
		}
		var testSuite TestSuite
		err = xml.Unmarshal(bytes, &testSuite)
		if err != nil {
			return err
		}
		testSuites.TestSuites = append(testSuites.TestSuites, testSuite)
	}

	suitesBytes, err := xml.Marshal(testSuites)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(resultFileName, suitesBytes, 0600)
	if err != nil {
		return err
	}

	return nil
}

func logErrorAndExitGracefully(message string, err error) error {
	log.Logger().Errorf("%s: %+v", message, err.Error())
	return nil
}
