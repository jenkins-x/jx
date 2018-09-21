package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	CertManagerDeployment = "cert-manager"
	CertManagerNamespace  = "cert-manager"
	Exposecontroller      = "exposecontroller"
)

// UpgradeIngressOptions the options for the create spring command
type UpgradeIngressOptions struct {
	CreateOptions

	SkipCertManager  bool
	Cluster          bool
	Namespaces       []string
	Version          string
	TargetNamespaces []string
	Services         []string

	IngressConfig kube.IngressConfig
}

// NewCmdUpgradeIngress defines the command
func NewCmdUpgradeIngress(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &UpgradeIngressOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
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
	options.addFlags(cmd)

	return cmd
}

func (o *UpgradeIngressOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&o.Cluster, "cluster", "", false, "Enable cluster wide Ingress upgrade")
	cmd.Flags().StringArrayVarP(&o.Namespaces, "namespaces", "", []string{}, "Namespaces to upgrade")
	cmd.Flags().BoolVarP(&o.SkipCertManager, "skip-certmanager", "", false, "Skips certmanager installation")
	cmd.Flags().StringArrayVarP(&o.Services, "services", "", []string{}, "Services to upgrdde")
}

// Run implements the command
func (o *UpgradeIngressOptions) Run() error {

	_, _, err := o.KubeClient()
	if err != nil {
		return fmt.Errorf("cannot connect to kubernetes cluster: %v", err)
	}

	o.devNamespace, _, err = kube.GetDevNamespace(o.KubeClientCached, o.currentNamespace)
	if err != nil {
		return err
	}

	// if existing ingress exist in the namespaces ask do you want to delete them?
	ingressToDelete, err := o.getExistingIngressRules()
	if err != nil {
		return err
	}

	// wizard to ask for config values
	err = o.confirmExposecontrollerConfig()
	if err != nil {
		return err
	}

	// confirm values
	util.Confirm(fmt.Sprintf("Using  config values %v, ok?", o.IngressConfig), true, "", o.In, o.Out, o.Err)

	// save details to a configmap
	_, err = kube.SaveAsConfigMap(o.KubeClientCached, kube.ConfigMapIngressConfig, o.devNamespace, o.IngressConfig)
	if err != nil {
		return err
	}

	err = o.CleanServiceAnnotations(o.Services...)
	if err != nil {
		return err
	}

	// if tls create CRDs
	if o.IngressConfig.TLS {
		err = o.ensureCertmanagerSetup()
		if err != nil {
			return err
		}
	}
	// annotate any service that has expose=true with correct certmanager staging / prod annotation
	err = o.AnnotateExposedServicesWithCertManager(o.Services...)
	if err != nil {
		return err
	}

	// delete ingress
	for name, namespace := range ingressToDelete {
		log.Infof("Deleting ingress %s/%s\n", namespace, name)
		err := o.KubeClientCached.ExtensionsV1beta1().Ingresses(namespace).Delete(name, &metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("cannot delete ingress rule %s in namespace %s: %v", name, namespace, err)
		}
	}

	err = o.recreateIngressRules()
	if err != nil {
		return err
	}

	err = o.updateJenkinsURL(o.TargetNamespaces)
	if err != nil {
		return err
	}
	// todo wait for certs secrets to update ingress rules?

	log.Success("Ingress rules recreated\n")

	if o.IngressConfig.TLS {
		log.Warn("It can take around 5 minutes for Cert Manager to get certificates from Lets Encrypt and update Ingress rules\n")
		log.Info("Use the following commands to diagnose any issues:\n")
		log.Infof("jx logs %s -n %s\n", CertManagerDeployment, CertManagerNamespace)
		log.Info("kubectl describe certificates\n")
		log.Info("kubectl describe issuers\n")
	}

	return nil
}

