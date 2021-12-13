package dashboard

import (
	"context"
	"net/url"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/services"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"

	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/pkg/browser"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
)

type Options struct {
	options.BaseOptions
	KubeClient          kubernetes.Interface
	Namespace           string
	ServiceName         string
	BasicAuthSecretName string
	NoBrowser           bool
	Quiet               bool
}

var (
	cmdLong = templates.LongDesc(`
		View the Jenkins X Pipelines Dashboard.`)

	cmdExample = templates.Examples(`
		# open the dashboard
		jx dashboard

		# display the URL only without opening a browser
		jx --no-open
`)

	info = termcolor.ColorInfo
)

// NewCmdDashboard opens the dashboard
func NewCmdDashboard() (*cobra.Command, *Options) {
	o := &Options{}
	cmd := &cobra.Command{
		Use:     "dashboard",
		Aliases: []string{"dash"},
		Short:   "View the Jenkins X Pipelines Dashboard",
		Long:    cmdLong,
		Example: cmdExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&o.NoBrowser, "no-open", "", false, "Disable opening the URL; just show it on the console")
	cmd.Flags().StringVarP(&o.ServiceName, "name", "n", "jx-pipelines-visualizer", "The name of the dashboard service")
	cmd.Flags().StringVarP(&o.BasicAuthSecretName, "secret", "s", "jx-basic-auth-user-password", "The name of the Secret containing the basic auth login/password")
	o.BaseOptions.AddBaseFlags(cmd)
	return cmd, o
}

func (o *Options) Run() error {
	var err error
	o.KubeClient, o.Namespace, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
	if err != nil {
		return errors.Wrap(err, "creating kubernetes client")
	}
	client := o.KubeClient

	u, err := services.FindServiceURL(client, o.Namespace, o.ServiceName)
	if err != nil {
		return errors.Wrapf(err, "failed to find dashboard URL. Check you have 'chart: jxgh/jx-pipelines-visualizer' in your helmfile.yaml")
	}
	if u == "" {
		return errors.Errorf("no dashboard URL. Check you have 'chart: jxgh/jx-pipelines-visualizer' in your helmfile.yaml")
	}

	log.Logger().Infof("Jenkins X dashboard is running at: %s", info(u))

	if o.NoBrowser {
		return nil
	}

	u, err = o.addUserPasswordToURL(u)
	if err != nil {
		return errors.Wrapf(err, "failed to enrich dashboard URL %s", u)
	}

	log.Logger().Debugf("opening: %s", info(u))

	err = browser.OpenURL(u)
	if err != nil {
		return err
	}
	return nil
}

func (o *Options) addUserPasswordToURL(urlText string) (string, error) {
	name := o.BasicAuthSecretName
	ns := o.Namespace
	secret, err := o.KubeClient.CoreV1().Secrets(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return urlText, errors.Wrapf(err, "failed to load Secret %s in namespace %s", name, ns)
	}
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	username := string(secret.Data["username"])
	password := string(secret.Data["password"])

	if username == "" {
		log.Logger().Warnf("secret %s in namespace %s has no username", name, ns)
		return urlText, nil
	}
	if password == "" {
		log.Logger().Warnf("secret %s in namespace %s has no password", name, ns)
		return urlText, nil
	}

	u, err := url.Parse(urlText)
	if err != nil {
		return urlText, errors.Wrapf(err, "failed to parse URL %s", urlText)
	}
	u.User = url.UserPassword(username, password)
	return u.String(), nil
}
