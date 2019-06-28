package opts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	randomdata "github.com/Pallinder/go-randomdata"
	filemutex "github.com/alexflint/go-filemutex"
	"github.com/blang/semver"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/cloud/gke/externaldns"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/kube/cluster"
	"github.com/jenkins-x/jx/pkg/kube/services"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/maven"
	"github.com/jenkins-x/jx/pkg/packages"
	"github.com/jenkins-x/jx/pkg/prow"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/jx/pkg/version"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	survey "gopkg.in/AlecAivazis/survey.v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	groovy = `
// imports
import jenkins.model.Jenkins
import jenkins.model.JenkinsLocationConfiguration

// parameters
def jenkinsParameters = [
  url:    '%s/'
]

// get Jenkins location configuration
def jenkinsLocationConfiguration = JenkinsLocationConfiguration.get()

// set Jenkins URL
jenkinsLocationConfiguration.setUrl(jenkinsParameters.url)

// set Jenkins admin email address
jenkinsLocationConfiguration.setAdminAddress(jenkinsParameters.email)

// save current Jenkins state to disk
jenkinsLocationConfiguration.save()
`
)

const (
	AdminSecretsFile            = "adminSecrets.yaml"
	ExtraValuesFile             = "extraValues.yaml"
	ValuesFile                  = "values.yaml"
	JXInstallConfig             = "jx-install-config"
	CloudEnvValuesFile          = "myvalues.yaml"
	CloudEnvSecretsFile         = "secrets.yaml"
	CloudEnvSopsConfigFile      = ".sops.yaml"
	DefaultInstallTimeout       = "6000"
	DefaultCloudEnvironmentsURL = "https://github.com/jenkins-x/cloud-environments"
	DefaultVersionsURL          = "https://github.com/jenkins-x/jenkins-x-versions.git"
)

// Prow keeps install information for prow chart
type Prow struct {
	Version     string
	Chart       string
	SetValues   string
	ReleaseName string
	HMACToken   string
	OAUTHToken  string
}

// DoInstallMissingDependencies install missing dependencies from the given list
func (o *CommonOptions) DoInstallMissingDependencies(install []string) error {
	// install package managers first
	for _, i := range install {
		if i == "brew" {
			log.Logger().Infof("Installing %s", util.ColorInfo(i))
			o.InstallBrew()
			break
		}
	}

	for _, i := range install {
		log.Logger().Infof("Installing %s", util.ColorInfo(i))
		var err error
		switch i {
		case "az":
			err = o.InstallAzureCli()
		case "kubectl":
			err = o.InstallKubectl(false)
		case "gcloud":
			err = o.InstallGcloud()
		case "helm":
			err = o.InstallHelm()
		case "ibmcloud":
			err = o.InstallIBMCloud(false)
		case "glooctl":
			err = o.InstallGlooctl()
		case "tiller":
			err = o.InstallTiller()
		case "helm3":
			err = o.InstallHelm3()
		case "hyperkit":
			err = o.InstallHyperkit()
		case "kops":
			err = o.InstallKops()
		case "kvm":
			err = o.InstallKvm()
		case "kvm2":
			err = o.InstallKvm2()
		case "ksync":
			_, err = o.InstallKSync()
		case "minikube":
			err = o.InstallMinikube()
		case "minishift":
			err = o.InstallMinishift()
		case "oc":
			err = o.InstallOc()
		case "virtualbox":
			err = o.InstallVirtualBox()
		case "xhyve":
			err = o.InstallXhyve()
		case "hyperv":
			err = o.Installhyperv()
		case "terraform":
			err = o.InstallTerraform()
		case "oci":
			err = o.InstallOciCli()
		case "aws":
			err = o.InstallAws()
		case "eksctl":
			err = o.InstallEksCtl(false)
		case "aws-iam-authenticator":
			err = o.InstallAwsIamAuthenticator(false)
		case "kustomize":
			err = o.InstallKustomize()
		default:
			return fmt.Errorf("unknown dependency to install %s\n", i)
		}
		if err != nil {
			return fmt.Errorf("error installing %s: %v\n", i, err)
		}
	}
	return nil
}

// BinaryShouldBeInstalled appends the binary to the deps array if it cannot be found on the $PATH
func BinaryShouldBeInstalled(d string) string {
	_, shouldInstall, err := ShouldInstallBinary(d)
	if err != nil {
		log.Logger().Warnf("Error detecting if binary should be installed: %s", err.Error())
		return ""
	}
	if shouldInstall {
		return d
	}
	return ""
}

// InstallBrew installs brew
func (o *CommonOptions) InstallBrew() error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	log.Logger().Infof("Please enter your root password when prompted by the %s installation", util.ColorInfo("brew"))
	//Make sure to run command through sh in order to get $() expanded.
	return o.RunCommand("sh", "-c", "/usr/bin/ruby -e \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)\"")
}

// ShouldInstallBinary checks if the given binary should be installed
func ShouldInstallBinary(name string) (fileName string, download bool, err error) {
	fileName = packages.BinaryWithExtension(name)
	download = false
	pgmPath, err := exec.LookPath(fileName)
	if err == nil {
		log.Logger().Debugf("%s is already available on your PATH at %s", util.ColorInfo(fileName), util.ColorInfo(pgmPath))
		return
	}

	binDir, err := util.JXBinLocation()
	if err != nil {
		return
	}

	// lets see if its been installed but just is not on the PATH
	exists, err := util.FileExists(filepath.Join(binDir, fileName))
	if err != nil {
		return
	}
	if exists {
		log.Logger().Debugf("Please add %s to your PATH", util.ColorInfo(binDir))
		return
	}
	download = true
	return
}

// UninstallBinary uninstalls given binary
func (o *CommonOptions) UninstallBinary(binDir string, name string) error {
	fileName := name
	if runtime.GOOS == "windows" {
		fileName += ".exe"
	}
	// try to remove the binary from all paths
	var err error
	for {
		path, err := exec.LookPath(fileName)
		if err == nil {
			err := os.Remove(path)
			if err != nil {
				return err
			}
		} else {
			break
		}
	}
	path := filepath.Join(binDir, fileName)
	exists, err := util.FileExists(path)
	if err != nil {
		return nil
	}
	if exists {
		return os.Remove(path)
	}
	return nil
}

// InstallOrUpdateBinaryOptions options for install or update binary
type InstallOrUpdateBinaryOptions struct {
	Binary                       string
	GitHubOrganization           string
	DownloadUrlTemplate          string
	DownloadUrlTemplateLowerCase bool
	Version                      string
	SkipPathScan                 bool
	VersionExtractor             packages.VersionExtractor
	Archived                     bool
	ArchiveDirectory             string
}