func (o *UpgradeIngressOptions) isIngressForServices(ingress *v1beta1.Ingress) bool {
	services := o.Services
	if len(services) == 0 {
		// allow all ingresses if no services filter is defined
		return true
	}
	rules := ingress.Spec.Rules
	for _, rule := range rules {
		http := rule.IngressRuleValue.HTTP
		if http == nil {
			continue
		}
		for _, path := range http.Paths {
			service := path.Backend.ServiceName
			i := util.StringArrayIndex(services, service)
			if i >= 0 {
				return true
			}
		}
	}
	return false
}

func (o *UpgradeIngressOptions) getExistingIngressRules() (map[string]string, error) {
	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	existingIngressNames := map[string]string{}
	var confirmMessage string
	if o.Cluster {
		confirmMessage = "Existing ingress rules found in the cluster.  Confirm to delete all and recreate them"

		ings, err := o.KubeClientCached.ExtensionsV1beta1().Ingresses("").List(metav1.ListOptions{})
		if err != nil {
			return existingIngressNames, fmt.Errorf("cannot list all ingresses in cluster: %v", err)
		}
		for _, i := range ings.Items {
			if i.Annotations[kube.ExposeGeneratedByAnnotation] == Exposecontroller {
				if o.isIngressForServices(&i) {
					existingIngressNames[i.Name] = i.Namespace
				}
			}
		}

		nsList, err := o.KubeClientCached.CoreV1().Namespaces().List(metav1.ListOptions{})
		for _, n := range nsList.Items {
			o.TargetNamespaces = append(o.TargetNamespaces, n.Name)
		}

	} else if len(o.Namespaces) > 0 {
		confirmMessage = fmt.Sprintf("Existing ingress rules found in namespaces %v namespace.  Confirm to delete and recreate them", o.Namespaces)
		// loop round each
		for _, n := range o.Namespaces {
			ings, err := o.KubeClientCached.ExtensionsV1beta1().Ingresses(n).List(metav1.ListOptions{})
			if err != nil {
				return existingIngressNames, fmt.Errorf("cannot list all ingresses in cluster: %v", err)
			}
			for _, i := range ings.Items {
				if i.Annotations[kube.ExposeGeneratedByAnnotation] == Exposecontroller {
					if o.isIngressForServices(&i) {
						existingIngressNames[i.Name] = i.Namespace
					}
				}
			}
			o.TargetNamespaces = append(o.TargetNamespaces, n)
		}
	} else {
		confirmMessage = "Existing ingress rules found in current namespace.  Confirm to delete and recreate them"
		// fall back to current ns only
		log.Infof("looking for existing ingress rules in current namespace %s\n", o.currentNamespace)

		ings, err := o.KubeClientCached.ExtensionsV1beta1().Ingresses(o.currentNamespace).List(metav1.ListOptions{})
		if err != nil {
			return existingIngressNames, fmt.Errorf("cannot list all ingresses in cluster: %v", err)
		}
		for _, i := range ings.Items {
			if i.Annotations[kube.ExposeGeneratedByAnnotation] == Exposecontroller {
				if o.isIngressForServices(&i) {
					existingIngressNames[i.Name] = i.Namespace
				}
			}
		}
		o.TargetNamespaces = append(o.TargetNamespaces, o.currentNamespace)
	}

	if len(existingIngressNames) == 0 {
		return existingIngressNames, errors.New("No ingress rules found")
	}

	confirm := &survey.Confirm{
		Message: confirmMessage,
		Default: true,
	}
	flag := true
	err := survey.AskOne(confirm, &flag, nil, surveyOpts)
	if err != nil {
		return existingIngressNames, err
	}
	if !flag {
		return existingIngressNames, errors.New("Not able to automatically delete existing ingress rules.  Either delete manually or change the scope the command should run in")
	}

	return existingIngressNames, nil
}

