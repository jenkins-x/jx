package config

import (
	"fmt"

	"strings"

	"crypto/sha1"
	"encoding/base64"

	"github.com/Pallinder/go-randomdata"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

const artifactorySettingsXml = `<settings>
    <!-- sets the local maven repository outside of the ~/.m2 folder for easier mounting of secrets and repo -->
    <localRepository>${user.home}/.mvnrepository</localRepository>
    <!-- lets disable the download progress indicator that fills up logs -->
    <interactiveMode>false</interactiveMode>
    <mirrors>
        <mirror>
            <id>artifactory-releases</id>
            <mirrorOf>external:*</mirrorOf>
            <url>%s/libs-release</url>
        </mirror>
        <mirror>
            <id>artifactory-snapshots</id>
            <mirrorOf>external:*</mirrorOf>
            <url>%s/libs-snapshot</url>
        </mirror>
    </mirrors>
    <servers>
        <server>
            <id>artifactory-releases</id>
            <username>%s</username>
            <password>%s</password>
        </server>
        <server>
            <id>artifactory-snapshots</id>
            <username>%s</username>
            <password>%s</password>
        </server>
    </servers>
    <profiles>
        <profile>
            <id>artifactory</id>
            <properties>
                <altDeploymentRepository>artifactory-snapshots::default::%s/libs-snapshot-local</altDeploymentRepository>
                <altReleaseDeploymentRepository>artifactory-releases::default::%s/libs-release-local</altReleaseDeploymentRepository>
                <altSnapshotDeploymentRepository>artifactory-snapshots::default::%s/libs-snapshot-local</altSnapshotDeploymentRepository>
            </properties>
            <repositories>
                <repository>
                    <snapshots/>
                    <id>artifactory-snapshots</id>
                    <url>%s/libs-snapshots</url>
                </repository>
            </repositories>
        </profile>
        <profile>
            <id>release</id>
            <properties>
                <gpg.executable>gpg</gpg.executable>
                <gpg.passphrase>mysecretpassphrase</gpg.passphrase>
            </properties>
        </profile>
    </profiles>
    <activeProfiles>
        <!--make the profile active all the time -->
        <activeProfile>artifactory</activeProfile>
    </activeProfiles>
</settings>
`

const nexusSettingsXml = `<settings>
      <!-- sets the local maven repository outside of the ~/.m2 folder for easier mounting of secrets and repo -->
      <localRepository>${user.home}/.mvnrepository</localRepository>
      <!-- lets disable the download progress indicator that fills up logs -->
      <interactiveMode>false</interactiveMode>
      <mirrors>
          <mirror>
          <id>nexus</id>
          <mirrorOf>external:*</mirrorOf>
          <url>http://nexus/repository/maven-group/</url>
          </mirror>
      </mirrors>
      <servers>
          <server>
          <id>local-nexus</id>
          <username>admin</username>
          <password>%s</password>
          </server>
      </servers>
      <profiles>
          <profile>
              <id>nexus</id>
              <properties>
                  <altDeploymentRepository>local-nexus::default::http://nexus/repository/maven-snapshots/</altDeploymentRepository>
                  <altReleaseDeploymentRepository>local-nexus::default::http://nexus/repository/maven-releases/</altReleaseDeploymentRepository>
                  <altSnapshotDeploymentRepository>local-nexus::default::http://nexus/repository/maven-snapshots/</altSnapshotDeploymentRepository>
              </properties>
          </profile>
          <profile>
              <id>release</id>
              <properties>
                  <gpg.executable>gpg</gpg.executable>
                  <gpg.passphrase>mysecretpassphrase</gpg.passphrase>
              </properties>
          </profile>
      </profiles>
      <activeProfiles>
          <!--make the profile active all the time -->
          <activeProfile>nexus</activeProfile>
      </activeProfiles>
  </settings>
`

type IngressBasicAuth struct {
	JXBasicAuth string `yaml:"JXBasicAuth"`
}

type ChartMuseum struct {
	ChartMuseumEnv ChartMuseumEnv `yaml:"env"`
}

type ChartMuseumEnv struct {
	ChartMuseumSecret ChartMuseumSecret `yaml:"secret"`
}

type ChartMuseumSecret struct {
	User     string `yaml:"BASIC_AUTH_USER"`
	Password string `yaml:"BASIC_AUTH_PASS"`
}

type Grafana struct {
	GrafanaSecret GrafanaSecret `yaml:"server"`
}

type GrafanaSecret struct {
	User     string `yaml:"adminUser"`
	Password string `yaml:"adminPassword"`
}

type Jenkins struct {
	JenkinsSecret JenkinsAdminSecret `yaml:"Master"`
}

type JenkinsAdminSecret struct {
	Password string `yaml:"AdminPassword"`
}

type PipelineSecrets struct {
	MavenSettingsXML string `yaml:"MavenSettingsXML,omitempty"`
}

type AdminSecretsConfig struct {
	IngressBasicAuth string           `yaml:"JXBasicAuth,omitempty"`
	ChartMuseum      *ChartMuseum     `yaml:"chartmuseum,omitempty"`
	Grafana          *Grafana         `yaml:"grafana,omitempty"`
	Jenkins          *Jenkins         `yaml:"jenkins,omitempty"`
	Nexus            *Nexus           `yaml:"nexus,omitempty"`
	PipelineSecrets  *PipelineSecrets `yaml:"PipelineSecrets,omitempty"`
}

type Nexus struct {
	DefaultAdminPassword string `yaml:"defaultAdminPassword,omitempty"`
}

type AdminSecretsService struct {
	FileName string
	Secrets  AdminSecretsConfig
	Flags    AdminSecretsFlags
}

type AdminSecretsFlags struct {
	DefaultAdminPassword string
	ArtifactoryUrl       string
	ArtifactoryUsername  string
	ArtifactoryPassword  string
}

func (s *AdminSecretsService) AddAdminSecretsValues(cmd *cobra.Command) {

	cmd.Flags().StringVarP(&s.Flags.DefaultAdminPassword, "default-admin-password", "", "", "the default admin password to access Jenkins, Kubernetes Dashboard, Chartmuseum and Nexus")
	cmd.Flags().StringVar(&s.Flags.ArtifactoryUrl, "artifactory-url", "", "Artifactory server URL")
	cmd.Flags().StringVar(&s.Flags.ArtifactoryUsername, "artifactory-user", "", "Artifactory server username")
	cmd.Flags().StringVar(&s.Flags.ArtifactoryPassword, "artifactory-password", "", "Artifactory server password")

	if s.Flags.DefaultAdminPassword == "" {
		s.Flags.DefaultAdminPassword = strings.ToLower(randomdata.SillyName())
	}

}

func (c AdminSecretsConfig) String() (string, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshall helm values %v", err)
	}
	return string(b), nil
}