// InstallOrUpdateBinary installs or updates a binary
func (o *CommonOptions) InstallOrUpdateBinary(options InstallOrUpdateBinaryOptions) error {
	isInstalled, err := packages.IsBinaryWithProperVersionInstalled(options.Binary, options.Version, options.VersionExtractor)
	if err != nil {
		return err
	}
	if isInstalled {
		return nil
	}

	downloadUrlTemplate := options.DownloadUrlTemplate
	if !options.Archived {
		downloadUrlTemplate = packages.BinaryWithExtension(downloadUrlTemplate)
	}
	urlTemplate, err := template.New(options.Binary).Parse(downloadUrlTemplate)
	if err != nil {
		return err
	}
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	fileName := options.Binary
	if !options.SkipPathScan {
		installFilename, flag, err := ShouldInstallBinary(options.Binary)
		fileName = installFilename
		if err != nil || !flag {
			return err
		}
	}

	if options.Version == "" {
		configDir, err := util.ConfigDir()
		if err != nil {
			return err
		}
		versionFile := filepath.Join(configDir, "jenkins-x-versions", "packages", options.Binary+".yml")
		ver, err := version.LoadStableVersionFile(versionFile)
		if err != nil {
			return err
		}
		if ver.Version != "" {
			options.Version = ver.Version
		}
	}

	if options.Version == "" {
		options.Version, err = util.GetLatestVersionStringFromGitHub(options.GitHubOrganization, options.Binary)
		if err != nil {
			return err
		}
	}
	extension := "tar.gz"
	if runtime.GOOS == "windows" {
		extension = "zip"
	}
	clientUrlBuffer := bytes.NewBufferString("")
	variables := map[string]string{"version": options.Version, "os": runtime.GOOS, "osTitle": strings.Title(runtime.GOOS), "arch": runtime.GOARCH, "extension": extension}
	urlTemplate.Execute(clientUrlBuffer, variables)
	fullPath := filepath.Join(binDir, fileName)
	tarFile := fullPath
	if options.Archived {
		tarFile = tarFile + "." + extension
	}
	downloadUrl := clientUrlBuffer.String()
	if options.DownloadUrlTemplateLowerCase {
		downloadUrl = strings.ToLower(downloadUrl)
	}
	err = packages.DownloadFile(downloadUrl, tarFile)
	if err != nil {
		return err
	}
	fileNameInArchive := fileName
	if options.ArchiveDirectory != "" {
		fileNameInArchive = filepath.Join(options.ArchiveDirectory, fileName)
	}
	if options.Archived {
		if extension == "zip" {
			zipDir := filepath.Join(binDir, options.Binary+"-tmp-"+uuid.NewUUID().String())
			err = os.MkdirAll(zipDir, util.DefaultWritePermissions)
			if err != nil {
				return err
			}
			err = util.Unzip(tarFile, zipDir)
			if err != nil {
				return err
			}

			f := filepath.Join(zipDir, fileNameInArchive)
			exists, err := util.FileExists(f)
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("Could not find file %s inside the downloaded file!", f)
			}
			err = os.Rename(f, fullPath)
			if err != nil {
				return err
			}
			err = os.RemoveAll(zipDir)
		} else {
			err = util.UnTargz(tarFile, binDir, []string{options.Binary, fileNameInArchive})
		}
		if err != nil {
			return err
		}
		err = os.Remove(tarFile)
		if err != nil {
			return err
		}
	}

	err = packages.RememberInstalledPackage(options.Binary, options.Version)
	if err != nil {
		return err
	}

	return os.Chmod(fullPath, 0755)
}

// InstallBrewIfRequired installs brew if required
func (o *CommonOptions) InstallBrewIfRequired() error {
	if runtime.GOOS != "darwin" || o.NoBrew {
		return nil
	}

	_, flag, err := ShouldInstallBinary("brew")
	if err != nil || !flag {
		return err
	}
	return o.InstallBrew()
}

// InstallKubectl installs kubectl
func (o *CommonOptions) InstallKubectl(skipPathScan bool) error {
	return o.InstallKubectlWithVersion(packages.KubectlVersion, skipPathScan)
}

// InstallKubectlWithVersion install a specific version of kubectl
func (o *CommonOptions) InstallKubectlWithVersion(version string, skipPathScan bool) error {
	return o.InstallOrUpdateBinary(InstallOrUpdateBinaryOptions{
		Binary:                       "kubectl",
		GitHubOrganization:           "",
		DownloadUrlTemplate:          "https://storage.googleapis.com/kubernetes-release/release/v{{.version}}/bin/{{.osTitle}}/{{.arch}}/kubectl",
		DownloadUrlTemplateLowerCase: true,
		Version:                      version,
		SkipPathScan:                 skipPathScan,
		VersionExtractor:             nil,
		Archived:                     false,
	})
}

// InstallGlooctl Installs glooctl tool
func (o *CommonOptions) InstallGlooctl() error {
	if runtime.GOOS == "darwin" && !o.NoBrew {
		return o.RunCommand("brew", "install", "glooctl")
	}
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	fileName, flag, err := ShouldInstallBinary("glooctl")
	if err != nil || !flag {
		return err
	}

	latestVersion, err := util.GetLatestVersionFromGitHub("solo-io", "gloo")
	if err != nil {
		return fmt.Errorf("unable to get latest version for github.com/%s/%s %v", "kubernetes-sigs", "kustomize", err)
	}

	suffix := runtime.GOARCH
	if runtime.GOOS == "windows" {
		suffix += ".exe"
	}
	clientURL := fmt.Sprintf("https://github.com/solo-io/gloo/releases/download/v%v/glooctl-%s-%s", latestVersion, runtime.GOOS, suffix)
	fullPath := filepath.Join(binDir, fileName)
	tmpFile := fullPath + ".tmp"
	err = packages.DownloadFile(clientURL, tmpFile)
	if err != nil {
		return err
	}
	err = util.RenameFile(tmpFile, fullPath)
	if err != nil {
		return err
	}
	return os.Chmod(fullPath, 0755)
}

// InstallKustomize installs kustomize
func (o *CommonOptions) InstallKustomize() error {
	if runtime.GOOS == "darwin" && !o.NoBrew {
		return o.RunCommand("brew", "install", "kustomize")
	}
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	fileName, flag, err := ShouldInstallBinary("kustomize")
	if err != nil || !flag {
		return err
	}

	latestVersion, err := util.GetLatestVersionFromGitHub("kubernetes-sigs", "kustomize")
	if err != nil {
		return fmt.Errorf("unable to get latest version for github.com/%s/%s %v", "kubernetes-sigs", "kustomize", err)
	}

	clientURL := fmt.Sprintf("https://github.com/kubernetes-sigs/kustomize/releases/download/v%v/kustomize_%s_%s_%s", latestVersion, latestVersion, runtime.GOOS, runtime.GOARCH)
	fullPath := filepath.Join(binDir, fileName)
	tmpFile := fullPath + ".tmp"
	err = packages.DownloadFile(clientURL, tmpFile)
	if err != nil {
		return err
	}
	err = util.RenameFile(tmpFile, fullPath)
	if err != nil {
		return err
	}
	return os.Chmod(fullPath, 0755)
}

// InstallOc installs oc cli
func (o *CommonOptions) InstallOc() error {
	// need to fix the version we download as not able to work out the oc sha in the URL yet
	sha := "191fece"
	latestVersion := "3.9.0"

	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	binary := "oc"
	fileName, flag, err := ShouldInstallBinary(binary)
	if err != nil || !flag {
		return err
	}

	var arch string
	clientURL := fmt.Sprintf("https://github.com/openshift/origin/releases/download/v%s/openshift-origin-client-tools-v%s-%s", latestVersion, latestVersion, sha)

	extension := ".zip"
	switch runtime.GOOS {
	case "windows":
		clientURL += "-windows.zip"
	case "darwin":
		clientURL += "-mac.zip"
	default:
		switch runtime.GOARCH {
		case "amd64":
			arch = "64bit"
		case "386":
			arch = "32bit"
		}
		extension = ".tar.gz"
		clientURL += fmt.Sprintf("-%s-%s.tar.gz", runtime.GOOS, arch)
	}

	fullPath := filepath.Join(binDir, fileName)
	tarFile := filepath.Join(binDir, "oc.tgz")
	if extension == ".zip" {
		tarFile = filepath.Join(binDir, "oc.zip")
	}
	err = packages.DownloadFile(clientURL, tarFile)
	if err != nil {
		return err
	}

	if extension == ".zip" {
		zipDir := filepath.Join(binDir, "oc-tmp-"+uuid.NewUUID().String())
		err = os.MkdirAll(zipDir, util.DefaultWritePermissions)
		if err != nil {
			return err
		}
		err = util.Unzip(tarFile, zipDir)
		if err != nil {
			return err
		}
		f := filepath.Join(zipDir, fileName)
		exists, err := util.FileExists(f)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("Could not find file %s inside the downloaded oc.zip!", f)
		}
		err = os.Rename(f, fullPath)
		if err != nil {
			return err
		}
		err = os.RemoveAll(zipDir)
	} else {
		err = util.UnTargz(tarFile, binDir, []string{binary, fileName})
	}
	if err != nil {
		return err
	}
	err = os.Remove(tarFile)
	if err != nil {
		return err
	}
	return os.Chmod(fullPath, 0755)
}

