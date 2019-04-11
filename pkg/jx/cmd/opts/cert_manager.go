package opts

import (
	"time"

	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/kube/pki"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// EnsureCertManager ensures cert-manager is installed
func (o *CommonOptions) EnsureCertManager() error {
	logrus.Infof("Looking for %q deployment in namespace %q...\n", pki.CertManagerDeployment, pki.CertManagerNamespace)
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
			logrus.Info("Installing cert-manager...\n")
			logrus.Infof("Installing CRDs from %q...\n", pki.CertManagerCRDsFile)
			output, err := o.ResourcesInstaller().Install(pki.CertManagerCRDsFile)
			if err != nil {
				return errors.Wrapf(err, "installing the cert-manager CRDs from %q", pki.CertManagerCRDsFile)
			}
			logrus.Info(output + "\n")

			logrus.Infof("Installing the chart %q in namespace %q...\n", pki.CertManagerChart, pki.CertManagerNamespace)
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

			logrus.Info("Waiting for CertManager deployment to be ready, this can take a few minutes\n")

			err = kube.WaitForDeploymentToBeReady(client, pki.CertManagerDeployment, pki.CertManagerNamespace, 10*time.Minute)
			if err != nil {
				return errors.Wrapf(err, "waiting for %q deployment", pki.CertManagerDeployment)
			}
		}
	}
	return nil
}
