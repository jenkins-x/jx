package cmd

import (
	"errors"
	"fmt"
	"github.com/Pallinder/go-randomdata"
	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/jx/cmd/certmanager"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
	"time"
)

var (
	upgradeIngressLong = templates.LongDesc(`
		Upgrades the Jenkins X Ingress rules
`)

	upgradeIngressExample = templates.Examples(`
		# Upgrades the Jenkins X Ingress rules
		jx upgrade ingress
	`)
)

const (
	CertManagerDeployment         = "cert-manager-cert-manager"
	CertManagerNamespace          = "cert-manager"
	CertmanagerCertificateProd    = "letsencrypt-prod"
	CertmanagerCertificateStaging = "letsencrypt-staging"
	CertmanagerIssuerProd         = "letsencrypt-prod"
	CertmanagerIssuerStaging      = "letsencrypt-staging"
)

// UpgradeIngressOptions the options for the create spring command
type UpgradeIngressOptions struct {
	CreateOptions

	SkipCertManager  bool
	Cluster          bool
	Namespaces       []string
	Version          string
	TargetNamespaces []string
	Email            string
	Domain           string
	Issuer           string
	tls              bool
}

// NewCmdUpgradeIngress defines the command
func NewCmdUpgradeIngress(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &UpgradeIngressOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "ingress",
		Short:   "Upgrades Ingress rules",
		Aliases: []string{"ing"},
		Long:    upgradeIngressLong,
		Example: upgradeIngressExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&options.Cluster, "cluster", "", false, "Enable cluster wide Ingress upgrade")
	cmd.Flags().StringArrayVarP(&options.Namespaces, "namespaces", "", []string{}, "Namespaces to upgrade")
	cmd.Flags().BoolVarP(&options.SkipCertManager, "skip-certmanager", "", false, "Skips certmanager installation")
	return cmd
}

// Run implements the command
func (o *UpgradeIngressOptions) Run() error {

	_, _, err := o.KubeClient()
	if err != nil {
		return fmt.Errorf("cannot connect to kubernetes cluster: %v", err)
	}

	// if existing ingress exist in the namespaces ask do you want to delete them?
	ingressToDelete, err := o.getExistingIngressRules()
	if err != nil {
		return err
	}

	// wizard to ask for config values
	exposecontrollerConfig, err := o.confirmExposecontrollerConfig()
	if err != nil {
		return err
	}

	// confirm values
	util.Confirm(fmt.Sprintf("Using exposecontroller config values %v, ok?", exposecontrollerConfig), true, "")

	err = o.cleanServiceAnnotations()
	if err != nil {
		return err
	}

	// if tls create CRDs
	if o.tls {
		err = o.ensureCertmanagerSetup(exposecontrollerConfig)
		if err != nil {
			return err
		}
	}
	// annotate any service that has expose=true with correct certmanager staging / prod annotation
	err = o.annotateExposedServicesWithCertManager()
	if err != nil {
		return err
	}

	// delete ingress
	for name, namespace := range ingressToDelete {
		log.Infof("Deleting ingress %s/%s\n", namespace, name)
		err := o.kubeClient.ExtensionsV1beta1().Ingresses(namespace).Delete(name, &metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("cannot delete ingress rule %s in namespace %s: %v", name, namespace, err)
		}
	}

	err = o.recreateIngressRules(exposecontrollerConfig)
	if err != nil {
		return err
	}

	err = o.updateJenkinsURL(o.TargetNamespaces)
	if err != nil {
		return err
	}
	// todo wait for certs secrets to update ingress rules?

	log.Success("Ingress rules recreated\n")

	if exposecontrollerConfig["tls-acme"] == "true" {
		log.Warn("It can take around 5 minutes for Cert Manager to get certificates from Lets Encrypt and update Ingress rules\n")
		log.Info("Use the following commands to diagnose any issues:\n")
		log.Info("jx logs cert-manager-cert-manager -n cert-manager\n")
		log.Info("kubectl describe certificates\n")
		log.Info("kubectl describe issuers\n")
	}

	return nil
}