// InstallHyperkit installs hyperkit
func (o *CommonOptions) InstallHyperkit() error {
	/*
		info, err := o.getCommandOutput("", "docker-machine-driver-hyperkit")
		if strings.Contains(info, "Docker") {
			o.Printf("docker-machine-driver-hyperkit is already installed\n")
			return nil
		}
		o.Printf("Result: %s and %v\n", info, err)
		err = o.runCommand("curl", "-LO", "https://storage.googleapis.com/minikube/releases/latest/docker-machine-driver-hyperkit")
		if err != nil {
			return err
		}

		err = o.runCommand("chmod", "+x", "docker-machine-driver-hyperkit")
		if err != nil {
			return err
		}

		log.Logger().Warn("Installing hyperkit does require sudo to perform some actions, for more details see https://github.com/kubernetes/minikube/blob/master/docs/drivers.md#hyperkit-driver")

		err = o.runCommand("sudo", "mv", "docker-machine-driver-hyperkit", "/usr/local/bin/")
		if err != nil {
			return err
		}

		err = o.runCommand("sudo", "chown", "root:wheel", "/usr/local/bin/docker-machine-driver-hyperkit")
		if err != nil {
			return err
		}

		return o.runCommand("sudo", "chmod", "u+s", "/usr/local/bin/docker-machine-driver-hyperkit")
	*/
	return nil
}

// InstallKvm installs kvm
func (o *CommonOptions) InstallKvm() error {
	log.Logger().Warnf("We cannot yet automate the installation of KVM - can you install this manually please?\nPlease see: https://www.linux-kvm.org/page/Downloads")
	return nil
}

// InstallKvm2 install kvm2
func (o *CommonOptions) InstallKvm2() error {
	log.Logger().Warnf("We cannot yet automate the installation of KVM with KVM2 driver - can you install this manually please?\nPlease see: https://www.linux-kvm.org/page/Downloads " +
		"and https://github.com/kubernetes/minikube/blob/master/docs/drivers.md#kvm2-driver")
	return nil
}

// InstallVirtualBox installs virtual box
func (o *CommonOptions) InstallVirtualBox() error {
	log.Logger().Warnf("We cannot yet automate the installation of VirtualBox - can you install this manually please?\nPlease see: https://www.virtualbox.org/wiki/Downloads")
	return nil
}

// InstallXhyve installs xhyve
func (o *CommonOptions) InstallXhyve() error {
	info, err := o.GetCommandOutput("", "brew", "info", "docker-machine-driver-xhyve")

	if err != nil || strings.Contains(info, "Not installed") {
		err = o.RunCommand("brew", "install", "docker-machine-driver-xhyve")
		if err != nil {
			return err
		}

		brewPrefix, err := o.GetCommandOutput("", "brew", "--prefix")
		if err != nil {
			return err
		}

		file := brewPrefix + "/opt/docker-machine-driver-xhyve/bin/docker-machine-driver-xhyve"
		err = o.RunCommand("sudo", "chown", "root:wheel", file)
		if err != nil {
			return err
		}

		err = o.RunCommand("sudo", "chmod", "u+s", file)
		if err != nil {
			return err
		}
		log.Logger().Info("xhyve driver installed")
	} else {
		pgmPath, _ := exec.LookPath("docker-machine-driver-xhyve")
		log.Logger().Infof("xhyve driver is already available on your PATH at %s", pgmPath)
	}
	return nil
}

// Installhyperv installs hyperv
func (o *CommonOptions) Installhyperv() error {
	info, err := o.GetCommandOutput("", "powershell", "Get-WindowsOptionalFeature", "-FeatureName", "Microsoft-Hyper-V-All", "-Online")

	if err != nil {
		return err
	}
	if strings.Contains(info, "Disabled") {

		log.Logger().Info("hyperv is Disabled, this computer will need to restart\n and after restart you will need to rerun your inputted commmand.")

		message := fmt.Sprintf("Would you like to restart your computer?")

		if util.Confirm(message, true, "Please indicate if you would like to restart your computer.", o.In, o.Out, o.Err) {

			err = o.RunCommand("powershell", "Enable-WindowsOptionalFeature", "-Online", "-FeatureName", "Microsoft-Hyper-V", "-All", "-NoRestart")
			if err != nil {
				return err
			}
			err = o.RunCommand("powershell", "Restart-Computer")
			if err != nil {
				return err
			}

		} else {
			err = errors.New("hyperv was not Disabled")
			return err
		}

	} else {
		log.Logger().Info("hyperv is already Enabled")
	}
	return nil
}

// InstallVaultCli installs vault cli
func (o *CommonOptions) InstallVaultCli() error {
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	binary := "vault"
	fileName, flag, err := ShouldInstallBinary(binary)
	if err != nil || !flag {
		return err
	}
	latestVersion, err := util.GetLatestFullTagFromGithub("hashicorp", "vault")
	if err != nil {
		return err
	}
	// Strip the v off the beginning of the version number
	latestVersion = strings.Replace(latestVersion, "v", "", 1)

	clientURL := fmt.Sprintf("https://releases.hashicorp.com/vault/%s/vault_%s_%s_%s.zip", latestVersion, latestVersion, runtime.GOOS, runtime.GOARCH)
	fullPath := filepath.Join(binDir, fileName)
	tarFile := fullPath + ".zip"
	err = packages.DownloadFile(clientURL, tarFile)
	if err != nil {
		return err
	}
	err = util.UnzipSpecificFiles(tarFile, binDir, fileName)
	if err != nil {
		return err
	}
	err = os.Remove(tarFile)
	if err != nil {
		return err
	}
	err = os.Chmod(fullPath, 0755)
	return err
}

// InstallHelm install helm cli
func (o *CommonOptions) InstallHelm() error {
	binary := "helm"

	if runtime.GOOS == "darwin" && !o.NoBrew {
		err := o.RunCommand("brew", "install", "kubernetes-helm")
		if err != nil {
			return err
		}
		return o.installHelmSecretsPlugin(binary, true)
	}

	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}

	fileName, flag, err := ShouldInstallBinary(binary)
	if err != nil || !flag {
		return err
	}
	latestVersion, err := util.GetLatestVersionFromGitHub("kubernetes", "helm")
	if err != nil {
		return err
	}
	clientURL := fmt.Sprintf("https://storage.googleapis.com/kubernetes-helm/helm-v%s-%s-%s.tar.gz", latestVersion, runtime.GOOS, runtime.GOARCH)
	fullPath := filepath.Join(binDir, fileName)
	tarFile := fullPath + ".tgz"
	err = packages.DownloadFile(clientURL, tarFile)
	if err != nil {
		return err
	}
	err = util.UnTargz(tarFile, binDir, []string{binary, fileName})
	if err != nil {
		return err
	}
	err = os.Remove(tarFile)
	if err != nil {
		return err
	}
	err = os.Chmod(fullPath, 0755)
	if err != nil {
		return err
	}
	return o.installHelmSecretsPlugin(fullPath, true)
}

// InstallTiller installs tiller
func (o *CommonOptions) InstallTiller() error {
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	binary := "tiller"
	fileName := binary
	if runtime.GOOS == "windows" {
		fileName += ".exe"
	}
	// TODO workaround until 2.11.x GA is released
	latestVersion := "2.11.0-rc.3"
	/*
		latestVersion, err := util.GetLatestVersionFromGitHub("kubernetes", "helm")
			if err != nil {
				return err
			}
	*/
	clientURL := fmt.Sprintf("https://storage.googleapis.com/kubernetes-helm/helm-v%s-%s-%s.tar.gz", latestVersion, runtime.GOOS, runtime.GOARCH)
	fullPath := filepath.Join(binDir, fileName)
	helmFullPath := filepath.Join(binDir, "helm")
	tarFile := fullPath + ".tgz"
	err = packages.DownloadFile(clientURL, tarFile)
	if err != nil {
		return err
	}
	err = util.UnTargz(tarFile, binDir, []string{binary, fileName, "helm"})
	if err != nil {
		return err
	}
	err = os.Remove(tarFile)
	if err != nil {
		return err
	}
	err = os.Chmod(fullPath, 0755)
	if err != nil {
		return err
	}
	err = helm.StartLocalTillerIfNotRunning()
	if err != nil {
		return err
	}
	return o.installHelmSecretsPlugin(helmFullPath, true)
}

