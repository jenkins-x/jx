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

type IngressBasicAuth struct {
	JXBasicAuth string `yaml:"JXBasicAuth"`
}

type ChartMuseum struct {
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

type AdminSecretsConfig struct {
	IngressBasicAuth string       `yaml:"JXBasicAuth,omitempty"`
	ChartMuseum      *ChartMuseum `yaml:"chartmuseum,omitempty"`
	Grafana          *Grafana     `yaml:"grafana,omitempty"`
	Jenkins          *Jenkins     `yaml:"jenkins,omitempty"`
}

type AdminSecretsService struct {
	FileName string
	Secrets  AdminSecretsConfig
	Flags    AdminSecretsFlags
}

type AdminSecretsFlags struct {
	DefaultAdminPassword string
}

func (s *AdminSecretsService) AddAdminSecretsValues(cmd *cobra.Command) {

	cmd.Flags().StringVarP(&s.Flags.DefaultAdminPassword, "default-admin-password", "", "", "the default admin password to access Jenkins, Kubernetes Dashboard, Chartmuseum and Nexus")

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

func (s *AdminSecretsService) NewAdminSecretsConfig() error {
	s.Secrets = AdminSecretsConfig{
		ChartMuseum: &ChartMuseum{},
		Grafana:     &Grafana{},
		Jenkins:     &Jenkins{},
	}

	s.Secrets.Jenkins.JenkinsSecret.Password = s.Flags.DefaultAdminPassword
	s.Secrets.ChartMuseum.ChartMuseumSecret.User = "admin"
	s.Secrets.ChartMuseum.ChartMuseumSecret.Password = s.Flags.DefaultAdminPassword
	s.Secrets.Grafana.GrafanaSecret.User = "admin"
	s.Secrets.Grafana.GrafanaSecret.Password = s.Flags.DefaultAdminPassword

	hash := hashSha(s.Flags.DefaultAdminPassword)

	s.Secrets.IngressBasicAuth = fmt.Sprintf("admin:{SHA}%s", hash)
	return nil
}

func hashSha(password string) string {
	s := sha1.New()
	s.Write([]byte(password))
	passwordSum := []byte(s.Sum(nil))
	return base64.StdEncoding.EncodeToString(passwordSum)
}
