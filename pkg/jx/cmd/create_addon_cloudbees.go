package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"gopkg.in/AlecAivazis/survey.v1"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultCloudBeesReleaseName = "cb"
	defaultCloudBeesNamespace   = "jx"
	cdxRepoName                 = "cb"
	cbServiceName               = "cb-cdx"
	cdxRepoUrl                  = "https://%s:%s@chartmuseum.jx.charts-demo.cloudbees.com"
	serviceaccountsClusterAdmin = "serviceaccounts-cluster-admin"
)

var (
	CreateAddonCloudBeesLong = templates.LongDesc(`
		Creates the CloudBees app for Kubernetes addon

		CloudBees app for Kubernetes provides unified Continuous Delivery Environment console to make it easier to do CI/CD and Environments across a number of microservices and teams

		For more information please see [https://www.cloudbees.com/blog/want-help-build-cloudbees-kubernetes-jenkins-x](https://www.cloudbees.com/blog/want-help-build-cloudbees-kubernetes-jenkins-x)
`)

	CreateAddonCloudBeesExample = templates.Examples(`
		# Create the cloudbees addon 
		jx create addon cloudbees
	`)
)

// CreateAddonCloudBeesOptions the options for the create spring command
type CreateAddonCloudBeesOptions struct {
	CreateAddonOptions
	Sso      bool
	Basic    bool
	Password string
}

