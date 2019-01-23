package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/jenkins-x/jx/pkg/kube/services"

	"github.com/jenkins-x/jx/pkg/binaries"

	"gopkg.in/yaml.v2"

	"github.com/Pallinder/go-randomdata"
	"github.com/alexflint/go-filemutex"
	"github.com/blang/semver"
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/maven"
	"github.com/jenkins-x/jx/pkg/prow"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/AlecAivazis/survey.v1"
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

type Prow struct {
	Version     string
	Chart       string
	SetValues   string
	ReleaseName string
	HMACToken   string
	OAUTHToken  string
}

func (o *CommonOptions) doInstallMissingDependencies(install []string) error {
	// install package managers first
	for _, i := range install {
		if i == "brew" {
			log.Infof("Installing %s\n", util.ColorInfo(i))
			o.installBrew()
			break
		}
	}

	for _, i := range install {
		log.Infof("Installing %s\n", util.ColorInfo(i))
		var err error
		switch i {
		case "az":
			err = o.installAzureCli()
		case "kubectl":
			err = o.installKubectl()
		case "gcloud":
			err = o.installGcloud()
		case "helm":
			err = o.installHelm()
		case "ibmcloud":
			err = o.installIBMCloud(false)
		case "tiller":
			err = o.installTiller()
		case "helm3":
			err = o.installHelm3()
		case "hyperkit":
			err = o.installHyperkit()
		case "kops":
			err = o.installKops()
		case "kvm":
			err = o.installKvm()
		case "kvm2":
			err = o.installKvm2()
		case "ksync":
			_, err = o.installKSync()
		case "minikube":
			err = o.installMinikube()
		case "minishift":
			err = o.installMinishift()
		case "oc":
			err = o.installOc()
		case "virtualbox":
			err = o.installVirtualBox()
		case "xhyve":
			err = o.installXhyve()
		case "hyperv":
			err = o.installhyperv()
		case "terraform":
			err = o.installTerraform()
		case "oci":
			err = o.installOciCli()
		case "aws":
			err = o.installAws()
		case "eksctl":
			err = o.installEksCtl(false)
		case "heptio-authenticator-aws":
			err = o.installHeptioAuthenticatorAws(false)
		case "kustomize":
			err = o.installKustomize()
		default:
			return fmt.Errorf("unknown dependency to install %s\n", i)
		}
		if err != nil {
			return fmt.Errorf("error installing %s: %v\n", i, err)
		}
	}
	return nil
}

// appends the binary to the deps array if it cannot be found on the $PATH
func binaryShouldBeInstalled(d string) string {
	_, shouldInstall, err := shouldInstallBinary(d)
	if err != nil {
		log.Warnf("Error detecting if binary should be installed: %s", err.Error())
		return ""
	}
	if shouldInstall {
		return d
	}
	return ""
}

func (o *CommonOptions) installBrew() error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	log.Infof("Please enter your root password when prompted by the %s installation\n", util.ColorInfo("brew"))
	//Make sure to run command through sh in order to get $() expanded.
	return o.RunCommand("sh", "-c", "/usr/bin/ruby -e \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)\"")
}

func shouldInstallBinary(name string) (fileName string, download bool, err error) {
	fileName = binaries.BinaryWithExtension(name)
	download = false
	pgmPath, err := exec.LookPath(fileName)
	if err == nil {
		logger.Debugf("%s is already available on your PATH at %s", util.ColorInfo(fileName), util.ColorInfo(pgmPath))
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
		logger.Debugf("Please add %s to your PATH", util.ColorInfo(binDir))
		return
	}
	download = true
	return
}

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

type InstallOrUpdateBinaryOptions struct {
	Binary              string
	GitHubOrganization  string
	DownloadUrlTemplate string
	Version             string
	SkipPathScan        bool
	VersionExtractor    binaries.VersionExtractor
	Archived            bool
	ArchiveDirectory    string
}

