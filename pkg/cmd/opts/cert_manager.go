package opts

import (
	"time"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/kube/pki"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
)

// EnsureCertManager ensures cert-manager is installed
func (o *CommonOptions) EnsureCertManager() error {
	log.Logger().Infof("Looking for %q deployment in namespace %q...", pki.CertManagerDeployment, pki.CertManagerNamespace)
	client, err := o.KubeClient()
	if err != nil {
		return errors.Wrap(err, "creating kube client")
	}
	_, err = kube.GetDeploymentPods(client, pki.CertManagerDeployment, pki.CertManagerNamespace)
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
			log.Logger().Info("Installing cert-manager...")
			log.Logger().Infof("Installing CRDs from %q...", pki.CertManagerCRDsFile)
			output, err := o.ResourcesInstaller().Install(pki.CertManagerCRDsFile)
			if err != nil {
				return errors.Wrapf(err, "installing the cert-manager CRDs from %q", pki.CertManagerCRDsFile)
			}
			log.Logger().Info(output)

			log.Logger().Infof("Installing the chart %q in namespace %q...", pki.CertManagerChart, pki.CertManagerNamespace)
			values := []string{
				"rbac.create=true",
				"webhook.enabled=false",
				"ingressShim.defaultIssuerName=letsencrypt-staging",
				"ingressShim.defaultIssuerKind=Issuer"}

			err = o.InstallChartWithOptions(helm.InstallChartOptions{
				ReleaseName: pki.CertManagerReleaseName,
				Chart:       pki.CertManagerChart,
				Version:     "",
				Ns:          pki.CertManagerNamespace,
				HelmUpdate:  true,
				SetValues:   values,
			})
			if err != nil {
				return errors.Wrapf(err, "installing %q chart", pki.CertManagerChart)
			}

			log.Logger().Info("Waiting for CertManager deployment to be ready, this can take a few minutes")

			err = kube.WaitForDeploymentToBeReady(client, pki.CertManagerDeployment, pki.CertManagerNamespace, 10*time.Minute)
			if err != nil {
				return errors.Wrapf(err, "waiting for %q deployment", pki.CertManagerDeployment)
			}
		}
	}
	return nil
}