// InstallHelm3 installs helm3 cli
func (o *CommonOptions) InstallHelm3() error {
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	binary := "helm3"
	fileName, flag, err := ShouldInstallBinary(binary)
	if err != nil || !flag {
		return err
	}

	// https://get.helm.sh/helm-v3.0.0-alpha.1-darwin-amd64.tar.gz
	latestVersion := "v3.0.0-alpha.1"
	clientURL := fmt.Sprintf("https://get.helm.sh/helm-%v-%s-%s.tar.gz", latestVersion, runtime.GOOS, runtime.GOARCH)

	tmpDir := filepath.Join(binDir, "helm3.tmp")
	err = os.MkdirAll(tmpDir, util.DefaultWritePermissions)
	if err != nil {
		return err
	}
	fullPath := filepath.Join(binDir, binary)
	tarFile := filepath.Join(tmpDir, fileName+".tgz")
	err = packages.DownloadFile(clientURL, tarFile)
	if err != nil {
		return err
	}
	err = util.UnTargz(tarFile, tmpDir, []string{"helm", "helm"})
	if err != nil {
		return err
	}
	err = os.Remove(tarFile)
	if err != nil {
		return err
	}
	err = os.Rename(filepath.Join(tmpDir, "helm"), fullPath)
	if err != nil {
		return err
	}
	err = os.RemoveAll(tmpDir)
	if err != nil {
		return err
	}

	err = os.Chmod(fullPath, 0755)
	if err != nil {
		return err
	}
	return o.installHelmSecretsPlugin(fullPath, false)
}

func (o *CommonOptions) installHelmSecretsPlugin(helmBinary string, clientOnly bool) error {
	log.Logger().Infof("Installing %s", util.ColorInfo("helm secrets plugin"))
	err := o.Helm().Init(clientOnly, "", "", false)
	if err != nil {
		return errors.Wrap(err, "failed to initialize helm")
	}
	// remove the plugin just in case is already installed
	cmd := util.Command{
		Name: helmBinary,
		Args: []string{"plugin", "remove", "secrets"},
	}
	_, err = cmd.RunWithoutRetry()
	if err != nil && !strings.Contains(err.Error(), "secrets not found") {
		return errors.Wrap(err, "failed to remove helm secrets")
	}
	cmd = util.Command{
		Name: helmBinary,
		Args: []string{"plugin", "install", "https://github.com/futuresimple/helm-secrets"},
	}
	_, err = cmd.RunWithoutRetry()
	// Workaround for Helm install on Windows caused by https://github.com/helm/helm/issues/4418
	if err != nil && runtime.GOOS == "windows" && strings.Contains(err.Error(), "Error: symlink") {
		// The install _does_ seem to work, but we get an error - catch this on windows and lob it in the bin
		return nil
	}
	// End of Workaround
	return err
}

// InstallMavenIfRequired installs maven if not available
func (o *CommonOptions) InstallMavenIfRequired() error {
	homeDir, err := util.ConfigDir()
	if err != nil {
		return err
	}
	m, err := filemutex.New(homeDir + "/jx.lock")
	if err != nil {
		panic(err)
	}
	m.Lock()

	cmd := util.Command{
		Name: "mvn",
		Args: []string{"-v"},
	}
	_, err = cmd.RunWithoutRetry()
	if err == nil {
		m.Unlock()
		return nil
	}
	// lets assume maven is not installed so lets download it
	clientURL := fmt.Sprintf("http://central.maven.org/maven2/org/apache/maven/apache-maven/%s/apache-maven-%s-bin.zip", maven.MavenVersion, maven.MavenVersion)

	log.Logger().Infof("Apache Maven is not installed so lets download: %s", util.ColorInfo(clientURL))

	mvnDir := filepath.Join(homeDir, "maven")
	mvnTmpDir := filepath.Join(homeDir, "maven-tmp")
	zipFile := filepath.Join(homeDir, "mvn.zip")

	err = os.MkdirAll(mvnDir, util.DefaultWritePermissions)
	if err != nil {
		m.Unlock()
		return err
	}

	log.Logger().Info("\ndownloadFile")
	err = packages.DownloadFile(clientURL, zipFile)
	if err != nil {
		m.Unlock()
		return err
	}

	log.Logger().Info("\nutil.Unzip")
	err = util.Unzip(zipFile, mvnTmpDir)
	if err != nil {
		m.Unlock()
		return err
	}

	// lets find a directory inside the unzipped folder
	log.Logger().Info("\nReadDir")
	files, err := ioutil.ReadDir(mvnTmpDir)
	if err != nil {
		m.Unlock()
		return err
	}
	for _, f := range files {
		name := f.Name()
		if f.IsDir() && strings.HasPrefix(name, "apache-maven") {
			os.RemoveAll(mvnDir)

			err = os.Rename(filepath.Join(mvnTmpDir, name), mvnDir)
			if err != nil {
				m.Unlock()
				return err
			}
			log.Logger().Infof("Apache Maven is installed at: %s", util.ColorInfo(mvnDir))
			m.Unlock()
			err = os.Remove(zipFile)
			if err != nil {
				m.Unlock()
				return err
			}
			err = os.RemoveAll(mvnTmpDir)
			if err != nil {
				m.Unlock()
				return err
			}
			m.Unlock()
			return nil
		}
	}
	m.Unlock()
	return fmt.Errorf("Could not find an apache-maven folder inside the unzipped maven distro at %s", mvnTmpDir)
}

// InstallTerraform installs terraform
func (o *CommonOptions) InstallTerraform() error {
	if runtime.GOOS == "darwin" && !o.NoBrew {
		return o.RunCommand("brew", "install", "terraform")
	}

	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	binary := "terraform"
	fileName, flag, err := ShouldInstallBinary(binary)
	if err != nil || !flag {
		return err
	}
	latestVersion, err := util.GetLatestVersionFromGitHub("hashicorp", "terraform")
	if err != nil {
		return err
	}

	clientURL := fmt.Sprintf("https://releases.hashicorp.com/terraform/%s/terraform_%s_%s_%s.zip", latestVersion, latestVersion, runtime.GOOS, runtime.GOARCH)
	fullPath := filepath.Join(binDir, fileName)
	zipFile := fullPath + ".zip"
	err = packages.DownloadFile(clientURL, zipFile)
	if err != nil {
		return err
	}
	err = util.Unzip(zipFile, binDir)
	if err != nil {
		return err
	}
	err = os.Remove(zipFile)
	if err != nil {
		return err
	}
	return os.Chmod(fullPath, 0755)
}

// GetLatestJXVersion returns latest jx version
func (o *CommonOptions) GetLatestJXVersion() (semver.Version, error) {
	if runtime.GOOS == "darwin" && !o.NoBrew {
		log.Logger().Debugf("Locating latest JX version from HomeBrew")
		// incase auto-update is not enabled, lets perform an explicit brew update first
		brewUpdate, err := o.GetCommandOutput("", "brew", "update")
		if err != nil {
			log.Logger().Errorf("unable to update brew - %s", brewUpdate)
			return semver.Version{}, err
		}
		log.Logger().Debugf("updating brew - %s", brewUpdate)

		brewInfo, err := o.GetCommandOutput("", "brew", "info", "--json", "jx")
		if err != nil {
			log.Logger().Errorf("unable to get brew info for jx - %s", brewInfo)
			return semver.Version{}, err
		}

		v, err := o.LatestJxBrewVersion(brewInfo)
		if err != nil {
			return semver.Version{}, err
		}

		return semver.Make(v)
	}
	log.Logger().Debugf("Locating latest JX version from GitHub")
	return util.GetLatestVersionFromGitHub("jenkins-x", "jx")
}

