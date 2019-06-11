package compliance

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/helper"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/client/results"
	"github.com/heptio/sonobuoy/pkg/plugin/aggregation"
	"github.com/jenkins-x/jx/pkg/jx/cmd/opts"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/onsi/ginkgo/reporters"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	complianceResultsLong = templates.LongDesc(`
		Shows the results of the compliance tests
	`)

	complianceResultsExample = templates.Examples(`
		# Show the compliance results
		jx compliance results
	`)
)

// ComplianceResultsOptions options for "compliance results" command
type ComplianceResultsOptions struct {
	*opts.CommonOptions
}

// NewCmdComplianceResults creates a command object for the "compliance results" action, which
// shows the results of E2E compliance tests
func NewCmdComplianceResults(commonOpts *opts.CommonOptions) *cobra.Command {
	options := &ComplianceResultsOptions{
		CommonOptions: commonOpts,
	}

	cmd := &cobra.Command{
		Use:     "results",
		Short:   "Shows the results of compliance tests",
		Long:    complianceResultsLong,
		Example: complianceResultsExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	return cmd
}

// Run implements the "compliance results" command
func (o *ComplianceResultsOptions) Run() error {
	cc, err := o.ComplianceClient()
	if err != nil {
		return errors.Wrap(err, "could not create the compliance client")
	}

	status, err := cc.GetStatus(complianceNamespace)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve the compliance status")
	}

	if status.Status != aggregation.CompleteStatus && status.Status != aggregation.FailedStatus {
		log.Logger().Info("Compliance results not ready. Run `jx compliance status` for status.")
		return nil
	}

	cfg := &client.RetrieveConfig{
		Namespace: complianceNamespace,
	}

	reader, errch := cc.RetrieveResults(cfg)
	eg := &errgroup.Group{}
	eg.Go(func() error { return <-errch })
	eg.Go(func() error {
		resultsReader, ec := untarResults(reader)
		gzr, err := gzip.NewReader(resultsReader)
		if err != nil {
			return errors.Wrap(err, "could not create a gzip reader for compliance results ")
		}

		testResults, err := cc.GetTests(gzr, "all")
		if err != nil {
			return errors.Wrap(err, "could not get the results of the compliance tests from the archive")
		}
		testResults = filterTests(
			func(tc reporters.JUnitTestCase) bool {
				return !results.Skipped(tc)
			}, testResults)
		sort.Sort(StatusSortedTestCases(testResults))
		o.printResults(testResults)

		err = <-ec
		if err != nil {
			return errors.Wrap(err, "could not extract the compliance results from archive")
		}
		return nil
	})

	err = eg.Wait()
	if err != nil {
		log.Logger().Infof("No compliance results found. Use %s command to start the compliance tests.", util.ColorInfo("jx compliance run"))
		log.Logger().Infof("You can watch the logs with %s command.", util.ColorInfo("jx compliance logs -f"))
	}
	return nil
}

// Exit the main goroutine with status
func (o *ComplianceResultsOptions) Exit(status int) {
	os.Exit(status)
}

// StatusSortedTestCases implements Sort by status of a list of test case
type StatusSortedTestCases []reporters.JUnitTestCase

var statuses = map[string]int{
	"FAILED":  0,
	"PASSED":  1,
	"SKIPPED": 2,
	"UNKNOWN": 3,
}

func (s StatusSortedTestCases) Len() int { return len(s) }
func (s StatusSortedTestCases) Less(i, j int) bool {
	si := statuses[status(s[i])]
	sj := statuses[status(s[j])]
	return si < sj
}
func (s StatusSortedTestCases) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (o *ComplianceResultsOptions) printResults(junitResults []reporters.JUnitTestCase) {
	table := o.CreateTable()
	table.SetColumnAlign(1, util.ALIGN_LEFT)
	table.SetColumnAlign(2, util.ALIGN_LEFT)
	table.AddRow("STATUS", "TEST", "TEST-CLASS")
	for _, t := range junitResults {
		table.AddRow(status(t), t.Name, t.ClassName)
	}
	table.Render()
}

func status(junitResult reporters.JUnitTestCase) string {
	if results.Skipped(junitResult) {
		return "SKIPPED"
	} else if results.Failed(junitResult) {
		return "FAILED"
	} else if results.Passed(junitResult) {
		return "PASSED"
	} else {
		return "UNKNOWN"
	}
}

func untarResults(src io.Reader) (io.Reader, <-chan error) {
	ec := make(chan error, 1)
	tarReader := tar.NewReader(src)
	reader, writer := io.Pipe()
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err != io.EOF {
				ec <- err
				return reader, ec
			}
			break
		}
		if strings.HasSuffix(header.Name, ".tar.gz") {
			go func(writer *io.PipeWriter, ec chan error) {
				defer writer.Close()
				defer close(ec)
				_, err := io.Copy(writer, tarReader)
				if err != nil {
					ec <- err
				}
				tarReader.Next()
			}(writer, ec)
			break
		}
	}
	return reader, ec
}

func filterTests(predicate func(testCase reporters.JUnitTestCase) bool, testCases []reporters.JUnitTestCase) []reporters.JUnitTestCase {
	out := make([]reporters.JUnitTestCase, 0)
	for _, tc := range testCases {
		if predicate(tc) {
			out = append(out, tc)
		}
	}
	return out
}
