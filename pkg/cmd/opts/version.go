package opts

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/table"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/version"
)

// VersionResolver resolves versions of charts, packages or docker images
type VersionResolver struct {
	VersionsDir string
}

// CreateVersionResolver creates a new VersionResolver service
func (o *CommonOptions) CreateVersionResolver(repo string, gitRef string) (*VersionResolver, error) {
	versionsDir, err := o.CloneJXVersionsRepo(repo, gitRef)
	if err != nil {
		return nil, err
	}
	return &VersionResolver{
		VersionsDir: versionsDir,
	}, nil
}

// GetPackageVersions returns the package versions and a table if they need to be rendered
func (o *CommonOptions) GetPackageVersions(ns string, helmTLS bool) (map[string]string, table.Table) {
	info := util.ColorInfo
	packages := map[string]string{}
	table := o.CreateTable()
	table.AddRow("NAME", "VERSION")
	jxVersion := version.GetVersion()
	table.AddRow("jx", info(jxVersion))
	packages["jx"] = jxVersion
	// Jenkins X version
	releases, _, err := o.Helm().ListReleases(ns)
	if err != nil {
		log.Logger().Warnf("Failed to find helm installs: %s", err)
	} else {
		for _, release := range releases {
			if release.Chart == "jenkins-x-platform" {
				table.AddRow("jenkins x platform", info(release.ChartVersion))
			}
		}
	}
	// Kubernetes version
	client, err := o.KubeClient()
	if err != nil {
		log.Logger().Warnf("Failed to connect to Kubernetes: %s", err)
	} else {
		serverVersion, err := client.Discovery().ServerVersion()
		if err != nil {
			log.Logger().Warnf("Failed to get Kubernetes server version: %s", err)
		} else if serverVersion != nil {
			version := serverVersion.String()
			packages["kubernetesCluster"] = version
			table.AddRow("Kubernetes cluster", info(version))
		}
	}
	// kubectl version
	output, err := o.GetCommandOutput("", "kubectl", "version", "--short")
	if err != nil {
		log.Logger().Warnf("Failed to get kubectl version: %s", err)
	} else {
		for i, line := range strings.Split(output, "\n") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				v := fields[2]
				if v != "" {
					switch i {
					case 0:
						table.AddRow("kubectl", info(v))
						packages["kubectl"] = v
					case 1:
						// Ignore K8S server details as we have these above
					}
				}
			}
		}
	}

	// helm version
	output, err = o.Helm().Version(helmTLS)
	if err != nil {
		log.Logger().Warnf("Failed to get helm version: %s", err)
	} else {
		helmBinary, noTiller, helmTemplate, _ := o.TeamHelmBin()
		if helmBinary == "helm3" || noTiller || helmTemplate {
			table.AddRow("helm client", info(output))
		} else {
			for i, line := range strings.Split(output, "\n") {
				fields := strings.Fields(line)
				if len(fields) > 1 {
					v := fields[1]
					if v != "" {
						switch i {
						case 0:
							table.AddRow("helm client", info(v))
							packages["helm"] = v
						case 1:
							table.AddRow("helm server", info(v))
						}
					}
				}
			}
		}
	}

	// git version
	version, err := o.Git().Version()
	if err != nil {
		log.Logger().Warnf("Failed to get git version: %s", err)
	} else {
		table.AddRow("git", info(version))
		packages["git"] = version
	}
	return packages, table
}

// ResolveDockerImage ensures the given docker image has a valid version if there is one in the version stream
func (v *VersionResolver) ResolveDockerImage(image string) (string, error) {
	return version.ResolveDockerImage(v.VersionsDir, image)
}

// StableVersion returns the stable version of the given kind name
func (v *VersionResolver) StableVersion(kind version.VersionKind, name string) (*version.StableVersion, error) {
	return version.LoadStableVersion(v.VersionsDir, kind, name)
}

// StableVersionNumber returns the stable version number of the given kind name
func (v *VersionResolver) StableVersionNumber(kind version.VersionKind, name string) (string, error) {
	return version.LoadStableVersionNumber(v.VersionsDir, kind, name)
}

// VerifyPackages verifies that the package keys and current version numbers are at the required minimum versions
func (v *VersionResolver) VerifyPackages(packages map[string]string) error {
	errs := []error{}
	keys := util.SortedMapKeys(packages)
	for _, p := range keys {
		version := packages[p]
		if version == "" {
			continue
		}
		err := v.VerifyPackage(p, version)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return util.CombineErrors(errs...)
}

// VerifyPackage verifies the package is of a sufficient version
func (v *VersionResolver) VerifyPackage(name string, currentVersion string) error {
	data, err := version.LoadStableVersion(v.VersionsDir, version.KindPackage, name)
	if err != nil {
		return err
	}
	return data.VerifyPackage(name, currentVersion, v.VersionsDir)
}

// GetVersionNumber returns the version number for the given kind and name or blank string if there is no locked version
func (o *CommonOptions) GetVersionNumber(kind version.VersionKind, name, repo string, gitRef string) (string, error) {
	versioner, err := o.CreateVersionResolver(repo, gitRef)
	if err != nil {
		return "", err
	}
	return versioner.StableVersionNumber(kind, name)
}