func (o *UpgradeIngressOptions) confirmExposecontrollerConfig() error {

	// get current ingress config to use as existing defaults
	devNamespace, _, err := kube.GetDevNamespace(o.KubeClientCached, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}

	o.IngressConfig, err = kube.GetIngressConfig(o.KubeClientCached, devNamespace)
	if err != nil {
		// carry on as it just means we dont have any defaults
	}

	o.IngressConfig.Exposer, err = util.PickNameWithDefault([]string{"Ingress", "Route"}, "Expose type", o.IngressConfig.Exposer, o.In, o.Out, o.Err)
	if err != nil {
		return err
	}

	o.IngressConfig.Domain, err = util.PickValue("Domain:", o.IngressConfig.Domain, true, o.In, o.Out, o.Err)
	if err != nil {
		return err
	}

	if !strings.HasSuffix(o.IngressConfig.Domain, "nip.io") {

		o.IngressConfig.TLS = util.Confirm("If your network is publicly available would you like to enable cluster wide TLS?", true, "Enables cert-manager and configures TLS with signed certificates from LetsEncrypt", o.In, o.Out, o.Err)

		if o.IngressConfig.TLS {
			clusterIssuer, err := util.PickNameWithDefault([]string{"prod", "staging"}, "Use LetsEncrypt staging or production?  Warning if testing use staging else you may be rate limited:", "staging", o.In, o.Out, o.Err)
			if err != nil {
				return err
			}
			o.IngressConfig.Issuer = "letsencrypt-" + clusterIssuer

			if o.IngressConfig.Email == "" {
				email1, err := o.getCommandOutput("", "git", "config", "user.email")
				if err != nil {
					return err
				}

				o.IngressConfig.Email = strings.TrimSpace(email1)
			}

			o.IngressConfig.Email, err = util.PickValue("Email address to register with LetsEncrypt:", o.IngressConfig.Email, true, o.In, o.Out, o.Err)
			if err != nil {
				return err
			}
		}
	}

	return nil

}
func (o *UpgradeIngressOptions) recreateIngressRules() error {
	devNamespace, _, err := kube.GetDevNamespace(o.KubeClientCached, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}
	for _, n := range o.TargetNamespaces {
		o.CleanExposecontrollerReources(n)

		err := o.cleanTLSSecrets(n)
		if err != nil {
			return err
		}

		err = kube.CleanCertmanagerResources(o.KubeClientCached, n, o.IngressConfig)
		if err != nil {
			return err
		}

		err = o.runExposecontroller(devNamespace, n, o.IngressConfig, o.Services...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *UpgradeIngressOptions) ensureCertmanagerSetup() error {
	if !o.SkipCertManager {
		return o.ensureCertmanager()
	}
	return nil
}

// AnnotateExposedServicesWithCertManager annotates exposed service with cert manager
func (o *UpgradeIngressOptions) AnnotateExposedServicesWithCertManager(services ...string) error {
	for _, n := range o.TargetNamespaces {
		err := kube.AnnotateNamespaceServicesWithCertManager(o.KubeClientCached, n, o.IngressConfig.Issuer, services...)
		if err != nil {
			return err
		}
	}
	return nil
}

// CleanServiceAnnotations cleans service annotations
func (o *UpgradeIngressOptions) CleanServiceAnnotations(services ...string) error {
	for _, n := range o.TargetNamespaces {
		err := kube.CleanServiceAnnotations(o.KubeClientCached, n, services...)
		if err != nil {
			return err
		}
	}

	return nil
}
func (o *UpgradeIngressOptions) cleanTLSSecrets(ns string) error {
	// delete the tls related secrets so we dont reuse old ones when switching from http to https
	secrets, err := o.KubeClientCached.CoreV1().Secrets(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, s := range secrets.Items {
		if strings.HasPrefix(s.Name, "tls-") {
			err := o.KubeClientCached.CoreV1().Secrets(ns).Delete(s.Name, &metav1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("failed to delete tls secret %s: %v", s.Name, err)
			}
		}
	}
	return nil
}