func (s *AdminSecretsService) NewAdminSecretsConfig(useArtifactory bool) error {
	s.Secrets = AdminSecretsConfig{
		ChartMuseum:     &ChartMuseum{},
		Grafana:         &Grafana{},
		Jenkins:         &Jenkins{},
		PipelineSecrets: &PipelineSecrets{},
		Nexus:           &Nexus{},
	}

	s.Secrets.Jenkins.JenkinsSecret.Password = s.Flags.DefaultAdminPassword
	s.Secrets.ChartMuseum.ChartMuseumEnv.ChartMuseumSecret.User = "admin"
	s.Secrets.ChartMuseum.ChartMuseumEnv.ChartMuseumSecret.Password = s.Flags.DefaultAdminPassword
	s.Secrets.Grafana.GrafanaSecret.User = "admin"
	s.Secrets.Grafana.GrafanaSecret.Password = s.Flags.DefaultAdminPassword
	s.Secrets.Nexus.DefaultAdminPassword = s.Flags.DefaultAdminPassword
	if useArtifactory {
		artifactoryUrl := s.Flags.ArtifactoryUrl
		artifactoryUser := s.Flags.ArtifactoryUsername
		artifactoryPass := s.Flags.ArtifactoryPassword
		s.Secrets.PipelineSecrets.MavenSettingsXML = fmt.Sprintf(artifactorySettingsXml,
			artifactoryUrl, artifactoryUrl, artifactoryUser, artifactoryPass, artifactoryUser, artifactoryPass, artifactoryUrl, artifactoryUrl, artifactoryUrl, artifactoryUrl)
	} else {
		s.Secrets.PipelineSecrets.MavenSettingsXML = fmt.Sprintf(nexusSettingsXml, s.Flags.DefaultAdminPassword)
	}
	hash := HashSha(s.Flags.DefaultAdminPassword)

	s.Secrets.IngressBasicAuth = fmt.Sprintf("admin:{SHA}%s", hash)
	return nil
}

func HashSha(password string) string {
	s := sha1.New()
	s.Write([]byte(password))
	passwordSum := s.Sum(nil)
	return base64.StdEncoding.EncodeToString(passwordSum)
}
