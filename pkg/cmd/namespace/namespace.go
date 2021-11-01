package namespace

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/input"
	"github.com/jenkins-x/jx-helpers/v3/pkg/input/survey"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-kube-client/v3/pkg/kubeclient"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"

	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd/api"
)

type Options struct {
	KubeClient kubernetes.Interface
	Input      input.Interface
	JXClient   jxc.Interface
	Args       []string
	Env        string
	PickEnv    bool
	Create     bool
	QuiteMode  bool
	BatchMode  bool
}

var (
	cmdLong = templates.LongDesc(`
		Displays or changes the current namespace.`)
	cmdExample = templates.Examples(`
		# view the current namespace
		jx --batch-mode ns

		# interactively select the namespace to switch to
		jx ns

		# change the current namespace to 'cheese'
		jx ns cheese

		# change the current namespace to 'brie' creating it if necessary
	    jx ns --create brie

		# switch to the namespace of the staging environment
		jx ns --env staging

		# switch back to the dev environment namespace
		jx ns --e dev

		# interactively select the Environment to switch to
		jx ns --pick
`)

	info = termcolor.ColorInfo
)

func NewCmdNamespace() (*cobra.Command, *Options) {
	o := &Options{}
	cmd := &cobra.Command{
		Use:     "namespace",
		Aliases: []string{"ns", "ctx"},
		Short:   "View or change the current namespace context in the current Kubernetes cluster",
		Long:    cmdLong,
		Example: cmdExample,
		Run: func(cmd *cobra.Command, args []string) {
			o.Args = args
			err := o.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&o.Create, "create", "c", false, "Creates the specified namespace if it does not exist")
	cmd.Flags().BoolVarP(&o.BatchMode, "batch-mode", "b", false, "Enables batch mode")
	cmd.Flags().BoolVarP(&o.QuiteMode, "quiet", "q", false, "Do not fail if the namespace does not exist")
	cmd.Flags().BoolVarP(&o.PickEnv, "pick", "v", false, "Pick the Environment to switch to")
	cmd.Flags().StringVarP(&o.Env, "env", "e", "", "The Environment name to switch to the namepsace")
	return cmd, o
}

func (o *Options) Run() error {
	var err error
	currentNS := ""
	o.KubeClient, currentNS, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, "")
	if err != nil {
		return errors.Wrap(err, "creating kubernetes client")
	}
	client := o.KubeClient

	f := kubeclient.NewFactory()
	config, err := f.CreateKubeConfig()
	if err != nil {
		return errors.Wrap(err, "creating kubernetes configuration")
	}
	cfg, pathOptions, err := kubeclient.LoadConfig()
	if err != nil {
		return errors.Wrap(err, "loading Kubernetes configuration")
	}

	ns := ""
	if o.Env != "" || o.PickEnv {
		ns, err = o.findNamespaceFromEnv(currentNS, o.Env)
		if err != nil {
			return errors.Wrapf(err, "failed to find Jenkins X environment: %s", o.Env)
		}
		if ns == "" {
			return nil
		}
	}
	if ns == "" {
		ns = namespace(o)
	}

	if ns == "" && !o.BatchMode {
		ns, err = pickNamespace(o, client, currentNS)
		if err != nil {
			return err
		}
	}

	if ns != "" && ns != currentNS {
		ctx, err := changeNamespace(client, cfg, pathOptions, ns, o.Create, o.QuiteMode)
		if err != nil {
			return err
		}
		if ctx == nil {
			log.Logger().Infof("No kube context - probably in a unit test or pod?\n")
		} else {
			log.Logger().Infof("Now using namespace '%s' on server '%s'.\n", info(ctx.Namespace), info(kube.Server(cfg, ctx)))
		}
	} else {
		if currentNS != "" {
			ns = currentNS
		}
		server := kube.CurrentServer(cfg)
		if config == nil {
			log.Logger().Infof("Using namespace '%s' on server '%s'. No context - probably a unit test or pod?\n", info(ns), info(server))
		} else {
			log.Logger().Infof("Using namespace '%s' from context named '%s' on server '%s'.\n", info(ns), info(cfg.CurrentContext), info(server))
		}
	}
	return nil
}