func (o *CommonOptions) LatestJxBrewVersion(jsonInfo string) (string, error) {
	var brewInfo []brewInfo
	err := json.Unmarshal([]byte(jsonInfo), &brewInfo)
	if err != nil {
		return "", err
	}
	return brewInfo[0].Versions.Stable, nil
}

// BrewInfo contains some of the `brew info` data.
type brewInfo struct {
	Name     string
	Outdated bool
	Versions struct {
		Stable string
	}
}

// InstallKops installs kops
func (o *CommonOptions) InstallKops() error {
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	binary := "kops"
	fileName, flag, err := ShouldInstallBinary(binary)
	if err != nil || !flag {
		return err
	}
	latestVersion, err := util.GetLatestVersionStringFromGitHub("kubernetes", "kops")
	if err != nil {
		return err
	}
	clientURL := fmt.Sprintf("https://github.com/kubernetes/kops/releases/download/%s/kops-%s-%s", latestVersion, runtime.GOOS, runtime.GOARCH)
	fullPath := filepath.Join(binDir, fileName)
	tmpFile := fullPath + ".tmp"
	err = packages.DownloadFile(clientURL, tmpFile)
	if err != nil {
		return err
	}
	err = util.RenameFile(tmpFile, fullPath)
	if err != nil {
		return err
	}
	return os.Chmod(fullPath, 0755)
}

// InstallKSync install ksync
func (o *CommonOptions) InstallKSync() (string, error) {
	binDir, err := util.JXBinLocation()
	if err != nil {
		return "", err
	}
	binary := "ksync"
	fileName, flag, err := ShouldInstallBinary(binary)
	if err != nil || !flag {
		// Exec `ksync` to find the version
		ksyncCmd := util.Command{
			Name: fileName,
			Args: []string{
				"version",
			},
		}
		// Explicitly ignore any errors from ksync version, as we just need the output!
		res, _ := ksyncCmd.RunWithoutRetry()
		lines := strings.Split(res, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Git Tag:") {
				return strings.TrimSpace(strings.TrimPrefix(line, "Git Tag:")), nil
			}
		}

		return "", fmt.Errorf("unable to find version of ksync")
	}
	latestVersion, err := util.GetLatestVersionFromGitHub("vapor-ware", "ksync")
	if err != nil {
		return "", err
	}
	clientURL := fmt.Sprintf("https://github.com/vapor-ware/ksync/releases/download/%s/ksync_%s_%s", latestVersion, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		clientURL += ".exe"
	}
	fullPath := filepath.Join(binDir, fileName)
	tmpFile := fullPath + ".tmp"
	err = packages.DownloadFile(clientURL, tmpFile)
	if err != nil {
		return "", err
	}
	err = util.RenameFile(tmpFile, fullPath)
	if err != nil {
		return "", err
	}
	return latestVersion.String(), os.Chmod(fullPath, 0755)
}

// InstallJx installs jx cli
func (o *CommonOptions) InstallJx(upgrade bool, version string) error {
	log.Logger().Debugf("installing jx %s", version)
	if runtime.GOOS == "darwin" && !o.NoBrew {
		if upgrade {
			return o.RunCommand("brew", "upgrade", "jx")
		} else {
			return o.RunCommand("brew", "install", "jx")
		}
	}
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	// Check for jx binary in non standard path and install there instead if found...
	nonStandardBinDir, err := util.JXBinaryLocation()
	if err == nil && binDir != nonStandardBinDir {
		binDir = nonStandardBinDir
	}
	binary := "jx"
	fileName := binary
	if !upgrade {
		f, flag, err := ShouldInstallBinary(binary)
		if err != nil || !flag {
			return err
		}
		fileName = f
	}
	org := "jenkins-x"
	repo := "jx"
	if version == "" {
		latestVersion, err := util.GetLatestVersionFromGitHub(org, repo)
		if err != nil {
			return err
		}
		version = fmt.Sprintf("%s", latestVersion)
	}
	extension := "tar.gz"
	if runtime.GOOS == "windows" {
		extension = "zip"
	}
	clientURL := fmt.Sprintf("https://github.com/"+org+"/"+repo+"/releases/download/v%s/"+binary+"-%s-%s.%s", version, runtime.GOOS, runtime.GOARCH, extension)
	fullPath := filepath.Join(binDir, fileName)
	if runtime.GOOS == "windows" {
		fullPath += ".exe"
	}
	tmpArchiveFile := fullPath + ".tmp"
	err = packages.DownloadFile(clientURL, tmpArchiveFile)
	if err != nil {
		return err
	}
	// Untar the new binary into a temp directory
	jxHome, err := util.ConfigDir()
	if err != nil {
		return err
	}

	if runtime.GOOS != "windows" {
		err = util.UnTargz(tmpArchiveFile, jxHome, []string{binary, fileName})
		if err != nil {
			return err
		}
		err = os.Remove(tmpArchiveFile)
		if err != nil {
			return err
		}
		err = os.Remove(filepath.Join(binDir, "jx"))
		if err != nil && o.Verbose {
			log.Logger().Infof("Skipping removal of old jx binary: %s", err)
		}
		// Copy over the new binary
		err = os.Rename(filepath.Join(jxHome, "jx"), filepath.Join(binDir, "jx"))
		if err != nil {
			return err
		}
	} else { // windows
		windowsBinaryFromArchive := "jx-windows-amd64.exe"
		err = util.UnzipSpecificFiles(tmpArchiveFile, jxHome, windowsBinaryFromArchive)
		if err != nil {
			return err
		}
		err = os.Remove(tmpArchiveFile)
		if err != nil {
			return err
		}
		// A standard remove and rename (or overwrite) will not work as the file will be locked as windows is running it
		// the trick is to rename to a tempfile :-o
		// this will leave old files around but well at least it updates.
		// we could schedule the file for cleanup at next boot but....
		// HKLM\System\CurrentControlSet\Control\Session Manager\PendingFileRenameOperations
		err = os.Rename(filepath.Join(binDir, "jx.exe"), filepath.Join(binDir, "jx.exe.deleteme"))
		// if we can not rename it this i pretty fatal as we won;t be able to overwrite either
		if err != nil {
			return err
		}
		// Copy over the new binary
		err = os.Rename(filepath.Join(jxHome, windowsBinaryFromArchive), filepath.Join(binDir, "jx.exe"))
		if err != nil {
			return err
		}
	}
	log.Logger().Infof("Jenkins X client has been installed into %s", util.ColorInfo(fullPath))
	return os.Chmod(fullPath, 0755)
}

// InstallMinikube installs minikube
func (o *CommonOptions) InstallMinikube() error {
	if runtime.GOOS == "darwin" && !o.NoBrew {
		return o.RunCommand("brew", "cask", "install", "minikube")
	}

	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	fileName, flag, err := ShouldInstallBinary("minikube")
	if err != nil || !flag {
		return err
	}
	latestVersion, err := util.GetLatestVersionFromGitHub("kubernetes", "minikube")
	if err != nil {
		return err
	}
	clientURL := fmt.Sprintf("https://github.com/kubernetes/minikube/releases/download/v%s/minikube-%s-%s", latestVersion, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		clientURL += ".exe"
	}
	fullPath := filepath.Join(binDir, fileName)
	tmpFile := fullPath + ".tmp"
	err = packages.DownloadFile(clientURL, tmpFile)
	if err != nil {
		return err
	}
	err = util.RenameFile(tmpFile, fullPath)
	if err != nil {
		return err
	}
	return os.Chmod(fullPath, 0755)
}

