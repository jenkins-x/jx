package quickstarts_test

import (
	"testing"

	"github.com/jenkins-x/jx/pkg/quickstarts"
	"github.com/stretchr/testify/assert"
)

func TestQuickstartModelFilterText(t *testing.T) {
	t.Parallel()

	quickstart1 := &quickstarts.Quickstart{
		ID:   "jenkins-x-quickstarts/node-http",
		Name: "node-http",
	}
	quickstart2 := &quickstarts.Quickstart{
		ID:   "jenkins-x-quickstarts/node-http-watch-pipeline-activity",
		Name: "node-http-watch-pipeline-activity",
	}
	quickstart3 := &quickstarts.Quickstart{
		ID:   "jenkins-x-quickstarts/ruby",
		Name: "ruby",
	}

	qstarts := make(map[string]*quickstarts.Quickstart)
	qstarts["node-http"] = quickstart1
	qstarts["node-http-watch-pipeline-activity"] = quickstart2
	qstarts["ruby"] = quickstart3

	quickstartModel := &quickstarts.QuickstartModel{
		Quickstarts: qstarts,
	}

	quickstartFilter := &quickstarts.QuickstartFilter{
		Text: "ruby",
	}

	results := quickstartModel.Filter(quickstartFilter)

	assert.Equal(t, 1, len(results))
	assert.Contains(t, results, quickstart3)
}

func TestQuickstartModelFilterTextMatchesMoreThanOne(t *testing.T) {
	t.Parallel()

	quickstart1 := &quickstarts.Quickstart{
		ID:   "jenkins-x-quickstarts/node-http",
		Name: "node-http",
	}
	quickstart2 := &quickstarts.Quickstart{
		ID:   "jenkins-x-quickstarts/node-http-watch-pipeline-activity",
		Name: "node-http-watch-pipeline-activity",
	}
	quickstart3 := &quickstarts.Quickstart{
		ID:   "jenkins-x-quickstarts/ruby",
		Name: "ruby",
	}

	qstarts := make(map[string]*quickstarts.Quickstart)
	qstarts["node-http"] = quickstart1
	qstarts["node-http-watch-pipeline-activity"] = quickstart2
	qstarts["ruby"] = quickstart3

	quickstartModel := &quickstarts.QuickstartModel{
		Quickstarts: qstarts,
	}

	quickstartFilter := &quickstarts.QuickstartFilter{
		Text: "node-htt",
	}

	results := quickstartModel.Filter(quickstartFilter)

	assert.Equal(t, 2, len(results))
	assert.Contains(t, results, quickstart1)
	assert.Contains(t, results, quickstart2)
}

func TestQuickstartModelFilterTextMatchesOneExactly(t *testing.T) {
	t.Parallel()

	quickstart1 := &quickstarts.Quickstart{
		ID:   "jenkins-x-quickstarts/node-http",
		Name: "node-http",
	}
	quickstart2 := &quickstarts.Quickstart{
		ID:   "jenkins-x-quickstarts/node-http-watch-pipeline-activity",
		Name: "node-http-watch-pipeline-activity",
	}
	quickstart3 := &quickstarts.Quickstart{
		ID:   "jenkins-x-quickstarts/ruby",
		Name: "ruby",
	}

	qstarts := make(map[string]*quickstarts.Quickstart)
	qstarts["node-http"] = quickstart1
	qstarts["node-http-watch-pipeline-activity"] = quickstart2
	qstarts["ruby"] = quickstart3

	quickstartModel := &quickstarts.QuickstartModel{
		Quickstarts: qstarts,
	}

	quickstartFilter := &quickstarts.QuickstartFilter{
		Text: "node-http",
	}

	results := quickstartModel.Filter(quickstartFilter)

	assert.Equal(t, 1, len(results))
	assert.Contains(t, results, quickstart1)
}
