package opts

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/envctx"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/table"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/jenkins-x/jx/pkg/versionstream"
	"github.com/jenkins-x/jx/pkg/versionstream/versionstreamrepo"
)

// EnvironmentContext gets or creates an environment context with the common values for working with requirements, team settings
// and version resolver.
//
// if preferRequirementsFile is true we look for the local `jx-requirements.yml` file first and then fall back to the
// development environment settings; otherwise we try to use the requirements from the environment if present
func (o *CommonOptions) EnvironmentContext(dir string, preferRequirementsFile bool) (*envctx.EnvironmentContext, error) {
	if o.envctx != nil {
		return o.envctx, nil
	}
	var err error
	ec := &envctx.EnvironmentContext{}
	ec.GitOps, ec.DevEnv = o.GetDevEnv()
	if ec.DevEnv == nil {
		ec.DevEnv = kube.CreateDefaultDevEnvironment("jx")
	}
	teamSettings := ec.TeamSettings()

	exists := false
	if preferRequirementsFile {
		// lets default to local file system for the requirements as we being invoked
		// in a `jx boot` pipeline and we have not yet fully populated the `Environment` resources yet
		fileName := ""
		ec.Requirements, fileName, err = config.LoadRequirementsConfig(dir)
		if err != nil {
			return ec, err
		}
		if fileName != "" {
			exists, _ = util.FileExists(fileName)
		}
	}
	if !preferRequirementsFile || !exists {
		// lets try the environment CRD if we have no local file
		req, err := config.GetRequirementsConfigFromTeamSettings(teamSettings)
		if err != nil {
			return ec, err
		}
		if req != nil {
			ec.Requirements = req
		}
	}
	if err != nil {
		return ec, err
	}

	// if we can't find a requirements then lets just create the defaults for now
	if ec.Requirements == nil {
		ec.Requirements, _, err = config.LoadRequirementsConfig(dir)
		if err != nil {
			return ec, err
		}
	}
	err = o.ConfigureCommonOptions(ec.Requirements)
	if err != nil {
		return ec, err
	}
	versionStreamURL := ""
	versionStreamRef := ""
	if preferRequirementsFile {
		versionStreamURL = ec.Requirements.VersionStream.URL
		versionStreamRef = ec.Requirements.VersionStream.Ref
	} else {
		versionStreamURL = teamSettings.VersionStreamURL
		versionStreamRef = teamSettings.VersionStreamRef
	}
	ec.VersionResolver, err = o.CreateVersionResolver(versionStreamURL, versionStreamRef)
	if err != nil {
		return ec, err
	}
	o.envctx = ec
	if o.versionResolver == nil {
		o.versionResolver = ec.VersionResolver
	}
	return ec, nil
}

// SetEnvironmentContext allows the EnvironmentContext to be specified.
// this method is mostly used for tests but can be used to share a cached EnvironmentContext between commands
func (o *CommonOptions) SetEnvironmentContext(envctx *envctx.EnvironmentContext) {
	o.envctx = envctx
}

// CreateVersionResolver creates a new VersionResolver service
func (o *CommonOptions) CreateVersionResolver(repo string, gitRef string) (*versionstream.VersionResolver, error) {
	versionsDir, _, err := o.CloneJXVersionsRepo(repo, gitRef)
	if err != nil {
		return nil, err
	}
	return &versionstream.VersionResolver{
		VersionsDir: versionsDir,
	}, nil
}

// GetVersionResolver gets a VersionResolver, lazy creating one if required so we can reuse it later
func (o *CommonOptions) GetVersionResolver() (*versionstream.VersionResolver, error) {
	var err error
	if o.versionResolver == nil {
		if o.envctx != nil {
			o.versionResolver = o.envctx.VersionResolver
		}
		if o.versionResolver == nil {
			o.versionResolver, err = o.CreateVersionResolver("", "")
		}
	}
	return o.versionResolver, err
}

// SetVersionResolver gets a VersionResolver, lazy creating one if required
func (o *CommonOptions) SetVersionResolver(resolver *versionstream.VersionResolver) {
	o.versionResolver = resolver
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

// GetVersionNumber returns the version number for the given kind and name or blank string if there is no locked version
func (o *CommonOptions) GetVersionNumber(kind versionstream.VersionKind, name, repo string, gitRef string) (string, error) {
	versioner, err := o.CreateVersionResolver(repo, gitRef)
	if err != nil {
		return "", err
	}
	return versioner.StableVersionNumber(kind, name)
}

// CloneJXVersionsRepo clones the jenkins-x versions repo to a local working dir
func (o *CommonOptions) CloneJXVersionsRepo(versionRepository string, versionRef string) (string, string, error) {
	settings, err := o.TeamSettings()
	if err != nil {
		log.Logger().Debugf("Unable to load team settings because %v", err)
	}
	return versionstreamrepo.CloneJXVersionsRepo(versionRepository, versionRef, settings, o.Git(), o.BatchMode, o.AdvancedMode, o.GetIOFileHandles())
}
