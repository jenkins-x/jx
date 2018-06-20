package cmd

import (
	"fmt"
	"io"
	"reflect"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	cmdutil "github.com/jenkins-x/jx/pkg/jx/cmd/util"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/spf13/cobra"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// StepEnvRoleBindingOptions the command line options
type StepEnvRoleBindingOptions struct {
	StepOptions

	Watch bool

	EnvRoleBindings map[string]*v1.EnvironmentRoleBinding
}

type StepEnvRoleBindingFlags struct {
	Version string
}

var (
	stepEnvRoleBindingLong = templates.LongDesc(`
		This pipeline step command creates a git tag using a version number prefixed with 'v' and pushes it to a
		remote origin repo.

		This commands effectively runs:

		git commit -a -m "release $(VERSION)" --allow-empty
		git tag -fa v$(VERSION) -m "Release version $(VERSION)"
		git push origin v$(VERSION)

`)

	stepEnvRoleBindingExample = templates.Examples(`

		# update the current RoleBinding resources in each environment based on the current EnvironmentRoleBindings
		jx step envrolebinding

		# watch for changes in Environments and EnvironmentRoleBindings
		# and update the RoleBinding resources in each environment namespace 
		jx step envrolebinding -w
`)
)

func NewCmdStepEnvRoleBinding(f cmdutil.Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := StepEnvRoleBindingOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "tag",
		Short:   "Creates a git tag and pushes to remote repo",
		Long:    stepEnvRoleBindingLong,
		Example: stepEnvRoleBindingExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			cmdutil.CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&options.Watch, "watch", "w", false, "Whether to watch the Environments and EnvironmentRoleBindings for changes")
	return cmd
}

