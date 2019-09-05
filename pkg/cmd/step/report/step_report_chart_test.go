package report

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/stretchr/testify/assert"
)

func TestStepReportChart(t *testing.T) {
	dirName, err := ioutil.TempDir("", uuid.New().String())
	defer os.Remove(dirName)
	assert.NoError(t, err, "there shouldn't be any problem creating a temp dir")

	reportName := uuid.New().String() + ".html"
	o := StepReportChartOptions{
		ChartsDir: filepath.Join("test_data", "step_report_chart", "jenkins-x-versions/charts"),
		StepReportOptions: StepReportOptions{
			OutputDir:        dirName,
			OutputReportName: reportName,
		},
	}
	o.CommonOptions = &opts.CommonOptions{}
	o.Out = os.Stdout
	o.Err = os.Stderr

	err = o.Run()
	assert.NoError(t, err)
}
