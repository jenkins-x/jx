package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jenkins-x/jx/pkg/binaries"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"k8s.io/client-go/kubernetes"
)

func (f *factory) GetHelm(verbose bool,
	helmBinary string,
	noTiller bool,
	helmTemplate bool,
	kubeClient kubernetes.Interface) helm.Helmer {

	featureFlag := "none"
	if helmTemplate {
		featureFlag = "template-mode"
	} else if noTiller {
		featureFlag = "no-tiller-server"
	}
	log.Infof("Using helmBinary %s with feature flag: %s\n", util.ColorInfo(helmBinary), util.ColorInfo(featureFlag))
	helmCLI := helm.NewHelmCLI(helmBinary, helm.V2, "", verbose)
	var h helm.Helmer
	h = helmCLI
	if helmTemplate {
		h = helm.NewHelmTemplate(helmCLI, "", kubeClient)
	}
	if noTiller {
		helmCLI.SetHost(tillerAddress())
		startLocalTillerIfNotRunning()
	}

	return h
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

// tillerAddress returns the address that tiller is listening on
func tillerAddress() string {
	tillerAddress := os.Getenv("TILLER_ADDR")
	if tillerAddress == "" {
		tillerAddress = ":44134"
	}
	return tillerAddress
}

func startLocalTillerIfNotRunning() error {
	return startLocalTiller(true)
}

func startLocalTiller(lazy bool) error {
	tillerAddress := getTillerAddress()
	tillerArgs := os.Getenv("TILLER_ARGS")
	args := []string{"-listen", tillerAddress, "-alsologtostderr"}
	if tillerArgs != "" {
		args = append(args, tillerArgs)
	}
	logsDir, err := util.LogsDir()
	if err != nil {
		return err
	}
	logFile := filepath.Join(logsDir, "tiller.log")
	f, err := os.Create(logFile)
	if err != nil {
		return errors.Wrapf(err, "Failed to create tiller log file %s: %s", logFile, err)
	}
	err = util.RunCommandBackground("tiller", f, !lazy, args...)
	if err == nil {
		log.Infof("running tiller locally and logging to file: %s\n", util.ColorInfo(logFile))
	} else if lazy {
		// lets assume its because the process is already running so lets ignore
		return nil
	}
	return err
}

func restartLocalTiller() error {
	log.Info("checking if we need to kill a local tiller process\n")
	util.KillProcesses("tiller")
	return startLocalTiller(false)
}

// tillerAddress returns the address that tiller is listening on
func getTillerAddress() string {
	tillerAddress := os.Getenv("TILLER_ADDR")
	if tillerAddress == "" {
		tillerAddress = ":44134"
	}
	return tillerAddress
}
