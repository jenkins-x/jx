package helm

// Version defines the helm version
type Version int

const (
	V2 Version = 2
	V3         = 3
)

type ChartSummary struct {
	Name         string
	ChartVersion string
	AppVersion   string
	Description  string
}