// InstallMinishift installs minishift
func (o *CommonOptions) InstallMinishift() error {
	if runtime.GOOS == "darwin" && !o.NoBrew {
		return o.RunCommand("brew", "cask", "install", "minishift")
	}

	binDir, err := util.JXBinLocation()
	binary := "minishift"
	if err != nil {
		return err
	}
	fileName, flag, err := ShouldInstallBinary(binary)
	if err != nil || !flag {
		return err
	}
	latestVersion, err := util.GetLatestVersionFromGitHub(binary, binary)
	if err != nil {
		return err
	}
	clientURL := fmt.Sprintf("https://github.com/minishift/minishift/releases/download/v%s/minishift-%s-%s-%s.tgz", latestVersion, latestVersion, runtime.GOOS, runtime.GOARCH)
	fullPath := filepath.Join(binDir, fileName)
	tarFile := fullPath + ".tgz"
	err = packages.DownloadFile(clientURL, tarFile)
	if err != nil {
		return err
	}
	err = util.UnTargz(tarFile, binDir, []string{binary, fileName})
	if err != nil {
		return err
	}
	return os.Chmod(fullPath, 0755)
}

// InstallGcloud installs gcloud cli
func (o *CommonOptions) InstallGcloud() error {
	if runtime.GOOS != "darwin" || o.NoBrew {
		return errors.New("please install missing gcloud sdk - see https://cloud.google.com/sdk/downloads#interactive")
	}
	err := o.RunCommand("brew", "tap", "caskroom/cask")
	if err != nil {
		return err
	}

	return o.RunCommand("brew", "cask", "install", "google-cloud-sdk")
}

// InstallAzureCli installs azure cli
func (o *CommonOptions) InstallAzureCli() error {
	return o.RunCommand("brew", "install", "azure-cli")
}

// InstallOciCli installs oci cli
func (o *CommonOptions) InstallOciCli() error {
	var err error
	filePath := "./install.sh"
	log.Logger().Info("Installing OCI CLI...")
	err = o.RunCommand("curl", "-LO", "https://raw.githubusercontent.com/oracle/oci-cli/master/scripts/install/install.sh")

	if err != nil {
		return err
	}
	os.Chmod(filePath, 0755)

	err = o.RunCommandVerbose(filePath, "--accept-all-defaults")
	if err != nil {
		return err
	}

	return os.Remove(filePath)
}

// InstallAws installs aws
func (o *CommonOptions) InstallAws() error {
	// TODO
	return nil
}

// InstallEksCtl installs eks cli
func (o *CommonOptions) InstallEksCtl(skipPathScan bool) error {
	return o.InstallEksCtlWithVersion("", skipPathScan)
}

// InstallEksCtlWithVersion install a specific version of eks cli
func (o *CommonOptions) InstallEksCtlWithVersion(version string, skipPathScan bool) error {
	return o.InstallOrUpdateBinary(InstallOrUpdateBinaryOptions{
		Binary:              "eksctl",
		GitHubOrganization:  "weaveworks",
		DownloadUrlTemplate: "https://github.com/weaveworks/eksctl/releases/download/{{.version}}/eksctl_{{.osTitle}}_{{.arch}}.{{.extension}}",
		Version:             version,
		SkipPathScan:        skipPathScan,
		VersionExtractor:    nil,
		Archived:            true,
	})
}

// InstallAwsIamAuthenticator install iam authenticator for AWS
func (o *CommonOptions) InstallAwsIamAuthenticator(skipPathScan bool) error {
	return o.InstallAwsIamAuthenticatorWithVersion(packages.IamAuthenticatorAwsVersion, skipPathScan)
}

// InstallAwsIamAuthenticatorWithVersion install a specific version of iam authenticator for AWS
func (o *CommonOptions) InstallAwsIamAuthenticatorWithVersion(version string, skipPathScan bool) error {
	return o.InstallOrUpdateBinary(InstallOrUpdateBinaryOptions{
		Binary:              "aws-iam-authenticator",
		GitHubOrganization:  "",
		DownloadUrlTemplate: "https://amazon-eks.s3-us-west-2.amazonaws.com/{{.version}}/2019-03-27/bin/{{.os}}/{{.arch}}/aws-iam-authenticator",
		Version:             version,
		SkipPathScan:        skipPathScan,
		VersionExtractor:    nil,
	})
}

// GetCloudProvider returns the cloud provider
func (o *CommonOptions) GetCloudProvider(p string) (string, error) {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	if p == "" {
		// lets detect Minikube
		currentContext, err := o.GetCommandOutput("", "kubectl", "config", "current-context")
		if err == nil && currentContext == "minikube" {
			p = cloud.MINIKUBE
		}
	}
	if p != "" {
		if !util.Contains(cloud.KubernetesProviders, p) {
			return "", util.InvalidArg(p, cloud.KubernetesProviders)
		}
	}

	if p == "" {
		prompt := &survey.Select{
			Message: "Cloud Provider",
			Options: cloud.KubernetesProviders,
			Default: cloud.MINIKUBE,
			Help:    "Cloud service providing the Kubernetes cluster, local VM (Minikube), Google (GKE), Oracle (OKE), Azure (AKS)",
		}

		survey.AskOne(prompt, &p, nil, surveyOpts)
	}
	return p, nil
}

// GetClusterDependencies returns the dependencies for a cloud provider
func (o *CommonOptions) GetClusterDependencies(depsToInstall []string) []string {
	deps := o.FilterInstalledDependencies(depsToInstall)
	d := BinaryShouldBeInstalled("kubectl")
	if d != "" && util.StringArrayIndex(deps, d) < 0 {
		deps = append(deps, d)
	}

	d = BinaryShouldBeInstalled("helm")
	if d != "" && util.StringArrayIndex(deps, d) < 0 {
		deps = append(deps, d)
	}

	// Platform specific deps
	if runtime.GOOS == "darwin" {
		if !o.NoBrew {
			d = BinaryShouldBeInstalled("brew")
			if d != "" && util.StringArrayIndex(deps, d) < 0 {
				deps = append(deps, d)
			}
		}
	}
	return deps
}

// FilterInstalledDependencies filters installed dependencies
func (o *CommonOptions) FilterInstalledDependencies(deps []string) []string {
	depsToInstall := []string{}
	for _, d := range deps {
		binary := BinaryShouldBeInstalled(d)
		if binary != "" {
			depsToInstall = append(depsToInstall, binary)
		}
	}
	return depsToInstall
}

// InstallMissingDependencies installs missing dependencies
func (o *CommonOptions) InstallMissingDependencies(providerSpecificDeps []string) error {
	deps := o.GetClusterDependencies(providerSpecificDeps)
	if len(deps) == 0 {
		return nil
	}

	install := []string{}

	if o.InstallDependencies {
		install = append(install, deps...)
	} else {
		surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
		if o.BatchMode {
			return errors.New(fmt.Sprintf("run without batch mode or manually install missing dependencies %v\n", deps))
		}

		prompt := &survey.MultiSelect{
			Message: "Missing required dependencies, deselect to avoid auto installing:",
			Options: deps,
			Default: deps,
		}
		survey.AskOne(prompt, &install, nil, surveyOpts)
	}

	return o.DoInstallMissingDependencies(install)
}

// InstallRequirements installs any requirements for the given provider kind
func (o *CommonOptions) InstallRequirements(cloudProvider string, extraDependencies ...string) error {
	var deps []string
	switch cloudProvider {
	case cloud.IKS:
		deps = o.AddRequiredBinary("ibmcloud", deps)
	case cloud.AWS:
		deps = o.AddRequiredBinary("kops", deps)
	case cloud.EKS:
		deps = o.AddRequiredBinary("eksctl", deps)
		deps = o.AddRequiredBinary("aws-iam-authenticator", deps)
	case cloud.AKS:
		deps = o.AddRequiredBinary("az", deps)
	case cloud.GKE:
		deps = o.AddRequiredBinary("gcloud", deps)
	case cloud.OKE:
		deps = o.AddRequiredBinary("oci", deps)
	case cloud.MINIKUBE:
		deps = o.AddRequiredBinary("minikube", deps)
	}

	for _, dep := range extraDependencies {
		deps = o.AddRequiredBinary(dep, deps)
	}

	return o.InstallMissingDependencies(deps)
}

