package extensions

import (
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jenkinsv1client "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/expose"
	"github.com/jenkins-x/jx/pkg/helm"
	"github.com/jenkins-x/jx/pkg/kube"
	certclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// OnApply examines the currently installs apps to perform any post-install actions
func OnApply(jxClient jenkinsv1client.Interface, kubeClient kubernetes.Interface, certClient certclient.Interface, ns string, helmer helm.Helmer,
	installTimeout string, versionsDir string) error {
	appList, err := jxClient.JenkinsV1().Apps(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, app := range appList.Items {
		err = OnInstall(&app, kubeClient, certClient, ns, helmer, installTimeout, versionsDir)
		if err != nil {
			return err
		}
	}
	return nil
}

// OnInstallFromName uses the App CRD installed by appName to perform any post-install actions.
func OnInstallFromName(appName string, jxClient jenkinsv1client.Interface, kubeClient kubernetes.Interface, certClient certclient.Interface,
	ns string, helmer helm.Helmer,
	installTimeout string, versionsDir string) error {
	app, err := jxClient.JenkinsV1().Apps(ns).Get(appName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	return OnInstall(app, kubeClient, certClient, ns, helmer, installTimeout, versionsDir)
}

// OnInstall uses the App CRD installed by appName to perform any post-install actions.
func OnInstall(app *jenkinsv1.App, kubeClient kubernetes.Interface, certClient certclient.Interface, ns string,
	helmer helm.Helmer, installTimeout string, versionsDir string) error {
	// Specific hooks go here
	err := exposeOnInstall(app, kubeClient, certClient, ns, helmer, installTimeout, versionsDir)
	if err != nil {
		return err
	}
	return nil
}

func exposeOnInstall(app *jenkinsv1.App, kubeClient kubernetes.Interface, certClient certclient.Interface, ns string,
	helmer helm.Helmer, installTimeout string, versionsDir string) error {
	for _, svc := range app.Spec.ExposedServices {
		err := exposeSvc(svc, kubeClient, certClient, ns, helmer, installTimeout, versionsDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func exposeSvc(svcName string, kubeClient kubernetes.Interface, certClient certclient.Interface, ns string,
	helmer helm.Helmer, installTimeout string, versionsDir string) error {
	svc, err := kubeClient.CoreV1().Services(ns).Get(svcName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "getting the addon service: %s", svc)
	}

	if svc.Annotations == nil {
		svc.Annotations = map[string]string{}
	}
	if svc.Annotations[kube.AnnotationExpose] == "" {
		svc.Annotations[kube.AnnotationExpose] = "true"
		svc, err = kubeClient.CoreV1().Services(ns).Update(svc)
		if err != nil {
			return errors.Wrap(err, "updating the service annotations")
		}
	}
	devNamespace, _, err := kube.GetDevNamespace(kubeClient, ns)
	if err != nil {
		return errors.Wrap(err, "retrieving the dev namespace")
	}
	return expose.Expose(kubeClient, certClient, devNamespace, ns, "", helmer, installTimeout, versionsDir)
}
