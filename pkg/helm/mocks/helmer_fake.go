package helm_test

import (
	"fmt"

	"github.com/jenkins-x/jx/pkg/helm"
)

// FakeHelmer a fake Helmer
type FakeHelmer struct {
}

// NewFakeHelmer creates a new fake Helmer
func NewFakeHelmer() helm.Helmer {
	return &FakeHelmer{}
}

// AddRepo add repo
func (FakeHelmer) AddRepo(repo, URL, username, password string) error {
	return nil
}

// BuildDependency build dependency
func (FakeHelmer) BuildDependency() error {
	return nil
}

// DecryptSecrets decrypt secrets
func (FakeHelmer) DecryptSecrets(location string) error {
	return nil
}

// DeleteRelease delete release
func (FakeHelmer) DeleteRelease(ns string, releaseName string, purge bool) error {
	return nil
}

// Env return env
func (FakeHelmer) Env() map[string]string {
	return map[string]string{}
}

// FetchChart fake
func (FakeHelmer) FetchChart(chart string, version *string, untar bool, untardir string, repo string,
	username string, password string) error {
	return nil
}

// FindChart fake
func (FakeHelmer) FindChart() (string, error) {
	return "", fmt.Errorf("not found")
}

// HelmBinary fake
func (FakeHelmer) HelmBinary() string {
	return "helm"
}

// Init fake
func (FakeHelmer) Init(clientOnly bool, serviceAccount string, tillerNamespace string, upgrade bool) error {
	return nil
}

// InstallChart fake
func (FakeHelmer) InstallChart(chart string, releaseName string, ns string, version *string, timeout *int,
	values []string, valueFiles []string, repo string, username string, password string) error {
	return nil
}

// IsRepoMissing fake
func (FakeHelmer) IsRepoMissing(URL string) (bool, error) {
	return false, nil
}

// Lint fake
func (FakeHelmer) Lint() (string, error) {
	return "", nil
}

// ListCharts fake
func (FakeHelmer) ListCharts() (string, error) {
	return "", nil
}

// ListRepos fake
func (FakeHelmer) ListRepos() (map[string]string, error) {
	return map[string]string{}, nil
}

// PackageChart fake
func (FakeHelmer) PackageChart() error {
	return nil
}

// RemoveRepo fake
func (FakeHelmer) RemoveRepo(repo string) error {
	return nil
}

// RemoveRequirementsLock fake
func (FakeHelmer) RemoveRequirementsLock() error {
	return nil
}

// SearchChartVersions fake
func (FakeHelmer) SearchChartVersions(chart string) ([]string, error) {
	return nil, nil
}

// SearchCharts fake
func (FakeHelmer) SearchCharts(filter string) ([]helm.ChartSummary, error) {
	return nil, nil
}

// SetCWD fake
func (FakeHelmer) SetCWD(dir string) {
}

// SetHelmBinary fake
func (FakeHelmer) SetHelmBinary(binary string) {
}

// SetHost fake
func (FakeHelmer) SetHost(host string) {
}

// StatusRelease fake
func (FakeHelmer) StatusRelease(ns string, releaseName string) error {
	return nil
}

// StatusReleases fake
func (FakeHelmer) StatusReleases(ns string) (map[string]helm.Release, error) {
	return map[string]helm.Release{}, nil
}

// UpdateRepo fake
func (FakeHelmer) UpdateRepo() error {
	return nil
}

// UpgradeChart fake
func (FakeHelmer) UpgradeChart(chart string, releaseName string, ns string, version *string, install bool,
	timeout *int, force bool, wait bool, values []string, valueFiles []string, repo string, username string,
	password string) error {
	return nil
}

// Version fake
func (FakeHelmer) Version(tls bool) (string, error) {
	return "", nil
}