// NewCmdCreateAddonCloudBees creates a command object for the "create" command
func NewCmdCreateAddonCloudBees(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &CreateAddonCloudBeesOptions{
		CreateAddonOptions: CreateAddonOptions{
			CreateOptions: CreateOptions{
				CommonOptions: CommonOptions{
					Factory: f,
					In:      in,
					Out:     out,
					Err:     errOut,
				},
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "cloudbees",
		Short:   "Create the CloudBees app for Kubernetes (a web console for working with CI/CD, Environments and GitOps)",
		Aliases: []string{"cloudbee", "cb", "cdx", "kubecd"},
		Long:    CreateAddonCloudBeesLong,
		Example: CreateAddonCloudBeesExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&options.Sso, "sso", "", false, "Enable single sign-on")
	cmd.Flags().BoolVarP(&options.Basic, "basic", "", false, "Enable basic auth")
	cmd.Flags().StringVarP(&options.Password, "password", "p", "", "Password to access UI when using basic auth.  Defaults to default Jenkins X admin password.")
	options.addCommonFlags(cmd)
	options.addFlags(cmd, defaultCloudBeesNamespace, defaultCloudBeesReleaseName)
	return cmd
}

// Run implements the command
func (o *CreateAddonCloudBeesOptions) Run() error {

	if o.Sso == false && o.Basic == false {
		return fmt.Errorf("please use --sso or --basic flag")
	}

	surveyOpts := survey.WithStdio(o.In, o.Out, o.Err)
	c, _, err := o.KubeClient()
	if err != nil {
		return err
	}

	// todo add correct roles to cdx rather than make EVERY service account cluster admin
	_, err = c.RbacV1().ClusterRoleBindings().Get(serviceaccountsClusterAdmin, v1.GetOptions{})
	if err != nil {

		ok := false
		log.Warn("CloudBees app for Kubernetes is in preview and for now requires cluster admin to be granted to ALL service accounts in your cluster.  Check CLI help for more info.\n")
		prompt := &survey.Confirm{
			Message: "create cluster admin rolebinding?",
			Default: false,
			Help:    "a cluster admin role provides full privileges and therefore this action should not be run on anything other than a demo cluster that can be recreated",
		}
		survey.AskOne(prompt, &ok, nil, surveyOpts)

		if !ok {
			log.Info("aborting the cdx addon\n")
			return nil
		}

		rb := rbacv1.ClusterRoleBinding{
			ObjectMeta: v1.ObjectMeta{
				Name: serviceaccountsClusterAdmin,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			},
			Subjects: []rbacv1.Subject{
				{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Group",
					Name:     "system:serviceaccounts",
				},
			},
		}
		_, err = c.RbacV1().ClusterRoleBindings().Create(&rb)
		if err != nil {
			return err
		}
	}

	// check if helm repo is missing, the repo is authenticated and includes username/password so check with dummy values
	// first as we wont need to prompt for username password if the host part of the URL matches an existing repo
	missing, err := o.isHelmRepoMissing(fmt.Sprintf(cdxRepoUrl, "dummy", "dummy"))
	if err != nil {
		return err
	}

	if missing {
		log.Infof(`
You will need your username and password to install this addon while it is in preview.
To register to get your username/password to to: %s

`, util.ColorInfo("https://pages.cloudbees.com/K8s"))

		username := ""
		prompt := &survey.Input{
			Message: "CloudBees Preview username",
			Help:    "CloudBees is in private preview which requires a username / password for installation",
		}
		survey.AskOne(prompt, &username, nil, surveyOpts)

		password := ""
		passPrompt := &survey.Password{
			Message: "CloudBees Preview password",
			Help:    "CloudBees is in private preview which requires a username / password for installation",
		}
		survey.AskOne(passPrompt, &password, nil, surveyOpts)

		err := o.addHelmRepoIfMissing(fmt.Sprintf(cdxRepoUrl, username, password), cdxRepoName)
		if err != nil {
			return err
		}
	}

	if o.Sso {
		log.Infof("Configuring %s...\n", util.ColorInfo("single sign-on"))
		o.devNamespace, _, err = kube.GetDevNamespace(o.KubeClientCached, o.currentNamespace)
		if err != nil {
			return errors.Wrap(err, "retrieving the development namesapce")
		}
		ingressConfig, err := kube.GetIngressConfig(o.KubeClientCached, o.devNamespace)
		if err != nil {
			return errors.Wrap(err, "retrieving existing ingress configuration")
		}
		domain, err := util.PickValue("Domain:", ingressConfig.Domain, true, o.In, o.Out, o.Err)
		if err != nil {
			return errors.Wrap(err, "reading domain")
		}

		dexURL, err := util.PickValue("Dex URL:", fmt.Sprintf("https://dex.sso.%s", ingressConfig.Domain), true, o.In, o.Out, o.Err)
		if err != nil {
			return errors.Wrap(err, "reading dex URL")
		}

		// Strip the trailing slash automatically
		dexURL = strings.TrimSuffix(dexURL, "/")

		err = o.ensureCertmanager()
		if err != nil {
			return errors.Wrap(err, "ensuring cert-manager is installed")
		}

		ingressConfig.TLS = true
		ingressConfig.Issuer = kube.CertmanagerIssuerProd
		err = kube.CleanCertmanagerResources(o.KubeClientCached, o.Namespace, ingressConfig)
		if err != nil {
			return errors.Wrap(err, "creating cert-manager issuer")
		}

		values := []string{
			"sso.create=true",
			"sso.oidcIssuerUrl=" + dexURL,
			"sso.domain=" + domain,
			"sso.certIssuerName=" + ingressConfig.Issuer}

		if len(o.SetValues) > 0 {
			o.SetValues = o.SetValues + "," + strings.Join(values, ",")
		} else {
			o.SetValues = strings.Join(values, ",")
		}

	} else if o.Basic {
		devNamespace, _, err := kube.GetDevNamespace(o.KubeClientCached, o.currentNamespace)
		if err != nil {
			return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
		}

		if o.Password == "" {
			o.Password, err = o.getDefaultAdminPassword(devNamespace)
			if err != nil {
				return err
			}
		}

		o.SetValues = strings.Join([]string{"sso.create=false"}, ",")

		svc, err := c.CoreV1().Services(o.currentNamespace).Get(cbServiceName, meta_v1.GetOptions{})
		if err != nil {
			return err
		}
		annotationsUpdated := false
		if svc.Annotations[kube.AnnotationExpose] == "" {
			svc.Annotations[kube.AnnotationExpose] = "true"
			annotationsUpdated = true
		}
		if svc.Annotations[kube.AnnotationIngress] == "" {
			svc.Annotations[kube.AnnotationIngress] = "nginx.ingress.kubernetes.io/auth-type: basic\nnginx.ingress.kubernetes.io/auth-secret: jx-basic-auth"
			annotationsUpdated = true
		}
		if annotationsUpdated {
			svc, err = o.KubeClientCached.CoreV1().Services(o.Namespace).Update(svc)
			if err != nil {
				return fmt.Errorf("failed to update service %s/%s", o.Namespace, cbServiceName)
			}
		}
	}

	err = o.CreateAddon("cb")
	if err != nil {
		return err
	}

	_, _, err = o.KubeClient()
	if err != nil {
		return err
	}

	devNamespace, _, err := kube.GetDevNamespace(o.KubeClientCached, o.currentNamespace)
	if err != nil {
		return fmt.Errorf("cannot find a dev team namespace to get existing exposecontroller config from. %v", err)
	}

	log.Infof("using exposecontroller config from dev namespace %s\n", devNamespace)
	log.Infof("target namespace %s\n", o.Namespace)

	// create the ingress rule
	err = o.expose(devNamespace, o.Namespace, o.Password)
	if err != nil {
		return err
	}

	return nil
}
