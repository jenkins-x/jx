// +build unit

package step

import (
	"testing"

	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"

	"github.com/jenkins-x/jx/v2/pkg/cmd/opts/step"

	"github.com/google/uuid"
	cmd_test "github.com/jenkins-x/jx/v2/pkg/cmd/clients/mocks"
	helm_test "github.com/jenkins-x/jx/v2/pkg/helm/mocks"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	"k8s.io/helm/pkg/proto/hapi/chart"

	"github.com/stretchr/testify/assert"
)

func TestStepWaitForChart(t *testing.T) {
	nameUUID, err := uuid.NewUUID()
	assert.NoError(t, err)

	name := nameUUID.String()
	version := "0.0.1"

	helmer := helm_test.NewMockHelmer()
	helm_test.StubFetchChart(name, "", kube.DefaultChartMuseumURL, &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: version,
		},
	}, helmer)

	assert.NoError(t, err)

	mockFactory := cmd_test.NewMockFactory()

	commonOpts := opts.NewCommonOptionsWithFactory(mockFactory)
	commonOpts.SetHelm(helmer)
	options := &WaitForChartOptions{
		ChartName:    name,
		ChartVersion: version,
		ChartRepo:    kube.DefaultChartMuseumURL,
		StepOptions: &step.StepOptions{
			CommonOptions: &commonOpts,
		},
	}

	err = options.Run()
	assert.NoError(t, err)
}
