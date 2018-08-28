package cmd

import (
	"io"
	"reflect"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// ControllerRoleOptions the command line options
type ControllerRoleOptions struct {
	ControllerOptions

	NoWatch bool

	Roles           map[string]*rbacv1.Role
	EnvRoleBindings map[string]*v1.EnvironmentRoleBinding
}

type ControllerRoleFlags struct {
	Version string
}

var (
	controllerRoleLong = templates.LongDesc(`
		Controller which replicas Role and EnvironmentRoleBinding resources to Roles and RoleBindings in all matching Environment namespaces. 

		RBAC in Kubernetes is either global with ClusterRoles or is namespace based with Roles per Namespace.

		We use a Custom Resource called EnvironmentRoleBinding which binds Roles and its bindings from the development 
		environment into each Environment's Namespace. 

		e.g. each EnvironmentRoleBinding will result in a RoleBinding and Role resource being create in each matching Environment. 
		So when a Preview environment is created it will have the correct Role and RoleBinding resources added. 

`)

	controllerRoleExample = templates.Examples(`
		# watch for changes in Role and EnvironmentRoleBindings in the dev namespace
		# and update the Role + RoleBinding resources in each environment namespace 
		jx controller role

		# update the current RoleBinding resources in each environment based on the current EnvironmentRoleBindings
		jx controller role --no-watch

`)
)

func NewCmdControllerRole(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := ControllerRoleOptions{
		ControllerOptions: ControllerOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}
	cmd := &cobra.Command{
		Use:     "role",
		Short:   "Controller which mirrors Role & EnvironmentRoleBinding resources to Roles and RoleBindings in all matching Environment namespaces",
		Long:    controllerRoleLong,
		Example: controllerRoleExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&options.NoWatch, "no-watch", "n", false, "To disable watching of the resources - to enable one-shot mode")
	return cmd
}

func (o *ControllerRoleOptions) Run() error {
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

	if !o.NoWatch {
		err = o.WatchRoles(kubeClient, ns)
		if err != nil {
			return err
		}
		err = o.WatchEnvironmentRoleBindings(jxClient, ns)
		if err != nil {
			return err
		}
		err = o.WatchEnvironments(kubeClient, jxClient, ns)
		if err != nil {
			return err
		}
	}

	roles, err := kubeClient.RbacV1().Roles(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, role := range roles.Items {
		o.upsertRole(&role)
	}
	bindings, err := jxClient.JenkinsV1().EnvironmentRoleBindings(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, binding := range bindings.Items {
		o.upsertEnvironmentRoleBinding(&binding)
	}
	envList, err := jxClient.JenkinsV1().Environments(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, env := range envList.Items {
		err = o.upsertEnvironment(kubeClient, ns, &env)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *ControllerRoleOptions) WatchRoles(kubeClient kubernetes.Interface, ns string) error {
	role := &rbacv1.Role{}
	listWatch := cache.NewListWatchFromClient(kubeClient.RbacV1().RESTClient(), "roles", ns, fields.Everything())
	kube.SortListWatchByName(listWatch)
	_, controller := cache.NewInformer(
		listWatch,
		role,
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				o.onRole(nil, obj)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				o.onRole(oldObj, newObj)
			},
			DeleteFunc: func(obj interface{}) {
				o.onRole(obj, nil)
			},
		},
	)

	stop := make(chan struct{})
	go controller.Run(stop)
	return nil
}

func (o *ControllerRoleOptions) WatchEnvironmentRoleBindings(jxClient versioned.Interface, ns string) error {
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

func (o *ControllerRoleOptions) WatchEnvironments(kubeClient kubernetes.Interface, jxClient versioned.Interface, ns string) error {
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

func (o *ControllerRoleOptions) onEnvironment(kubeClient kubernetes.Interface, ns string, oldObj interface{}, newObj interface{}) {
	var newEnv *v1.Environment
	if newObj != nil {
		newEnv = newObj.(*v1.Environment)
	}
	if oldObj != nil {
		oldEnv := oldObj.(*v1.Environment)
		if oldEnv != nil {
			if newEnv == nil || newEnv.Spec.Namespace != oldEnv.Spec.Namespace {
				err := o.removeEnvironment(kubeClient, ns, oldEnv)
				if err != nil {
					log.Warnf("Failed to remove role bindings for environment %s: %s", oldEnv.Name, err)
				}
			}
		}
	}
	if newEnv != nil {
		err := o.upsertEnvironment(kubeClient, ns, newEnv)
		if err != nil {
			log.Warnf("Failed to upsert role bindings for environment %s: %s", newEnv.Name, err)
		}
	}
}

func (o *ControllerRoleOptions) upsertEnvironment(kubeClient kubernetes.Interface, curNs string, env *v1.Environment) error {
	var answer error
	ns := env.Spec.Namespace
	if ns != "" {
		for _, binding := range o.EnvRoleBindings {
			if kube.EnvironmentMatchesAny(env, binding.Spec.Environments) {
				var err error
				if ns != curNs {
					roleName := binding.Spec.RoleRef.Name
					role := o.Roles[roleName]
					if role == nil {
						log.Warnf("Cannot find role %s in namespace %s", roleName, curNs)
					} else {
						roles := kubeClient.RbacV1().Roles(ns)
						var oldRole *rbacv1.Role
						oldRole, err = roles.Get(roleName, metav1.GetOptions{})
						if err == nil && oldRole != nil {
							// lets update it
							changed := false
							if !reflect.DeepEqual(oldRole.Rules, role.Rules) {
								oldRole.Rules = role.Rules
								changed = true
							}
							if changed {
								log.Infof("Updating Role %s in namespace %s\n", roleName, ns)
								_, err = roles.Update(oldRole)
							}
						} else {
							log.Infof("Creating Role %s in namespace %s\n", roleName, ns)
							newRole := &rbacv1.Role{
								ObjectMeta: metav1.ObjectMeta{
									Name: roleName,
									Labels: map[string]string{
										kube.LabelCreatedBy: kube.ValueCreatedByJX,
										kube.LabelTeam:      curNs,
									},
								},
								Rules: role.Rules,
							}
							_, err = roles.Create(newRole)
						}
					}
				}

				bindingName := binding.Name
				roleBindings := kubeClient.RbacV1().RoleBindings(ns)
				var old *rbacv1.RoleBinding
				old, err = roleBindings.Get(bindingName, metav1.GetOptions{})
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
						log.Infof("Updating RoleBinding %s in namespace %s\n", bindingName, ns)
						_, err = roleBindings.Update(old)
					}
				} else {
					log.Infof("Creating RoleBinding %s in namespace %s\n", bindingName, ns)
					newBinding := &rbacv1.RoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Name: bindingName,
							Labels: map[string]string{
								kube.LabelCreatedBy: kube.ValueCreatedByJX,
								kube.LabelTeam:      curNs,
							},
						},
						Subjects: binding.Spec.Subjects,
						RoleRef:  binding.Spec.RoleRef,
					}
					_, err = roleBindings.Create(newBinding)
				}
				if err != nil {
					log.Warnf("Failed: %s\n", err)
					if answer == nil {
						answer = err
					}
				}
			}
		}
	}
	return answer
}

func (o *ControllerRoleOptions) removeEnvironment(kubeClient kubernetes.Interface, curNs string, env *v1.Environment) error {
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

func (o *ControllerRoleOptions) onEnvironmentRoleBinding(oldObj interface{}, newObj interface{}) {
	if o.EnvRoleBindings == nil {
		o.EnvRoleBindings = map[string]*v1.EnvironmentRoleBinding{}
	}
	if oldObj != nil {
		oldEnv := oldObj.(*v1.EnvironmentRoleBinding)
		if oldEnv != nil {
			delete(o.EnvRoleBindings, oldEnv.Name)
		}
	}
	if newObj != nil {
		newEnv := newObj.(*v1.EnvironmentRoleBinding)
		o.upsertEnvironmentRoleBinding(newEnv)
	}
}

func (o *ControllerRoleOptions) upsertEnvironmentRoleBinding(newEnv *v1.EnvironmentRoleBinding) {
	if newEnv != nil {
		if o.EnvRoleBindings == nil {
			o.EnvRoleBindings = map[string]*v1.EnvironmentRoleBinding{}
		}
		o.EnvRoleBindings[newEnv.Name] = newEnv
	}
}

func (o *ControllerRoleOptions) onRole(oldObj interface{}, newObj interface{}) {
	if o.Roles == nil {
		o.Roles = map[string]*rbacv1.Role{}
	}
	if oldObj != nil {
		oldRole := oldObj.(*rbacv1.Role)
		if oldRole != nil {
			delete(o.Roles, oldRole.Name)
		}
	}
	if newObj != nil {
		newRole := newObj.(*rbacv1.Role)
		o.upsertRole(newRole)
	}
}

func (o *ControllerRoleOptions) upsertRole(newRole *rbacv1.Role) {
	if newRole != nil {
		if o.Roles == nil {
			o.Roles = map[string]*rbacv1.Role{}
		}
		o.Roles[newRole.Name] = newRole
	}
}
