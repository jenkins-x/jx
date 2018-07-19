package helm

//
//import (
//	"strings"
//
//	"github.com/jenkins-x/jx/pkg/log"
//)
//
//type HelmFake struct {
//	helm *HelmCLI
//}
//
//func createRunner(output string) helmRunner {
//	return helmRunner{
//		run: func(dir string, name string, args ...string) error {
//			log.Infof("Dir: '%s', Executed: '%s'\n", dir, name+" "+strings.Join(args, " "))
//			return nil
//		},
//		runWithOutput: func(dir string, name string, args ...string) (string, error) {
//			log.Infof("Dir: '%s', Executed: '%s'\n", dir, name+" "+strings.Join(args, " "))
//			return output, nil
//		},
//	}
//}
//
//func NewHelmFake(binary string, version Version, cwd string, output string) *HelmFake {
//	fakeRunner := createRunner(output)
//	return &HelmFake{
//		helm: newHelmCLIWithRunner(binary, version, cwd, fakeRunner),
//	}
//}
//
//func (h *HelmFake) SetOutput(output string) {
//	h.helm.runner = createRunner(output)
//}
//
//func (h *HelmFake) SetCWD(dir string) {
//	h.helm.SetCWD(dir)
//}
//
//func (h *HelmFake) HelmBinary() string {
//	return h.helm.HelmBinary()
//}
//
//func (h *HelmFake) SetHelmBinary(binary string) {
//	h.helm.SetHelmBinary(binary)
//}
//
//func (h *HelmFake) Init(clientOnly bool, serviceAccount string, tillerNamespace string, upgrade bool) error {
//	return h.helm.Init(clientOnly, serviceAccount, tillerNamespace, upgrade)
//}
//
//func (h *HelmFake) AddRepo(repo string, URL string) error {
//	return h.helm.AddRepo(repo, URL)
//}
//
//func (h *HelmFake) RemoveRepo(repo string) error {
//	return h.helm.RemoveRepo(repo)
//}
//
//func (h *HelmFake) ListRepos() (map[string]string, error) {
//	return h.helm.ListRepos()
//}
//
//func (h *HelmFake) UpdateRepo() error {
//	return h.helm.UpdateRepo()
//}
//
//func (h *HelmFake) IsRepoMissing(URL string) (bool, error) {
//	return h.helm.IsRepoMissing(URL)
//}
//
//func (h *HelmFake) RemoveRequirementsLock() error {
//	return h.helm.RemoveRequirementsLock()
//}
//
//func (h *HelmFake) BuildDependency() error {
//	return h.helm.BuildDependency()
//}
//
//func (h *HelmFake) InstallChart(chart string, releaseName string, ns string, version *string,
//	timeout *int, values []string, valueFiles []string) error {
//	return h.helm.InstallChart(chart, releaseName, ns, version, timeout, values, valueFiles)
//}
//
//func (h *HelmFake) UpgradeChart(chart string, releaseName string, ns string, version *string,
//	install bool, timeout *int, force bool, wait bool, values []string, valueFiles []string) error {
//	return h.helm.UpgradeChart(chart, releaseName, ns, version, install, timeout, force, wait, values, valueFiles)
//}
//
//func (h *HelmFake) DeleteRelease(releaseName string, purge bool) error {
//	return h.helm.DeleteRelease(releaseName, purge)
//}
//
//func (h *HelmFake) ListCharts() (string, error) {
//	return h.helm.ListCharts()
//}
//
//func (h *HelmFake) SearchChartVersions(chart string) ([]string, error) {
//	return h.helm.SearchChartVersions(chart)
//}
//
//func (h *HelmFake) FindChart() (string, error) {
//	return h.helm.FindChart()
//}
//
//func (h *HelmFake) PackageChart() error {
//	return h.helm.PackageChart()
//}
//
//func (h *HelmFake) StatusRelease(releaseName string) error {
//	return h.helm.StatusRelease(releaseName)
//}
//
//func (h *HelmFake) StatusReleases() (map[string]string, error) {
//	return h.helm.StatusReleases()
//}
//
//func (h *HelmFake) Lint() (string, error) {
//	return h.helm.Lint()
//}
//
//func (h *HelmFake) Version(tls bool) (string, error) {
//	return h.helm.Version(tls)
//}