func (o *CommonOptions) installOrUpdateBinary(options InstallOrUpdateBinaryOptions) error {
	shouldInstall, err := binaries.ShouldInstallBinary(options.Binary, options.Version, options.VersionExtractor)
	if err != nil {
		return err
	}
	if !shouldInstall {
		return nil
	}

	configDir, err := util.ConfigDir()
	if err != nil {
		return err
	}
	binariesConfiguration := filepath.Join(configDir, "/binaries.yml")
	binariesVersions := map[string]string{}
	if _, err := os.Stat(binariesConfiguration); err == nil {
		binariesBytes, err := ioutil.ReadFile(binariesConfiguration)
		if err != nil {
			return err
		}
		yaml.Unmarshal(binariesBytes, &binariesVersions)
		if binariesVersions[options.Binary] == options.Version {
			return nil
		}
	}

	urlTemplate, err := template.New(options.Binary).Parse(options.DownloadUrlTemplate)
	if err != nil {
		return err
	}
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	fileName := options.Binary
	if !options.SkipPathScan {
		installFilename, flag, err := shouldInstallBinary(options.Binary)
		fileName = installFilename
		if err != nil || !flag {
			return err
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
	err = binaries.DownloadFile(clientUrlBuffer.String(), tarFile)
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
			err = os.MkdirAll(zipDir, DefaultWritePermissions)
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

	binariesVersions[options.Binary] = options.Version
	binariesBytes, err := yaml.Marshal(binariesVersions)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(binariesConfiguration, binariesBytes, 0644)
	if err != nil {
		return err
	}

	return os.Chmod(fullPath, 0755)
}

func (o *CommonOptions) installBrewIfRequired() error {
	if runtime.GOOS != "darwin" || o.NoBrew {
		return nil
	}

	_, flag, err := shouldInstallBinary("brew")
	if err != nil || !flag {
		return err
	}
	return o.installBrew()
}

func (o *CommonOptions) installKubectl() error {
	if runtime.GOOS == "darwin" && !o.NoBrew {
		return o.RunCommand("brew", "install", "kubectl")
	}
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	fileName, flag, err := shouldInstallBinary("kubectl")
	if err != nil || !flag {
		return err
	}
	kubernetes := "kubernetes"
	latestVersion, err := o.getLatestVersionFromKubernetesReleaseUrl()
	if err != nil {
		return fmt.Errorf("Unable to get latest version for github.com/%s/%s %v", kubernetes, kubernetes, err)
	}

	clientURL := fmt.Sprintf("https://storage.googleapis.com/kubernetes-release/release/v%s/bin/%s/%s/%s", latestVersion, runtime.GOOS, runtime.GOARCH, fileName)
	fullPath := filepath.Join(binDir, fileName)
	tmpFile := fullPath + ".tmp"
	err = binaries.DownloadFile(clientURL, tmpFile)
	if err != nil {
		return err
	}
	err = util.RenameFile(tmpFile, fullPath)
	if err != nil {
		return err
	}
	return os.Chmod(fullPath, 0755)
}

func (o *CommonOptions) installKustomize() error {
	if runtime.GOOS == "darwin" && !o.NoBrew {
		return o.RunCommand("brew", "install", "kustomize")
	}
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	fileName, flag, err := shouldInstallBinary("kustomize")
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
	err = binaries.DownloadFile(clientURL, tmpFile)
	if err != nil {
		return err
	}
	err = util.RenameFile(tmpFile, fullPath)
	if err != nil {
		return err
	}
	return os.Chmod(fullPath, 0755)
}

func (o *CommonOptions) installOc() error {
	// need to fix the version we download as not able to work out the oc sha in the URL yet
	sha := "191fece"
	latestVersion := "3.9.0"

	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	binary := "oc"
	fileName, flag, err := shouldInstallBinary(binary)
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
	err = binaries.DownloadFile(clientURL, tarFile)
	if err != nil {
		return err
	}

	if extension == ".zip" {
		zipDir := filepath.Join(binDir, "oc-tmp-"+uuid.NewUUID().String())
		err = os.MkdirAll(zipDir, DefaultWritePermissions)
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

// get the latest version from kubernetes, parse it and return it
func (o *CommonOptions) getLatestVersionFromKubernetesReleaseUrl() (sem semver.Version, err error) {
	response, err := util.GetClient().Get(stableKubeCtlVersionURL)

	if err != nil {
		return semver.Version{}, fmt.Errorf("Cannot get url " + stableKubeCtlVersionURL)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return semver.Version{}, fmt.Errorf("download of %s failed with return code %d", stableKubeCtlVersionURL, response.StatusCode)
	}

	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return semver.Version{}, fmt.Errorf("Cannot get url body")
	}

	s := strings.TrimSpace(string(bytes))
	if s != "" {
		return semver.Make(strings.TrimPrefix(s, "v"))
	}

	return semver.Version{}, fmt.Errorf("Cannot get release name")
}

func (o *CommonOptions) installHyperkit() error {
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

		log.Warn("Installing hyperkit does require sudo to perform some actions, for more details see https://github.com/kubernetes/minikube/blob/master/docs/drivers.md#hyperkit-driver")

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

func (o *CommonOptions) installKvm() error {
	log.Warnf("We cannot yet automate the installation of KVM - can you install this manually please?\nPlease see: https://www.linux-kvm.org/page/Downloads\n")
	return nil
}

func (o *CommonOptions) installKvm2() error {
	log.Warnf("We cannot yet automate the installation of KVM with KVM2 driver - can you install this manually please?\nPlease see: https://www.linux-kvm.org/page/Downloads " +
		"and https://github.com/kubernetes/minikube/blob/master/docs/drivers.md#kvm2-driver\n")
	return nil
}

func (o *CommonOptions) installVirtualBox() error {
	log.Warnf("We cannot yet automate the installation of VirtualBox - can you install this manually please?\nPlease see: https://www.virtualbox.org/wiki/Downloads\n")
	return nil
}

func (o *CommonOptions) installXhyve() error {
	info, err := o.getCommandOutput("", "brew", "info", "docker-machine-driver-xhyve")

	if err != nil || strings.Contains(info, "Not installed") {
		err = o.RunCommand("brew", "install", "docker-machine-driver-xhyve")
		if err != nil {
			return err
		}

		brewPrefix, err := o.getCommandOutput("", "brew", "--prefix")
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
		log.Infoln("xhyve driver installed")
	} else {
		pgmPath, _ := exec.LookPath("docker-machine-driver-xhyve")
		log.Infof("xhyve driver is already available on your PATH at %s\n", pgmPath)
	}
	return nil
}

func (o *CommonOptions) installhyperv() error {
	info, err := o.getCommandOutput("", "powershell", "Get-WindowsOptionalFeature", "-FeatureName", "Microsoft-Hyper-V-All", "-Online")

	if err != nil {
		return err
	}
	if strings.Contains(info, "Disabled") {

		log.Info("hyperv is Disabled, this computer will need to restart\n and after restart you will need to rerun your inputted commmand.")

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
		log.Infoln("hyperv is already Enabled")
	}
	return nil
}

func (o *CommonOptions) installVaultCli() error {
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	binary := "vault"
	fileName, flag, err := shouldInstallBinary(binary)
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
	err = binaries.DownloadFile(clientURL, tarFile)
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

func (o *CommonOptions) installHelm() error {
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

	fileName, flag, err := shouldInstallBinary(binary)
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
	err = binaries.DownloadFile(clientURL, tarFile)
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

func (o *CommonOptions) installTiller() error {
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
	err = binaries.DownloadFile(clientURL, tarFile)
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
	err = startLocalTillerIfNotRunning()
	if err != nil {
		return err
	}
	return o.installHelmSecretsPlugin(helmFullPath, true)
}

func (o *CommonOptions) installHelm3() error {
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	binary := "helm3"
	fileName, flag, err := shouldInstallBinary(binary)
	if err != nil || !flag {
		return err
	}
	/*
	   latestVersion, err := util.GetLatestVersionFromGitHub("kubernetes", "helm")
	   	if err != nil {
	   		return err
	   	}
	*/
	/*
		latestVersion := "3"
		clientURL := fmt.Sprintf("https://storage.googleapis.com/kubernetes-helm/helm-dev-v%s-%s-%s.tar.gz", latestVersion, runtime.GOOS, runtime.GOARCH)
	*/
	// let use our patched version
	latestVersion := "untagged-93375777c6644a452a64"
	clientURL := fmt.Sprintf("https://github.com/jstrachan/helm/releases/download/%v/helm-%s-%s.tar.gz", latestVersion, runtime.GOOS, runtime.GOARCH)

	tmpDir := filepath.Join(binDir, "helm3.tmp")
	err = os.MkdirAll(tmpDir, DefaultWritePermissions)
	if err != nil {
		return err
	}
	fullPath := filepath.Join(binDir, binary)
	tarFile := filepath.Join(tmpDir, fileName+".tgz")
	err = binaries.DownloadFile(clientURL, tarFile)
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
	log.Infof("Installing %s\n", util.ColorInfo("helm secrets plugin"))
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

func (o *CommonOptions) installMavenIfRequired() error {
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

	log.Infof("Apache Maven is not installed so lets download: %s\n", util.ColorInfo(clientURL))

	mvnDir := filepath.Join(homeDir, "maven")
	mvnTmpDir := filepath.Join(homeDir, "maven-tmp")
	zipFile := filepath.Join(homeDir, "mvn.zip")

	err = os.MkdirAll(mvnDir, DefaultWritePermissions)
	if err != nil {
		m.Unlock()
		return err
	}

	log.Info("\ndownloadFile\n")
	err = binaries.DownloadFile(clientURL, zipFile)
	if err != nil {
		m.Unlock()
		return err
	}

	log.Info("\nutil.Unzip\n")
	err = util.Unzip(zipFile, mvnTmpDir)
	if err != nil {
		m.Unlock()
		return err
	}

	// lets find a directory inside the unzipped folder
	log.Info("\nReadDir\n")
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
			log.Infof("Apache Maven is installed at: %s\n", util.ColorInfo(mvnDir))
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

func (o *CommonOptions) installTerraform() error {
	if runtime.GOOS == "darwin" && !o.NoBrew {
		return o.RunCommand("brew", "install", "terraform")
	}

	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	binary := "terraform"
	fileName, flag, err := shouldInstallBinary(binary)
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
	err = binaries.DownloadFile(clientURL, zipFile)
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

func (o *CommonOptions) GetLatestJXVersion() (semver.Version, error) {
	return util.GetLatestVersionFromGitHub("jenkins-x", "jx")
}

func (o *CommonOptions) installKops() error {
	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	binary := "kops"
	fileName, flag, err := shouldInstallBinary(binary)
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
	err = binaries.DownloadFile(clientURL, tmpFile)
	if err != nil {
		return err
	}
	err = util.RenameFile(tmpFile, fullPath)
	if err != nil {
		return err
	}
	return os.Chmod(fullPath, 0755)
}

func (o *CommonOptions) installKSync() (string, error) {
	binDir, err := util.JXBinLocation()
	if err != nil {
		return "", err
	}
	binary := "ksync"
	fileName, flag, err := shouldInstallBinary(binary)
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
	err = binaries.DownloadFile(clientURL, tmpFile)
	if err != nil {
		return "", err
	}
	err = util.RenameFile(tmpFile, fullPath)
	if err != nil {
		return "", err
	}
	return latestVersion.String(), os.Chmod(fullPath, 0755)
}

func (o *CommonOptions) installJx(upgrade bool, version string) error {
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
		f, flag, err := shouldInstallBinary(binary)
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
	err = binaries.DownloadFile(clientURL, tmpArchiveFile)
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
			log.Infof("Skipping removal of old jx binary: %s\n", err)
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
	log.Infof("Jenkins X client has been installed into %s\n", util.ColorInfo(fullPath))
	return os.Chmod(fullPath, 0755)
}

func (o *CommonOptions) installMinikube() error {
	if runtime.GOOS == "darwin" && !o.NoBrew {
		return o.RunCommand("brew", "cask", "install", "minikube")
	}

	binDir, err := util.JXBinLocation()
	if err != nil {
		return err
	}
	fileName, flag, err := shouldInstallBinary("minikube")
	if err != nil || !flag {
		return err
	}
	latestVersion, err := util.GetLatestVersionFromGitHub("kubernetes", "minikube")
	if err != nil {
		return err
	}
	clientURL := fmt.Sprintf("https://github.com/kubernetes/minikube/releases/download/v%s/minikube-%s-%s", latestVersion, runtime.GOOS, runtime.GOARCH)
	fullPath := filepath.Join(binDir, fileName)
	tmpFile := fullPath + ".tmp"
	err = binaries.DownloadFile(clientURL, tmpFile)
	if err != nil {
		return err
	}
	err = util.RenameFile(tmpFile, fullPath)
	if err != nil {
		return err
	}
	return os.Chmod(fullPath, 0755)
}

func (o *CommonOptions) installMinishift() error {
	if runtime.GOOS == "darwin" && !o.NoBrew {
		return o.RunCommand("brew", "cask", "install", "minishift")
	}

	binDir, err := util.JXBinLocation()
	binary := "minishift"
	if err != nil {
		return err
	}
	fileName, flag, err := shouldInstallBinary(binary)
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
	err = binaries.DownloadFile(clientURL, tarFile)
	if err != nil {
		return err
	}
	err = util.UnTargz(tarFile, binDir, []string{binary, fileName})
	if err != nil {
		return err
	}
	return os.Chmod(fullPath, 0755)
}

func (o *CommonOptions) installGcloud() error {
	if runtime.GOOS != "darwin" || o.NoBrew {
		return errors.New("please install missing gcloud sdk - see https://cloud.google.com/sdk/downloads#interactive")
	}
	err := o.RunCommand("brew", "tap", "caskroom/cask")
	if err != nil {
		return err
	}

	return o.RunCommand("brew", "cask", "install", "google-cloud-sdk")
}

func (o *CommonOptions) installAzureCli() error {
	return o.RunCommand("brew", "install", "azure-cli")
}

func (o *CommonOptions) installOciCli() error {
	var err error
	filePath := "./install.sh"
	log.Info("Installing OCI CLI...\n")
	err = o.RunCommand("curl", "-LO", "https://raw.githubusercontent.com/oracle/oci-cli/master/scripts/install/install.sh")

	if err != nil {
		return err
	}
	os.Chmod(filePath, 0755)

	err = o.runCommandVerbose(filePath, "--accept-all-defaults")
	if err != nil {
		return err
	}

	return os.Remove(filePath)
}

func (o *CommonOptions) installAws() error {
	// TODO
	return nil
}

func (o *CommonOptions) installEksCtl(skipPathScan bool) error {
	return o.installEksCtlWithVersion(binaries.EksctlVersion, skipPathScan)
}

func (o *CommonOptions) installEksCtlWithVersion(version string, skipPathScan bool) error {
	return o.installOrUpdateBinary(InstallOrUpdateBinaryOptions{
		Binary:              "eksctl",
		GitHubOrganization:  "weaveworks",
		DownloadUrlTemplate: "https://github.com/weaveworks/eksctl/releases/download/{{.version}}/eksctl_{{.osTitle}}_{{.arch}}.{{.extension}}",
		Version:             version,
		SkipPathScan:        skipPathScan,
		VersionExtractor:    nil,
		Archived:            true,
	})
}

func (o *CommonOptions) installHeptioAuthenticatorAws(skipPathScan bool) error {
	return o.installHeptioAuthenticatorAwsWithVersion(binaries.HeptioAuthenticatorAwsVersion, skipPathScan)
}

func (o *CommonOptions) installHeptioAuthenticatorAwsWithVersion(version string, skipPathScan bool) error {
	return o.installOrUpdateBinary(InstallOrUpdateBinaryOptions{
		Binary:              "heptio-authenticator-aws",
		GitHubOrganization:  "",
		DownloadUrlTemplate: "https://amazon-eks.s3-us-west-2.amazonaws.com/{{.version}}/2018-06-05/bin/{{.os}}/{{.arch}}/heptio-authenticator-aws",
		Version:             version,
		SkipPathScan:        skipPathScan,
		VersionExtractor:    nil,
	})
}

func (o *CommonOptions) GetCloudProvider(p string) (string, error) {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	if p == "" {
		// lets detect Minikube
		currentContext, err := o.getCommandOutput("", "kubectl", "config", "current-context")
		if err == nil && currentContext == "minikube" {
			p = MINIKUBE
		}
	}
	if p != "" {
		if !util.Contains(KUBERNETES_PROVIDERS, p) {
			return "", util.InvalidArg(p, KUBERNETES_PROVIDERS)
		}
	}

	if p == "" {
		prompt := &survey.Select{
			Message: "Cloud Provider",
			Options: KUBERNETES_PROVIDERS,
			Default: MINIKUBE,
			Help:    "Cloud service providing the Kubernetes cluster, local VM (Minikube), Google (GKE), Oracle (OKE), Azure (AKS)",
		}

		survey.AskOne(prompt, &p, nil, surveyOpts)
	}
	return p, nil
}

func (o *CommonOptions) getClusterDependencies(depsToInstall []string) []string {
	deps := o.filterInstalledDependencies(depsToInstall)
	d := binaryShouldBeInstalled("kubectl")
	if d != "" && util.StringArrayIndex(deps, d) < 0 {
		deps = append(deps, d)
	}

	d = binaryShouldBeInstalled("helm")
	if d != "" && util.StringArrayIndex(deps, d) < 0 {
		deps = append(deps, d)
	}

	// Platform specific deps
	if runtime.GOOS == "darwin" {
		if !o.NoBrew {
			d = binaryShouldBeInstalled("brew")
			if d != "" && util.StringArrayIndex(deps, d) < 0 {
				deps = append(deps, d)
			}
		}
	}
	return deps
}

func (o *CommonOptions) filterInstalledDependencies(deps []string) []string {
	depsToInstall := []string{}
	for _, d := range deps {
		binary := binaryShouldBeInstalled(d)
		if binary != "" {
			depsToInstall = append(depsToInstall, binary)
		}
	}
	return depsToInstall
}

func (o *CommonOptions) installMissingDependencies(providerSpecificDeps []string) error {
	deps := o.getClusterDependencies(providerSpecificDeps)
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

	return o.doInstallMissingDependencies(install)
}

// installRequirements installs any requirements for the given provider kind
func (o *CommonOptions) installRequirements(cloudProvider string, extraDependencies ...string) error {
	var deps []string
	switch cloudProvider {
	case IKS:
		deps = o.addRequiredBinary("ibmcloud", deps)
	case AWS:
		deps = o.addRequiredBinary("kops", deps)
	case EKS:
		deps = o.addRequiredBinary("eksctl", deps)
		deps = o.addRequiredBinary("heptio-authenticator-aws", deps)
	case AKS:
		deps = o.addRequiredBinary("az", deps)
	case GKE:
		deps = o.addRequiredBinary("gcloud", deps)
	case OKE:
		deps = o.addRequiredBinary("oci", deps)
	case MINIKUBE:
		deps = o.addRequiredBinary("minikube", deps)
	}

	for _, dep := range extraDependencies {
		deps = o.addRequiredBinary(dep, deps)
	}

	return o.installMissingDependencies(deps)
}

func (o *CommonOptions) addRequiredBinary(binName string, deps []string) []string {
	d := binaryShouldBeInstalled(binName)
	if d != "" && util.StringArrayIndex(deps, d) < 0 {
		deps = append(deps, d)
	}
	return deps
}

func (o *CommonOptions) createClusterAdmin() error {

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

	_, err1 := o.getCommandOutput("", "kubectl", "create", "clusterrolebinding", "kube-system-cluster-admin", "--clusterrole", "cluster-admin", "--serviceaccount", "kube-system:default")
	if err1 != nil {
		if strings.Contains(err1.Error(), "AlreadyExists") {
			log.Success("role cluster-admin already exists for the cluster")
		} else {
			return err1
		}
	}

	_, err2 := o.getCommandOutput("", "kubectl", "create", "-f", tmpfile.Name())
	if err2 != nil {
		if strings.Contains(err2.Error(), "AlreadyExists") {
			log.Success("clusterroles.rbac.authorization.k8s.io 'cluster-admin' already exists")
		} else {
			return err2
		}
	}

	return nil
}

func (o *CommonOptions) updateJenkinsURL(namespaces []string) error {
	client, err := o.KubeClient()
	if err != nil {
		return err
	}
	// loop over each namespace and update the Jenkins URL if a Jenkins service is found
	for _, n := range namespaces {
		externalURL, err := services.GetServiceURLFromName(client, "jenkins", n)
		if err != nil {
			// skip namespace if no Jenkins service found
			continue
		}

		log.Infof("Updating Jenkins with new external URL details %s\n", externalURL)

		jenkins, err := o.CreateJenkinsClient(client, n, o.In, o.Out, o.Err)

		if err != nil {
			return err
		}

		data := url.Values{}
		data.Add("script", fmt.Sprintf(groovy, externalURL))

		err = jenkins.Post("/scriptText", data, nil)
	}

	return nil
}

func (o *CommonOptions) GetClusterUserName() (string, error) {

	username, _ := o.getCommandOutput("", "gcloud", "config", "get-value", "core/account")

	if username != "" {
		return GetSafeUsername(username), nil
	}

	config, _, err := o.Kube().LoadConfig()
	if err != nil {
		return username, err
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

func GetSafeUsername(username string) string {
	if strings.Contains(username, "Your active configuration is") {
		return strings.Split(username, "\n")[1]
	}
	return username
}

func (o *CommonOptions) installProw() error {

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
			return err
		}

		config := authConfigSvc.Config()
		// lets assume github.com for now so ignore config.CurrentServer
		server := config.GetOrCreateServer("https://github.com")
		message := fmt.Sprintf("%s bot user for CI/CD pipelines (not your personal Git user):", server.Label())
		userAuth, err := config.PickServerUserAuth(server, message, o.BatchMode, "", o.In, o.Out, o.Err)
		if err != nil {
			return err
		}
		o.OAUTHToken = userAuth.ApiToken
	}

	if o.Username == "" {
		o.Username, err = o.GetClusterUserName()
		if err != nil {
			return err
		}
	}

	client, err := o.KubeClient()
	if err != nil {
		return err
	}

	devNamespace, _, err := kube.GetDevNamespace(client, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}

	values := []string{"user=" + o.Username, "oauthToken=" + o.OAUTHToken, "hmacToken=" + o.HMACToken}
	setValues := strings.Split(o.SetValues, ",")
	values = append(values, setValues...)

	// create initial configmaps if they don't already exist, use a dummy repo so tide doesn't start scanning all github
	_, err = client.CoreV1().ConfigMaps(devNamespace).Get("config", metav1.GetOptions{})
	if err != nil {
		err = prow.AddApplication(client, []string{"jenkins-x/dummy"}, devNamespace, "base")
		if err != nil {
			return err
		}
	}
	log.Infof("\nInstalling knative into namespace %s\n", util.ColorInfo(devNamespace))

	kvalues := []string{"build.auth.git.username=" + o.Username, "build.auth.git.password=" + o.OAUTHToken}
	kvalues = append(kvalues, setValues...)

	err = o.retry(2, time.Second, func() (err error) {
		return o.installChart(kube.DefaultKnativeBuildReleaseName, kube.ChartKnativeBuild, "", devNamespace, true,
			kvalues, nil, "")
	})

	if err != nil {
		return errors.Wrap(err, "failed to install Knative build")
	}

	log.Infof("\nInstalling Prow into namespace %s\n", util.ColorInfo(devNamespace))
	err = o.retry(2, time.Second, func() (err error) {
		return o.installChart(o.ReleaseName, o.Chart, o.Version, devNamespace, true, values, nil, "")
	})

	if err != nil {
		return errors.Wrap(err, "failed to install Prow")
	}

	log.Infof("\nInstalling BuildTemplates into namespace %s\n", util.ColorInfo(devNamespace))

	err = o.retry(2, time.Second, func() (err error) {
		return o.installChart(kube.DefaultBuildTemplatesReleaseName, kube.ChartBuildTemplates, "", devNamespace, true,
			values, nil, "")
	})

	if err != nil {
		return errors.Wrap(err, "failed to install JX Build Templates")
	}

	return nil
}

func (o *CommonOptions) createWebhookProw(gitURL string, gitProvider gits.GitProvider) error {
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
	baseURL, err := services.GetServiceURLFromName(client, "hook", ns)
	if err != nil {
		return err
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

func (o *CommonOptions) isProw() (bool, error) {
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

func (o *CommonOptions) installIBMCloud(skipPathScan bool) error {
	return o.installIBMCloudWithVersion(binaries.IBMCloudVersion, skipPathScan)
}

func (o *CommonOptions) installIBMCloudWithVersion(version string, skipPathScan bool) error {
	if runtime.GOOS == "darwin" {
		return o.installOrUpdateBinary(InstallOrUpdateBinaryOptions{
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
	return o.installOrUpdateBinary(InstallOrUpdateBinaryOptions{
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
