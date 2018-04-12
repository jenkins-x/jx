package addon

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
)

// GetChartStatusMap return the map of chart name -> status
func GetChartStatusMap() (map[string]string, error) {
	statusMap := map[string]string{}
	output, err := util.GetCommandOutput("", "helm", "list")
	if err == nil {
		for _, line := range strings.Split(output, "\n") {
			fields := strings.Split(line, "\t")
			if len(fields) > 3 {
				statusMap[strings.TrimSpace(fields[0])] = fields[3]
			}
		}
	}
	return statusMap, err
}

func ProviderAccessTokenURL(kind string, url string) string {
	switch kind {
	default:
		return ""
	}
}