func (o *Options) findNamespaceFromEnv(ns, name string) (string, error) {
	var err error
	o.JXClient, ns, err = jxclient.LazyCreateJXClientAndNamespace(o.JXClient, ns)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create jx client")
	}

	names, err := o.GetEnvironmentNames(ns)
	if err != nil {
		return "", err
	}

	if len(names) == 0 {
		// lets find the dev namespace to use that to find environments
		devNS := ""
		devNS, _, err = jxenv.GetDevNamespace(o.KubeClient, ns)
		if err != nil {
			return "", errors.Wrapf(err, "failed to find current dev namespace from %s", ns)
		}
		if devNS != ns {
			log.Logger().Infof("using the team namespace %s to find Environments", info(devNS))
			ns = devNS
			names, err = o.GetEnvironmentNames(ns)
			if err != nil {
				return "", err
			}
		}
	}
	if name == "" {
		name, err = o.pickName(names, "", "Pick environment:", "pick the kubernetes namespace for the current kubernetes cluster")
		if err != nil {
			return "", errors.Wrapf(err, "failed to pick environment")
		}
		if name == "" {
			return "", nil
		}
	}

	env, err := o.JXClient.JenkinsV1().Environments(ns).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", options.InvalidArg(name, names)
		}
		return "", errors.Wrapf(err, "failed to load Environment %s in namespace %s", name, ns)
	}
	return env.Spec.Namespace, nil
}

// GetEnvironmentNames returns the environment names in te given namespace
func (o *Options) GetEnvironmentNames(ns string) ([]string, error) {
	names, err := jxenv.GetEnvironmentNames(o.JXClient, ns)
	if err != nil && apierrors.IsNotFound(err) {
		err = nil
	}
	if err != nil {
		return names, errors.Wrapf(err, "failed to find Environment names in namespace %s", ns)
	}
	return names, err
}

func namespace(o *Options) string {
	ns := ""
	args := o.Args
	if len(args) > 0 {
		ns = args[0]
	}
	return ns
}

func changeNamespace(client kubernetes.Interface, config *api.Config, pathOptions clientcmd.ConfigAccess, ns string, create, quietMode bool) (*api.Context, error) {
	_, err := client.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
	if err != nil {
		switch err.(type) {
		case *apierrors.StatusError:
			err = handleStatusError(err, client, ns, create, quietMode)
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.Wrapf(err, "getting namespace %q", ns)
		}
	}
	ctx := kube.CurrentContext(config)

	if ctx == nil {
		ctx = &api.Context{}
		name := "pod"
		config.Contexts[name] = ctx
		config.CurrentContext = name
	}

	if ctx.Namespace == ns {
		return ctx, nil
	}
	ctx.Namespace = ns

	newConfig := *config
	err = clientcmd.ModifyConfig(pathOptions, newConfig, false)
	if err != nil {
		return nil, fmt.Errorf("failed to update the kube config %s", err)
	}
	return ctx, nil
}

func handleStatusError(err error, client kubernetes.Interface, ns string, create, quietMode bool) error {
	statusErr, _ := err.(*apierrors.StatusError)
	if statusErr.Status().Reason == metav1.StatusReasonNotFound {
		if quietMode {
			log.Logger().Infof("namespace %s does not exist yet", ns)
			os.Exit(0)
			return nil
		}
		if create {
			err = createNamespace(client, ns)
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	return err
}

func createNamespace(client kubernetes.Interface, ns string) error {
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
		},
	}
	_, err := client.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "unable to create namespace %s", ns)
	}
	return nil
}

func pickNamespace(o *Options, client kubernetes.Interface, defaultNamespace string) (string, error) {
	names, err := getNamespaceNames(client)
	if err != nil {
		return "", errors.Wrap(err, "retrieving namespace the names of the namespaces")
	}

	selectedNamespace, err := o.pickName(names, defaultNamespace, "Change namespace:", "pick the kubernetes namespace for the current kubernetes cluster")
	if err != nil {
		return "", errors.Wrap(err, "picking the namespace")
	}
	return selectedNamespace, nil
}

// getNamespaceNames returns the sorted list of environment names
func getNamespaceNames(client kubernetes.Interface) ([]string, error) {
	var names []string
	list, err := client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return names, fmt.Errorf("loading namespaces %s", err)
	}
	for k := range list.Items {
		names = append(names, list.Items[k].Name)
	}
	sort.Strings(names)
	return names, nil
}

func (o *Options) pickName(names []string, defaultValue, message, help string) (string, error) {
	if len(names) == 0 {
		return "", nil
	}
	if len(names) == 1 {
		return names[0], nil
	}
	if o.Input == nil {
		o.Input = survey.NewInput()
	}
	name, err := o.Input.PickNameWithDefault(names, message, defaultValue, help)
	return name, err
}

// IsInCluster tells if we are running incluster
func IsInCluster() bool {
	_, err := rest.InClusterConfig()
	return err == nil
}
