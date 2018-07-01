package helm

type Version int

const (
	V2 Version = 2
	V3         = 3
)

type Helmer interface {
	SetCWD(dir string)
	Init(clientOnly bool, serviceAccount string, tillerNamespace string, upgrade bool) error
	AddRepo(repo string, URL string) error
	RemoveRepo(repo string) error
	ListRepos() (map[string]string, error)
	UpdateRepo() error
	IsRepoMissing(URL string) (bool, error)
	RemoveRequirementsLock() error
	BuildDependency() error
	InstallChart(chart string, releaseName string, ns string, values []string) error
	UpgradeChart(chart string, releaseName string, ns string, version *string, install bool,
		timeout *int, force bool, wait bool, values []string) error
	DeleteRelease(releaseName string, purge bool) error
	ListCharts() (string, error)
	SearchChartVersions(chart string) ([]string, error)
	FindChart() (string, error)
	StatusRelease(releaseName string) error
	Lint() (string, error)
	Version() (string, error)
}