// AddRequiredBinary add the required binary
func (o *CommonOptions) AddRequiredBinary(binName string, deps []string) []string {
	d := BinaryShouldBeInstalled(binName)
	if d != "" && util.StringArrayIndex(deps, d) < 0 {
		deps = append(deps, d)
	}
	return deps
}

// CreateClusterAdmin creates a cluster admin
func (o *CommonOptions) CreateClusterAdmin() error {

	content := []byte(
		`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  name: cluster-admin
  annotations:
    rbac.authorization.kubernetes.io/autoupdate: "true"
rules:
- apiGroups:
  - '*'
  resources:
  - '*'
  verbs:
  - '*'
- nonResourceURLs:
  - '*'
  verbs:
  - '*'`)

	fileName := randomdata.SillyName() + ".yml"
	fileName = filepath.Join(os.TempDir(), fileName)
	tmpfile, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}

	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(content); err != nil {
		return err
	}
	if err := tmpfile.Close(); err != nil {
		return err
	}

	_, err1 := o.GetCommandOutput("", "kubectl", "create", "clusterrolebinding", "kube-system-cluster-admin", "--clusterrole", "cluster-admin", "--serviceaccount", "kube-system:default")
	if err1 != nil {
		if strings.Contains(err1.Error(), "AlreadyExists") {
			log.Logger().Info("role cluster-admin already exists for the cluster")
		} else {
			return err1
		}
	}

	_, err2 := o.GetCommandOutput("", "kubectl", "create", "-f", tmpfile.Name())
	if err2 != nil {
		if strings.Contains(err2.Error(), "AlreadyExists") {
			log.Logger().Info("clusterroles.rbac.authorization.k8s.io 'cluster-admin' already exists")
		} else {
			return err2
		}
	}

	return nil
}

// GetClusterUserName returns cluster and user name
func (o *CommonOptions) GetClusterUserName() (string, error) {
	username, _ := o.GetCommandOutput("", "gcloud", "config", "get-value", "core/account")

	if username != "" {
		return GetSafeUsername(username), nil
	}

	config, _, err := o.Kube().LoadConfig()
	if err != nil {
		return username, errors.Wrap(err, "loading kube config")
	}
	if config == nil || config.Contexts == nil || len(config.Contexts) == 0 {
		return username, fmt.Errorf("No Kubernetes contexts available! Try create or connect to cluster?")
	}
	contextName := config.CurrentContext
	if contextName == "" {
		return username, fmt.Errorf("No Kubernetes context selected. Please select one (e.g. via jx context) first")
	}
	context := config.Contexts[contextName]
	if context == nil {
		return username, fmt.Errorf("No Kubernetes context available for context %s", contextName)
	}
	username = context.AuthInfo

	return username, nil
}

// GetSafeUsername returns username by checking the active configuration
func GetSafeUsername(username string) string {
	if strings.Contains(username, "Your active configuration is") {
		return strings.Split(username, "\n")[1]
	}
	return username
}

// InstallProw installs prow
func (o *CommonOptions) InstallProw(useTekton bool, useExternalDNS bool, isGitOps bool, gitOpsDir string, gitOpsEnvDir string, gitUsername string, valuesFiles []string) error {
	if o.ReleaseName == "" {
		o.ReleaseName = kube.DefaultProwReleaseName
	}

	if o.Chart == "" {
		o.Chart = kube.ChartProw
	}

	var err error
	if o.HMACToken == "" {
		// why 41?  seems all examples so far have a random token of 41 chars
		o.HMACToken, err = util.RandStringBytesMaskImprSrc(41)
		if err != nil {
			return fmt.Errorf("cannot create a random hmac token for Prow")
		}
	}

	if o.OAUTHToken == "" {
		authConfigSvc, err := o.CreateGitAuthConfigService()
		if err != nil {
			return errors.Wrap(err, "creating git auth config svc")
		}

		config := authConfigSvc.Config()
		// lets assume github.com for now so ignore config.CurrentServer
		server := config.GetOrCreateServer("https://github.com")
		message := fmt.Sprintf("%s bot user for CI/CD pipelines (not your personal Git user):", server.Label())
		userAuth, err := config.PickServerUserAuth(server, message, o.BatchMode, "", o.In, o.Out, o.Err)
		if err != nil {
			return errors.Wrap(err, "picking bot user auth")
		}
		o.OAUTHToken = userAuth.ApiToken
	}

	if o.Username == "" {
		o.Username, err = o.GetClusterUserName()
		if err != nil {
			return errors.Wrap(err, "retrieving the cluster user name")
		}
	}
	if gitUsername == "" {
		gitUsername = o.Username
	}

	client, devNamespace, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return errors.Wrap(err, "creating kube client")
	}

	setValues := strings.Split(o.SetValues, ",")

	settings, err := o.TeamSettings()
	if err != nil {
		return errors.Wrap(err, "reading the team settings")
	}

	log.Logger().Infof("Setting up prow config into namespace %s", util.ColorInfo(devNamespace))

	// create initial configmaps if they don't already exist, use a dummy repo so tide doesn't start scanning all github
	_, err = client.CoreV1().ConfigMaps(devNamespace).Get("config", metav1.GetOptions{})
	if err != nil {
		err = prow.AddApplication(client, []string{"jenkins-x/dummy"}, devNamespace, "base", settings)
		if err != nil {
			return errors.Wrap(err, "adding dummy application")
		}
	}

	knativeOrTekton := "tekton"
	if !useTekton {
		knativeOrTekton = "knative"
	}
	log.Logger().Infof("Installing %s into namespace %s", knativeOrTekton, util.ColorInfo(devNamespace))

	ksecretValues := []string{}
	if settings.HelmTemplate || settings.NoTiller || settings.HelmBinary != "helm" {
		// lets disable tiller
		setValues = append(setValues, "tillerNamespace=")
	}

	prowVersion := o.Version

	if useTekton {
		setValues = append(setValues,
			"auth.git.username="+gitUsername)

		ksecretValues = append(ksecretValues,
			"auth.git.password="+o.OAUTHToken)

		err = o.Retry(2, time.Second, func() (err error) {
			return o.InstallChartOrGitOps(isGitOps, gitOpsDir, gitOpsEnvDir, kube.DefaultTektonReleaseName,
				kube.ChartTekton, "tekton", "", devNamespace, true, setValues, ksecretValues, valuesFiles, "")
		})
		if err != nil {
			return errors.Wrap(err, "failed to install Tekton")
		}

		setValues = append(setValues,
			"buildnum.enabled=false",
			"build.enabled=false",
			"pipelinerunner.enabled=true",
		)

	} else {
		setValues = append(setValues, "build.auth.git.username="+gitUsername)
		ksecretValues = append(ksecretValues, "build.auth.git.username="+gitUsername, "build.auth.git.password="+o.OAUTHToken)
		err = o.Retry(2, time.Second, func() (err error) {
			return o.InstallChartOrGitOps(isGitOps, gitOpsDir, gitOpsEnvDir, kube.DefaultKnativeBuildReleaseName,
				kube.ChartKnativeBuild, "knativebuild", "", devNamespace, true, setValues, ksecretValues, valuesFiles, "")
		})
		if err != nil {
			return errors.Wrap(err, "failed to install Knative build")
		}

		// lets use the stable knative build prow
		if prowVersion == "" {
			prowVersion, err = o.GetVersionNumber(version.KindChart, "jenkins-x/prow-knative", "", "")
			if err != nil {
				return errors.Wrap(err, "failed to find Prow Knative build version")
			}
		}
	}

	if useExternalDNS && strings.Contains(o.Domain, "nip.io") {
		log.Logger().Warnf("Skipping install of External DNS, %s domain is not supported while using External DNS", util.ColorInfo(o.Domain))
		log.Logger().Warnf("External DNS only supports the use of personally operated domains")
	} else if useExternalDNS && o.Domain != "" {
		log.Logger().Infof("Preparing to install ExternalDNS into namespace %s", util.ColorInfo(devNamespace))
		log.Logger().Infof("External DNS for Jenkins X is currently only supoorted on GKE")

		err = o.installExternalDNSGKE()
		if err != nil {
			return errors.Wrap(err, "failed to install external-dns")
		}
	}

	log.Logger().Infof("Installing Prow into namespace %s", util.ColorInfo(devNamespace))

	for _, value := range valuesFiles {
		log.Logger().Infof("with values file %s", util.ColorInfo(value))
	}

	secretValues := []string{"user=" + gitUsername, "oauthToken=" + o.OAUTHToken, "hmacToken=" + o.HMACToken}
	err = o.Retry(2, time.Second, func() (err error) {
		return o.InstallChartOrGitOps(isGitOps, gitOpsDir, gitOpsEnvDir, o.ReleaseName,
			o.Chart, "prow", prowVersion, devNamespace, true, setValues, secretValues, valuesFiles, "")
	})
	if err != nil {
		return errors.Wrap(err, "failed to install Prow")
	}

	if !useTekton {
		log.Logger().Infof("\nInstalling BuildTemplates into namespace %s", util.ColorInfo(devNamespace))
		err = o.Retry(2, time.Second, func() (err error) {
			return o.InstallChartOrGitOps(isGitOps, gitOpsDir, gitOpsEnvDir, kube.DefaultBuildTemplatesReleaseName,
				kube.ChartBuildTemplates, "jxbuildtemplates", "", devNamespace, true, nil, nil, nil, "")
		})
		if err != nil {
			return errors.Wrap(err, "failed to install JX Build Templates")
		}
	}
	return nil
}