func (o *UpgradeIngressOptions) getExistingIngressRules() (map[string]string, error) {
	existingIngressNames := map[string]string{}
	var confirmMessage string
	if o.Cluster {
		confirmMessage = "Existing ingress rules found in the cluster.  Confirm to delete all and recreate them"

		ings, err := o.kubeClient.ExtensionsV1beta1().Ingresses("").List(metav1.ListOptions{})
		if err != nil {
			return existingIngressNames, fmt.Errorf("cannot list all ingresses in cluster: %v", err)
		}
		for _, i := range ings.Items {
			existingIngressNames[i.Name] = i.Namespace
		}

		nsList, err := o.kubeClient.CoreV1().Namespaces().List(metav1.ListOptions{})
		for _, n := range nsList.Items {
			o.TargetNamespaces = append(o.TargetNamespaces, n.Name)
		}

	} else if len(o.Namespaces) > 0 {
		confirmMessage = fmt.Sprintf("Existing ingress rules found in namespaces %v namespace.  Confirm to delete and recreate them", o.Namespaces)
		// loop round each
		for _, n := range o.Namespaces {
			ings, err := o.kubeClient.ExtensionsV1beta1().Ingresses(n).List(metav1.ListOptions{})
			if err != nil {
				return existingIngressNames, fmt.Errorf("cannot list all ingresses in cluster: %v", err)
			}
			for _, i := range ings.Items {
				existingIngressNames[i.Name] = i.Namespace
			}
			o.TargetNamespaces = append(o.TargetNamespaces, n)
		}
	} else {
		confirmMessage = "Existing ingress rules found in current namespace.  Confirm to delete and recreate them"
		// fall back to current ns only
		log.Infof("looking for existing ingress rules in current namesace %s\n", o.currentNamespace)

		ings, err := o.kubeClient.ExtensionsV1beta1().Ingresses(o.currentNamespace).List(metav1.ListOptions{})
		if err != nil {
			return existingIngressNames, fmt.Errorf("cannot list all ingresses in cluster: %v", err)
		}
		for _, i := range ings.Items {
			existingIngressNames[i.Name] = i.Namespace
		}
		o.TargetNamespaces = append(o.TargetNamespaces, o.currentNamespace)
	}

	confirm := &survey.Confirm{
		Message: confirmMessage,
		Default: true,
	}
	flag := true
	err := survey.AskOne(confirm, &flag, nil)
	if err != nil {
		return existingIngressNames, err
	}
	if !flag {
		return existingIngressNames, errors.New("Not able to automatically delete existing ingress rules.  Either delete manually or change the scope the command should run in")
	}

	return existingIngressNames, nil
}

