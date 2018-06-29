package helm

type Version int

const (
	V2 Version = 2
	V3         = 3
)

type Helmer interface {
	InstallBinary(version Version) error
	Init(clientOnly bool, serviceAccount string, tillerNamespace string, upgrade bool)
	AddRepo(repo string, URL string) error
	RemoveRepo(repo string) error
	ListRepos() ([]string, error)
	UpdateRepo() error
	RemoveRequirementsLock() error
	BuildDependency() error
	InstallChart(chart string, releaseName string, ns string, values []string) error
	UpgradeChart(chart string, releaseName string, ns string, timeout int, version string,
		force bool, wait bool, values []string) error
	DeleteRelease(releaseName string, purge bool) error
	SearchChartVersions(chart string) ([]string, error)
	FindChart() (string, error)
	StatusRelease(releaseName string) error
	Lint() error
	Version() error
}