// CreateWebhookProw create a webhook for prow using the given git provider
func (o *CommonOptions) CreateWebhookProw(gitURL string, gitProvider gits.GitProvider) error {
	client, err := o.KubeClient()
	if err != nil {
		return err
	}
	ns, _, err := kube.GetDevNamespace(client, o.currentNamespace)
	if err != nil {
		return err
	}
	gitInfo, err := gits.ParseGitURL(gitURL)
	if err != nil {
		return err
	}
	baseURL, err := services.FindServiceURL(client, ns, "hook")
	if err != nil {
		return errors.Wrapf(err, "in namespace %s", ns)
	}
	if baseURL == "" {
		return fmt.Errorf("failed to find external URL of service hook in namespace %s", ns)
	}
	webhookUrl := util.UrlJoin(baseURL, "hook")

	hmacToken, err := client.CoreV1().Secrets(ns).Get("hmac-token", metav1.GetOptions{})
	if err != nil {
		return err
	}
	webhook := &gits.GitWebHookArguments{
		Owner:  gitInfo.Organisation,
		Repo:   gitInfo,
		URL:    webhookUrl,
		Secret: string(hmacToken.Data["hmac"]),
	}
	return gitProvider.CreateWebHook(webhook)
}

// IsProw checks if prow is available in the cluster
func (o *CommonOptions) IsProw() (bool, error) {
	ns := o.devNamespace
	jxClient, devNs, err := o.JXClientAndDevNamespace()
	if err != nil {
		return false, err
	}
	if ns == "" {
		ns = devNs
	}
	env, err := kube.GetEnvironment(jxClient, ns, "dev")
	if err != nil {
		return false, err
	}

	return env.Spec.TeamSettings.PromotionEngine == jenkinsv1.PromotionEngineProw, nil
}

func (o *CommonOptions) installExternalDNSGKE() error {

	if o.ReleaseName == "" {
		o.ReleaseName = kube.DefaultExternalDNSReleaseName
	}

	if o.Chart == "" {
		o.Chart = kube.ChartExternalDNS
	}

	var err error

	client, err := o.KubeClient()
	if err != nil {
		return err
	}

	devNamespace, _, err := kube.GetDevNamespace(client, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}

	clusterName, err := cluster.Name(o.Kube())
	if err != nil {
		return errors.Wrap(err, "failed to get clusterName")
	}

	err = o.helm.AddRepo(kube.ChartOwnerExternalDNS, kube.ChartURLExternalDNS, "", "")
	if err != nil {
		return errors.Wrapf(err, "adding helm repo")
	}

	googleProjectID, err := gke.GetCurrentProject()
	if err != nil {
		return errors.Wrap(err, "failed to get project")
	}

	// Create managed zone for external dns if it doesn't exist
	var nameServers = []string{}
	err = gke.CreateManagedZone(googleProjectID, o.Domain)
	if err != nil {
		return errors.Wrap(err, "while trying to creating a CloudDNS managed zone for external-dns")
	}
	_, nameServers, err = gke.GetManagedZoneNameServers(googleProjectID, o.Domain)
	if err != nil {
		return errors.Wrap(err, "while trying to retrieve the managed zone name servers for external-dns")
	}

	o.NameServers = nameServers

	var gcpServiceAccountSecretName string
	gcpServiceAccountSecretName, err = externaldns.CreateExternalDNSGCPServiceAccount(client,
		kube.DefaultExternalDNSReleaseName, devNamespace, clusterName, googleProjectID)
	if err != nil {
		return errors.Wrap(err, "failed to create service account for ExternalDNS")
	}

	err = gke.EnableAPIs(googleProjectID, "dns")
	if err != nil {
		return errors.Wrap(err, "unable to enable 'dns' api")
	}

	sources := []string{
		"ingress",
	}

	sourcesList := "{" + strings.Join(sources, ", ") + "}"

	values := []string{
		"provider=" + "google",
		"sources=" + sourcesList,
		"rbac.create=" + "true",
		"google.serviceAccountSecret=" + gcpServiceAccountSecretName,
		"txt-owner-id=" + "jx-external-dns",
		"domainFilters=" + "{" + o.Domain + "}",
	}

	log.Logger().Infof("\nInstalling External DNS into namespace %s", util.ColorInfo(devNamespace))
	err = o.Retry(2, time.Second, func() (err error) {
		return o.InstallChartOrGitOps(false, "", "", kube.DefaultExternalDNSReleaseName, kube.ChartExternalDNS,
			kube.ChartExternalDNS, "", devNamespace, true, values, nil, nil, "")
	})
	if err != nil {
		return errors.Wrap(err, "failed to install External DNS")
	}

	return nil
}

func (o *CommonOptions) InstallIBMCloud(skipPathScan bool) error {
	return o.InstallIBMCloudWithVersion(packages.IBMCloudVersion, skipPathScan)
}

// InstallIBMCloudWithVersion  installs a specific version of IBM cloud CLI
func (o *CommonOptions) InstallIBMCloudWithVersion(version string, skipPathScan bool) error {
	if runtime.GOOS == "darwin" {
		return o.InstallOrUpdateBinary(InstallOrUpdateBinaryOptions{
			Binary:              "ibmcloud",
			GitHubOrganization:  "",
			DownloadUrlTemplate: "https://public.dhe.ibm.com/cloud/bluemix/cli/bluemix-cli/{{.version}}/binaries/IBM_Cloud_CLI_{{.version}}_macos.tgz",
			Version:             version,
			SkipPathScan:        skipPathScan,
			VersionExtractor:    nil,
			Archived:            true,
			ArchiveDirectory:    "IBM_Cloud_CLI",
		})
	}
	return o.InstallOrUpdateBinary(InstallOrUpdateBinaryOptions{
		Binary:              "ibmcloud",
		GitHubOrganization:  "",
		DownloadUrlTemplate: "https://public.dhe.ibm.com/cloud/bluemix/cli/bluemix-cli/{{.version}}/binaries/IBM_Cloud_CLI_{{.version}}_{{.os}}_{{.arch}}.{{.extension}}",
		Version:             version,
		SkipPathScan:        skipPathScan,
		VersionExtractor:    nil,
		Archived:            true,
		ArchiveDirectory:    "IBM_Cloud_CLI",
	})
}