func (o *UpgradeIngressOptions) confirmExposecontrollerConfig() (map[string]string, error) {
	config := map[string]string{}
	// get current exposecontroller configmap to use as defaults
	devNamespace, _, err := kube.GetDevNamespace(o.kubeClient, o.currentNamespace)
	if err != nil {
		return config, fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}

	config, err = kube.GetTeamExposecontrollerConfig(o.kubeClient, devNamespace)
	if err != nil {
		log.Warnf("cannot get existing team exposecontroller config from namespace %s: %v", devNamespace, err)
		config = map[string]string{}
	}

	config["exposer"], err = util.PickNameWithDefault([]string{"Ingress", "Route"}, "Expose type", config["exposer"])
	if err != nil {
		return config, err
	}

	config["domain"], err = util.PickValue("Domain:", config["domain"], true)
	if err != nil {
		return config, err
	}
	o.Domain = config["domain"]

	config["http"] = "true"

	if !strings.HasSuffix(config["domain"], "nip.io") {

		o.tls = util.Confirm("If your network is publicly available would you like to enable cluster wide TLS?", true, "Enables cert-manager and configures TLS with signed certificates from LetsEncrypt")
		config["tls-acme"] = strconv.FormatBool(o.tls)

		if o.tls {
			clusterIssuer, err := util.PickNameWithDefault([]string{"prod", "staging"}, "Use LetsEncrypt staging or production?  Warning if testing use staging else you may be rate limited:", "staging")
			if err != nil {
				return config, err
			}
			o.Issuer = "letsencrypt-" + clusterIssuer

			email1, err := o.getCommandOutput("", "git", "config", "user.email")
			if err != nil {
				return config, err
			}

			o.Email = strings.TrimSpace(email1)

			o.Email, err = util.PickValue("Email address to register with LetsEncrypt:", o.Email, true)
			if err != nil {
				return config, err
			}

			config["http"] = "false"
		}
	}
	return config, nil

}
func (o *UpgradeIngressOptions) recreateIngressRules(config map[string]string) error {
	devNamespace, _, err := kube.GetDevNamespace(o.kubeClient, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}
	for _, n := range o.TargetNamespaces {
		err = o.cleanExposecontrollerReources(n)
		if err != nil {
			return err
		}

		err := o.cleanTLSSecrets(n)
		if err != nil {
			return err
		}

		err = o.cleanCertmanagerReources(n)
		if err != nil {
			return err
		}

		err = o.runExposecontroller(devNamespace, n, strings.ToLower(randomdata.SillyName()), config)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *UpgradeIngressOptions) ensureCertmanagerSetup(config map[string]string) error {

	if !o.SkipCertManager {
		log.Infof("Looking for %s deployment in namespace %s\n", CertManagerDeployment, CertManagerNamespace)
		_, err := kube.GetDeploymentPods(o.kubeClient, CertManagerDeployment, CertManagerNamespace)
		if err != nil {
			ok := util.Confirm("CertManager deployment not found, shall we install it now?", true, "CertManager automatically configures Ingress rules with TLS using signed certificates from LetsEncrypt")
			if ok {

				values := []string{"rbac.create=true", "ingressShim.extraArgs='{--default-issuer-name=letsencrypt-staging,--default-issuer-kind=ClusterIssuer}'"}
				err = o.installChart("cert-manager", "stable/cert-manager", "", CertManagerNamespace, true, values)
				if err != nil {
					return fmt.Errorf("CertManager deployment failed: %v", err)
				}

				log.Info("waiting for CertManager deployment to be ready, this can take a few minutes\n")

				err = kube.WaitForDeploymentToBeReady(o.kubeClient, CertManagerDeployment, CertManagerNamespace, 10*time.Minute)
				if err != nil {
					return err
				}

			}
		}
	}
	return nil
}

func (o *UpgradeIngressOptions) annotateExposedServicesWithCertManager() error {
	for _, n := range o.TargetNamespaces {
		svcList, err := kube.GetServices(o.kubeClient, n)
		if err != nil {
			return err
		}
		for _, s := range svcList {

			if s.Annotations[kube.ExposeAnnotation] == "true" && s.Annotations[kube.JenkinsXSkipTLSAnnotation] != "true" {
				existingAnnotations, _ := s.Annotations[kube.ExposeIngressAnnotation]
				// if no existing `fabric8.io/ingress.annotations` initialise and add else update with ClusterIssuer
				if len(existingAnnotations) > 0 {
					s.Annotations[kube.ExposeIngressAnnotation] = existingAnnotations + "\n" + kube.CertManagerAnnotation + ": " + o.Issuer
				} else {
					s.Annotations[kube.ExposeIngressAnnotation] = kube.CertManagerAnnotation + ": " + o.Issuer
				}
				_, err = o.kubeClient.CoreV1().Services(n).Update(s)
				if err != nil {
					return fmt.Errorf("failed to annotate and update service %s in namespace %s: %v", s.Name, n, err)
				}
			}
		}
	}
	return nil
}

func (o *UpgradeIngressOptions) cleanExposecontrollerReources(ns string) error {

	// lets not error if they dont exist
	o.kubeClient.RbacV1().Roles(ns).Delete("exposecontroller", &metav1.DeleteOptions{})
	o.kubeClient.RbacV1().RoleBindings(ns).Delete("exposecontroller", &metav1.DeleteOptions{})
	o.kubeClient.RbacV1().ClusterRoleBindings().Delete("exposecontroller", &metav1.DeleteOptions{})
	o.kubeClient.CoreV1().ConfigMaps(ns).Delete("exposecontroller", &metav1.DeleteOptions{})
	o.kubeClient.CoreV1().ServiceAccounts(ns).Delete("exposecontroller", &metav1.DeleteOptions{})
	o.kubeClient.BatchV1().Jobs(ns).Delete("exposecontroller", &metav1.DeleteOptions{})

	return nil
}

func (o *UpgradeIngressOptions) cleanCertmanagerReources(ns string) error {

	if o.Issuer == CertmanagerIssuerProd {
		_, err := o.kubeClient.CoreV1().RESTClient().Get().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/issuers", ns)).Name(CertmanagerIssuerProd).DoRaw()
		if err == nil {
			// existing clusterissuers found, recreate
			_, err = o.kubeClient.CoreV1().RESTClient().Delete().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/issuers", ns)).Name(CertmanagerIssuerProd).DoRaw()
			if err != nil {
				return fmt.Errorf("failed to delete issuer %s %v", "letsencrypt-prod", err)
			}
		}

		if o.tls {
			issuerProd := fmt.Sprintf(certmanager.Cert_manager_issuer_prod, o.Email)
			json, err := yaml.YAMLToJSON([]byte(issuerProd))

			resp, err := o.kubeClient.CoreV1().RESTClient().Post().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/issuers", ns)).Body(json).DoRaw()
			if err != nil {
				return fmt.Errorf("failed to create issuer %v: %s", err, string(resp))
			}
		}

	} else {
		_, err := o.kubeClient.CoreV1().RESTClient().Get().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/issuers", ns)).Name(CertmanagerIssuerStaging).DoRaw()
		if err == nil {
			// existing clusterissuers found, recreate
			resp, err := o.kubeClient.CoreV1().RESTClient().Delete().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/issuers", ns)).Name(CertmanagerIssuerStaging).DoRaw()
			if err != nil {
				return fmt.Errorf("failed to delete issuer %v: %s", err, string(resp))
			}
		}

		if o.tls {
			issuerStage := fmt.Sprintf(certmanager.Cert_manager_issuer_stage, o.Email)
			json, err := yaml.YAMLToJSON([]byte(issuerStage))

			resp, err := o.kubeClient.CoreV1().RESTClient().Post().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/issuers", ns)).Body(json).DoRaw()
			if err != nil {
				return fmt.Errorf("failed to create issuer %v: %s", err, string(resp))
			}
		}
	}

	// lets not error if they dont exist
	o.kubeClient.CoreV1().RESTClient().Delete().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/certificates", ns)).Name(CertmanagerCertificateStaging).DoRaw()
	o.kubeClient.CoreV1().RESTClient().Delete().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/certificates", ns)).Name(CertmanagerCertificateProd).DoRaw()

	if o.tls {
		cert := fmt.Sprintf(certmanager.Cert_manager_certificate, o.Issuer, o.Issuer, o.Domain, o.Domain, o.Domain)
		json, err := yaml.YAMLToJSON([]byte(cert))
		if err != nil {
			return fmt.Errorf("unable to convert YAML %s to JSON: %v", cert, err)
		}
		_, err = o.kubeClient.CoreV1().RESTClient().Post().RequestURI(fmt.Sprintf("/apis/certmanager.k8s.io/v1alpha1/namespaces/%s/certificates", ns)).Body(json).DoRaw()
		if err != nil {
			return fmt.Errorf("failed to create certificate %v", err)
		}
	}

	return nil
}

func (o *UpgradeIngressOptions) cleanServiceAnnotations() error {
	for _, n := range o.TargetNamespaces {
		svcList, err := kube.GetServices(o.kubeClient, n)
		if err != nil {
			return err
		}
		for _, s := range svcList {
			if s.Annotations[kube.ExposeAnnotation] == "true" && s.Annotations[kube.JenkinsXSkipTLSAnnotation] != "true" {
				// if no existing `fabric8.io/ingress.annotations` initialise and add else update with ClusterIssuer
				annotationsForIngress, _ := s.Annotations[kube.ExposeIngressAnnotation]
				if len(annotationsForIngress) > 0 {

					var newAnnotations []string
					annotations := strings.Split(annotationsForIngress, "\n")
					for _, element := range annotations {
						annotation := strings.SplitN(element, ":", 2)
						key, _ := annotation[0], strings.TrimSpace(annotation[1])
						if key != kube.CertManagerAnnotation {
							newAnnotations = append(newAnnotations, element)
						}
					}
					annotationsForIngress = ""
					for _, v := range newAnnotations {
						if len(annotationsForIngress) > 0 {
							annotationsForIngress = annotationsForIngress + "\n" + v
						} else {
							annotationsForIngress = v
						}
					}
					s.Annotations[kube.ExposeIngressAnnotation] = annotationsForIngress

				}
				delete(s.Annotations, kube.ExposeURLAnnotation)

				_, err = o.kubeClient.CoreV1().Services(n).Update(s)
				if err != nil {
					return fmt.Errorf("failed to clean service %s annotations in namespace %s: %v", s.Name, n, err)
				}
			}
		}
	}

	return nil
}
func (o *UpgradeIngressOptions) cleanTLSSecrets(ns string) error {
	// delete the tls related secrets so we dont reuse old ones when switching from http to https
	secrets, err := o.kubeClient.CoreV1().Secrets(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, s := range secrets.Items {
		if strings.HasPrefix(s.Name, "tls-") {
			err := o.kubeClient.CoreV1().Secrets(ns).Delete(s.Name, &metav1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("failed to delete tls secret %s: %v", s.Name, err)
			}
		}
	}
	return nil
}
