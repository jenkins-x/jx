package installer

import (
	"testing"

	"k8s.io/helm/pkg/chartutil"
)

func TestDefaultChart(t *testing.T) {
	_, err := chartutil.LoadFiles(DefaultChartFiles)
	if err != nil {
		t.Errorf("expected loading the default chart to not fail, got %v", err)
	}
}