func (o *StepEnvRoleBindingOptions) Run() error {
	apiClient, err := o.CreateApiExtensionsClient()
	if err != nil {
		return err
	}
	err = kube.RegisterEnvironmentCRD(apiClient)
	if err != nil {
		return err
	}
	err = kube.RegisterEnvironmentRoleBindingCRD(apiClient)
	if err != nil {
		return err
	}

	jxClient, ns, err := o.JXClient()
	if err != nil {
		return err
	}

	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return err
	}

	if o.Watch {
		err = o.WatchEnvironmentRoleBindings(jxClient, ns)
		if err != nil {
			return err
		}
		err = o.WatchEnvironments(kubeClient, jxClient, ns)
		if err != nil {
			return err
		}
	}

	bindings, err := jxClient.JenkinsV1().EnvironmentRoleBindings(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("Found %d EnvironmentRoleBindings in namespace %s\n", len(bindings.Items), ns)
	for _, binding := range bindings.Items {
		o.upsertEnvironmentRoleBinding(&binding)
	}
	envList, err := jxClient.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("Found %d Environments in namespace %s\n", len(envList.Items), ns)
	for _, env := range envList.Items {
		err = o.upsertEnvironment(kubeClient, ns, &env)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *StepEnvRoleBindingOptions) WatchEnvironmentRoleBindings(jxClient versioned.Interface, ns string) error {
	environmentRoleBinding := &v1.EnvironmentRoleBinding{}
	listWatch := cache.NewListWatchFromClient(jxClient.JenkinsV1().RESTClient(), "environmentrolebindings", ns, fields.Everything())
	kube.SortListWatchByName(listWatch)
	_, controller := cache.NewInformer(
		listWatch,
		environmentRoleBinding,
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o.onEnvironmentRoleBinding(nil, obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o.onEnvironmentRoleBinding(oldObj, newObj)
			},
			DeleteFunc: func(obj interface{}) {
				o.onEnvironmentRoleBinding(obj, nil)
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)
	return nil
}

func (o *StepEnvRoleBindingOptions) WatchEnvironments(kubeClient kubernetes.Interface, jxClient versioned.Interface, ns string) error {
	environment := &v1.Environment{}
	listWatch := cache.NewListWatchFromClient(jxClient.JenkinsV1().RESTClient(), "environments", ns, fields.Everything())
	kube.SortListWatchByName(listWatch)
	_, controller := cache.NewInformer(
		listWatch,
		environment,
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o.onEnvironment(kubeClient, ns, nil, obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o.onEnvironment(kubeClient, ns, oldObj, newObj)
			},
			DeleteFunc: func(obj interface{}) {
				o.onEnvironment(kubeClient, ns, obj, nil)
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)

	// Wait forever
	select {}
}

func (o *StepEnvRoleBindingOptions) onEnvironment(kubeClient kubernetes.Interface, ns string, oldObj interface{}, newObj interface{}) {
	oldEnv := oldObj.(*v1.Environment)
	newEnv := newObj.(*v1.Environment)

	if oldEnv != nil {
		if newEnv == nil || newEnv.Spec.Namespace != oldEnv.Spec.Namespace {
			err := o.removeEnvironment(kubeClient, ns, oldEnv)
			if err != nil {
				o.warnf("Failed to remove role bindings for environment %s: %s", oldEnv.Name, err)
			}
		}
	}
	if newEnv != nil {
		err := o.upsertEnvironment(kubeClient, ns, newEnv)
		if err != nil {
			o.warnf("Failed to upsert role bindings for environment %s: %s", newEnv.Name, err)
		}
	}
}

func (o *StepEnvRoleBindingOptions) upsertEnvironment(kubeClient kubernetes.Interface, curNs string, env *v1.Environment) error {
	var answer error
	ns := env.Spec.Namespace
	if ns != "" {
		for _, binding := range o.EnvRoleBindings {
			if kube.EnvironmentMatchesAny(env, binding.Spec.Environments) {
				var err error
				name := binding.Name
				roleBindings := kubeClient.RbacV1().RoleBindings(ns)
				old, err := roleBindings.Get(name, metav1.GetOptions{})
				if err == nil && old != nil {
					// lets update it
					changed := false

					if !reflect.DeepEqual(old.RoleRef, binding.Spec.RoleRef) {
						old.RoleRef = binding.Spec.RoleRef
						changed = true
					}
					if !reflect.DeepEqual(old.Subjects, binding.Spec.Subjects) {
						old.Subjects = binding.Spec.Subjects
						changed = true
					}
					if changed {
						o.Printf("Updating RoleBinding %s in namespace %s\n", name, ns)
						_, err = roleBindings.Update(old)
					}
				} else {
					o.Printf("Creating RoleBinding %s in namespace %s\n", name, ns)
					newBinding := &rbacv1.RoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Name: name,
						},
						Subjects: binding.Spec.Subjects,
						RoleRef:  binding.Spec.RoleRef,
					}
					_, err = roleBindings.Create(newBinding)
				}
				if err != nil {
					o.warnf("Failed: %s\n", err)
					if answer == nil {
						answer = err
					}
				}
			}
		}
	}
	return answer
}

func (o *StepEnvRoleBindingOptions) removeEnvironment(kubeClient kubernetes.Interface, curNs string, env *v1.Environment) error {
	ns := env.Spec.Namespace
	if ns != "" {
		for _, binding := range o.EnvRoleBindings {
			if kube.EnvironmentMatchesAny(env, binding.Spec.Environments) {
				// ignore errors
				kubeClient.RbacV1().RoleBindings(ns).Delete(binding.Name, nil)
			}
		}
	}
	return nil
}

func (o *StepEnvRoleBindingOptions) onEnvironmentRoleBinding(oldObj interface{}, newObj interface{}) {
	oldEnv := oldObj.(*v1.EnvironmentRoleBinding)
	newEnv := newObj.(*v1.EnvironmentRoleBinding)

	if o.EnvRoleBindings == nil {
		o.EnvRoleBindings = map[string]*v1.EnvironmentRoleBinding{}
	}
	if oldEnv != nil {
		delete(o.EnvRoleBindings, oldEnv.Name)
	}
	o.upsertEnvironmentRoleBinding(newEnv)
}

func (o *StepEnvRoleBindingOptions) upsertEnvironmentRoleBinding(newEnv *v1.EnvironmentRoleBinding) {
	if newEnv != nil {
		if o.EnvRoleBindings == nil {
			o.EnvRoleBindings = map[string]*v1.EnvironmentRoleBinding{}
		}
		o.EnvRoleBindings[newEnv.Name] = newEnv
	}
}
