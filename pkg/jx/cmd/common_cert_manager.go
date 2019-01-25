package cmd

import (
	"fmt"
	"time"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

const certManagerChartVersion = "v0.5.2"

func (o *CommonOptions) ensureCertmanager() error {
	log.Infof("Looking for %s deployment in namespace %s...\n", CertManagerDeployment, CertManagerNamespace)
	client, err := o.KubeClient()
	if err != nil {
		return errors.Wrap(err, "creating kube client")
	}
	_, err = kube.GetDeploymentPods(client, CertManagerDeployment, CertManagerNamespace)
	if err != nil {
		ok := true
		if !o.BatchMode {
			ok = util.Confirm(
				"CertManager deployment not found, shall we install it now?",
				true,
				"CertManager automatically configures Ingress rules with TLS using signed certificates from LetsEncrypt",
				o.In, o.Out, o.Err)
		}
		if ok {

			values := []string{"rbac.create=true", "ingressShim.extraArgs='{--default-issuer-name=letsencrypt-staging,--default-issuer-kind=Issuer}'"}
			err = o.installChartOptions(helm.InstallChartOptions{
				ReleaseName: "cert-manager",
				Chart:       "stable/cert-manager",
				Version:     certManagerChartVersion,
				Ns:          CertManagerNamespace,
				HelmUpdate:  true,
				SetValues:   values,
			})
			if err != nil {
				return fmt.Errorf("CertManager deployment failed: %v", err)
			}

			log.Info("Waiting for CertManager deployment to be ready, this can take a few minutes\n")

			err = kube.WaitForDeploymentToBeReady(client, CertManagerDeployment, CertManagerNamespace, 10*time.Minute)
			if err != nil {
				return err
			}
		}
	}
	return err
}
