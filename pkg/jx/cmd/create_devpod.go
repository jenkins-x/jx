package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	optionLabel      = "label"
	optionRequestCpu = "request-cpu"
	devPodGoPath     = "/workspace"
)

var (
	createDevPodLong = templates.LongDesc(`
		Creates a new DevPod

		For more documentation see: [https://jenkins-x.io/developing/devpods/](https://jenkins-x.io/developing/devpods/)

`)

	createDevPodExample = templates.Examples(`
		# creates a new DevPod asking the user for the label to use
		jx create devpod

		# creates a new Maven DevPod 
		jx create devpod -l maven
	`)
)

// CreateDevPodResults the results of running the command
type CreateDevPodResults struct {
	TheaServiceURL string
	ExposeHost     string
}

// CreateDevPodOptions the options for the create spring command
type CreateDevPodOptions struct {
	CreateOptions

	Label      string
	Suffix     string
	WorkingDir string
	RequestCpu string
	Dir        string
	Reuse      bool
	Sync       bool
	Ports      []int
	AutoExpose bool
	Persist    bool
	ImportUrl  string
	Import     bool

	Results CreateDevPodResults
}

// NewCmdCreateDevPod creates a command object for the "create" command
func NewCmdCreateDevPod(f Factory, out io.Writer, errOut io.Writer) *cobra.Command {
	options := &CreateDevPodOptions{
		CreateOptions: CreateOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "devpod",
		Short:   "Creates a Developer Pod for running builds and tests inside the cluster",
		Aliases: []string{"dpod", "buildpod"},
		Long:    createDevPodLong,
		Example: createDevPodExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&options.Label, optionLabel, "l", "", "The label of the pod template to use")
	cmd.Flags().StringVarP(&options.Suffix, "suffix", "s", "", "The suffix to append the pod name")
	cmd.Flags().StringVarP(&options.WorkingDir, "working-dir", "w", "", "The working directory of the dev pod")
	cmd.Flags().StringVarP(&options.RequestCpu, optionRequestCpu, "c", "1", "The request CPU of the dev pod")
	cmd.Flags().BoolVarP(&options.Reuse, "reuse", "", true, "Reuse an existing DevPod if a suitable one exists. The DevPod will be selected based on the label (or current working directory)")
	cmd.Flags().BoolVarP(&options.Sync, "sync", "", false, "Also synchronise the local file system into the DevPod")
	cmd.Flags().IntSliceVarP(&options.Ports, "ports", "p", []int{}, "Container ports exposed by the DevPod")
	cmd.Flags().BoolVarP(&options.AutoExpose, "auto-expose", "", true, "Automatically expose useful ports as services such as the debug port, as well as any ports specified using --ports")
	cmd.Flags().BoolVarP(&options.Persist, "persist", "", false, "Persist changes made to the DevPod. Cannot be used with --sync")
	cmd.Flags().StringVarP(&options.ImportUrl, "import-url", "u", "", "Clone a Git repository into the DevPod. Cannot be used with --sync")
	cmd.Flags().BoolVarP(&options.Import, "import", "", true, "Detect if there is a Git repository in the current directory and attempt to clone it into the DevPod. Ignored if used with --sync")
	options.addCommonFlags(cmd)
	return cmd
}

// Run implements this command
func (o *CreateDevPodOptions) Run() error {

	addedServices := false

	if o.Persist && o.Sync {
		return errors.New("Cannot specify --persist and --sync")
	}

	if o.ImportUrl != "" && o.Sync {
		return errors.New("Cannot specify --import-url && --sync")
	}

	client, curNs, err := o.KubeClient()
	if err != nil {
		return err
	}
	ns, _, err := kube.GetDevNamespace(client, curNs)
	if err != nil {
		return err
	}
	u, err := user.Current()
	if err != nil {
		return err
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	cm, err := client.CoreV1().ConfigMaps(ns).Get(kube.ConfigMapJenkinsPodTemplates, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Failed to find ConfigMap %s in namespace %s: %s", kube.ConfigMapJenkinsPodTemplates, ns, err)
	}
	podTemplates := cm.Data
	labels := util.SortedMapKeys(podTemplates)

	label := o.Label
	if label == "" {
		label = o.guessDevPodLabel(dir, labels)
	}
	if label == "" {
		label, err = util.PickName(labels, "Pick which kind of dev pod you wish to create: ")
		if err != nil {
			return err
		}
	}
	yml := podTemplates[label]
	if yml == "" {
		return util.InvalidOption(optionLabel, label, labels)
	}

	editEnv, err := o.getOrCreateEditEnvironment()
	if err != nil {
		return err
	}

	pod := &corev1.Pod{}
	err = yaml.Unmarshal([]byte(yml), pod)
	if err != nil {
		return fmt.Errorf("Failed to parse Pod Template YAML: %s\n%s", err, yml)
	}
	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	userName := u.Username
	name := kube.ToValidName(userName + "-" + label)
	if o.Suffix != "" {
		name += "-" + o.Suffix
	}
	names, err := kube.GetPodNames(client, ns, "")
	if err != nil {
		return err
	}

	name = uniquePodName(names, name)

	pod.Name = name
	pod.Labels[kube.LabelPodTemplate] = label
	pod.Labels[kube.LabelDevPodName] = name
	pod.Labels[kube.LabelDevPodUsername] = userName

	if len(pod.Spec.Containers) == 0 {
		return fmt.Errorf("No containers specified for label %s with YAML: %s", label, yml)
	}
	container1 := &pod.Spec.Containers[0]

	workspaceVolumeName := "workspace-volume"
	// lets remove the default workspace volume as we don't need it
	for i, v := range pod.Spec.Volumes {
		if v.Name == workspaceVolumeName {
			pod.Spec.Volumes = append(pod.Spec.Volumes[:i], pod.Spec.Volumes[i+1:]...)
			break
		}
	}
	for ci, c := range pod.Spec.Containers {
		for i, v := range c.VolumeMounts {
			if v.Name == workspaceVolumeName {
				pod.Spec.Containers[ci].VolumeMounts = append(c.VolumeMounts[:i], c.VolumeMounts[i+1:]...)
				break
			}
		}
	}

	// Trying to reuse workspace-volume as a name seems to prevent us modifying the volumes!
	workspaceVolumeName = "ws-volume"
	var workspaceVolume corev1.Volume
	workspaceClaimName := fmt.Sprintf("%s-pvc", pod.Name)
	workspaceVolumeMount := corev1.VolumeMount{
		Name:      workspaceVolumeName,
		MountPath: "/workspace",
	}
	if o.Persist {
		workspaceVolume = corev1.Volume{
			Name: workspaceVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: workspaceClaimName,
				},
			},
		}
	} else {
		workspaceVolume = corev1.Volume{
			Name: workspaceVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
	}

	if !o.Sync {
		pod.Spec.Volumes = append(pod.Spec.Volumes, workspaceVolume)
		container1.VolumeMounts = append(container1.VolumeMounts, workspaceVolumeMount)

		cpuLimit, _ := resource.ParseQuantity("400m")
		cpuRequest, _ := resource.ParseQuantity("200m")
		memoryLimit, _ := resource.ParseQuantity("1Gi")
		memoryRequest, _ := resource.ParseQuantity("128Mi")

		// Add Theia - note Theia won't work in --sync mode as we can't share a volume
		theiaContainer := corev1.Container{
			Name:  "theia",
			Image: "theiaide/theia-full:latest",
			Ports: []corev1.ContainerPort{
				corev1.ContainerPort{
					ContainerPort: 3000,
				},
			},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					"cpu":    cpuLimit,
					"memory": memoryLimit,
				},
				Requests: corev1.ResourceList{
					"cpu":    cpuRequest,
					"memory": memoryRequest,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				workspaceVolumeMount,
			},
			LivenessProbe: &corev1.Probe{
				InitialDelaySeconds: 60,
				PeriodSeconds:       10,
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Port: intstr.FromInt(3000),
					},
				},
			},
			SecurityContext: &corev1.SecurityContext{
				RunAsUser: func(i int64) *int64 { return &i }(0),
			},
			Command: []string{"yarn", "theia", "start", "/workspace", "--hostname=0.0.0.0"},
		}

		pod.Spec.Containers = append(pod.Spec.Containers, theiaContainer)

	}

	if o.RequestCpu != "" {
		q, err := resource.ParseQuantity(o.RequestCpu)
		if err != nil {
			return util.InvalidOptionError(optionRequestCpu, o.RequestCpu, err)
		}
		container1.Resources.Requests[corev1.ResourceCPU] = q
	}

	workingDir := o.WorkingDir
	if workingDir == "" {
		workingDir = "/workspace"

		if o.Sync {
			// lets check for gopath stuff if we are in --sync mode so that we sync into gopath
			gopath := os.Getenv("GOPATH")
			if gopath != "" {
				rel, err := filepath.Rel(gopath, dir)
				if err == nil && rel != "" {
					workingDir = filepath.Join(devPodGoPath, rel)
				}
			}
		}
	}
	pod.Annotations[kube.AnnotationWorkingDir] = workingDir
	pod.Annotations[kube.AnnotationLocalDir] = dir
	container1.Env = append(container1.Env, corev1.EnvVar{
		Name:  "WORK_DIR",
		Value: workingDir,
	})
	container1.Stdin = true

	if editEnv != nil {
		container1.Env = append(container1.Env, corev1.EnvVar{
			Name:  "SKAFFOLD_DEPLOY_NAMESPACE",
			Value: editEnv.Spec.Namespace,
		})
	}
	// Assign the container the ports provided as input
	var exposeServicePorts []int

	for _, port := range o.Ports {
		cp := corev1.ContainerPort{
			Name:          fmt.Sprintf("port-%d", port),
			ContainerPort: int32(port),
		}
		container1.Ports = append(container1.Ports, cp)
	}

	// Assign the container the ports provided automatically
	if o.AutoExpose {
		exposeServicePorts = o.Ports
		if portsStr, ok := pod.Annotations["jenkins-x.io/devpodPorts"]; ok {
			ports := strings.Split(portsStr, ", ")
			for _, portStr := range ports {
				port, _ := strconv.Atoi(portStr)
				exposeServicePorts = append(exposeServicePorts, port)
				cp := corev1.ContainerPort{
					Name:          fmt.Sprintf("port-%d", port),
					ContainerPort: int32(port),
				}
				container1.Ports = append(container1.Ports, cp)
			}
		}
	}

	podResources := client.CoreV1().Pods(ns)

	create := true
	if o.Reuse {
		matchLabels := map[string]string{
			kube.LabelPodTemplate:    label,
			kube.LabelDevPodUsername: userName,
		}
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: matchLabels})
		if err != nil {
			return err
		}
		options := metav1.ListOptions{
			LabelSelector: selector.String(),
		}
		podsList, err := podResources.List(options)
		if err != nil {
			return err
		}
		for _, p := range podsList.Items {
			ann := p.Annotations
			if ann != nil && ann[kube.AnnotationLocalDir] == dir && p.DeletionTimestamp == nil {
				create = false
				pod = &p
				log.Infof("Reusing pod %s - waiting for it to be ready...\n", util.ColorInfo(pod.Name))
				break
			}
		}
	}

	theiaServiceName := name + "-theia"
	if create {
		log.Infof("Creating a dev pod of label: %s\n", util.ColorInfo(label))
		_, err = podResources.Create(pod)
		if err != nil {
			if o.Verbose {
				return fmt.Errorf("Failed to create pod %s\nYAML: %s", err, yml)
			} else {
				return fmt.Errorf("Failed to create pod %s", err)
			}
		}

		log.Infof("Created pod %s - waiting for it to be ready...\n", util.ColorInfo(name))

		// Get the pod UID
		pod, err = client.CoreV1().Pods(curNs).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// Create PVC if needed
		if o.Persist {
			storageRequest, _ := resource.ParseQuantity("2Gi")
			pvc := corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: workspaceClaimName,
					OwnerReferences: []metav1.OwnerReference{
						ownerRef(pod),
					},
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"storage": storageRequest,
						},
					},
				},
			}
			_, err = client.CoreV1().PersistentVolumeClaims(curNs).Create(&pvc)

			if err != nil {
				return err
			}
		}

		// Create services

		// Create a service for every port we expose

		if len(exposeServicePorts) > 0 {
			var servicePorts []corev1.ServicePort
			for _, port := range exposeServicePorts {
				portName := fmt.Sprintf("%s-%d", pod.Name, port)
				servicePorts = append(servicePorts, corev1.ServicePort{
					Name:       portName,
					Port:       int32(port),
					TargetPort: intstr.FromInt(port),
				})
			}

			service := corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"fabric8.io/expose": "true",
					},
					Name: pod.Name,
					OwnerReferences: []metav1.OwnerReference{
						ownerRef(pod),
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: servicePorts,
					Selector: map[string]string{
						"jenkins.io/devpod": pod.Name,
					},
				},
			}
			_, err = client.CoreV1().Services(curNs).Create(&service)

			if err != nil {
				return err
			}
			addedServices = true
		}
		if !o.Sync {

			// Create a service for theia
			theiaService := corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"fabric8.io/expose": "true",
					},
					Name: theiaServiceName,
					OwnerReferences: []metav1.OwnerReference{
						ownerRef(pod),
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						corev1.ServicePort{
							Name:       theiaServiceName,
							Port:       80,
							TargetPort: intstr.FromInt(3000),
						},
					},
					Selector: map[string]string{
						"jenkins.io/devpod": pod.Name,
					},
				},
			}
			_, err = client.CoreV1().Services(curNs).Create(&theiaService)
			if err != nil {
				return err
			}
			addedServices = true
		}

		if addedServices {
			err = o.updateExposeController(client, ns, ns)

			if err != nil {
				return err
			}
		}

		err = kube.WaitForPodNameToBeReady(client, ns, name, time.Hour)
		if err != nil {
			return err
		}

	}

	log.Infof("Pod %s is now ready!\n", util.ColorInfo(pod.Name))
	log.Infof("You can open other shells into this DevPod via %s\n", util.ColorInfo("jx create devpod --reuse"))

	theiaServiceURL, err := kube.FindServiceURL(client, curNs, theiaServiceName)
	if theiaServiceURL != "" {
		if err != nil {
			return err
		}
		log.Infof("You can edit your app using Theia (a browser based IDE) at %s\n", util.ColorInfo(theiaServiceURL))
		o.Results.TheaServiceURL = theiaServiceURL
	}

	exposePortsServiceHost, err := kube.FindServiceHostname(client, curNs, name)
	if err != nil {
		return err
	}
	if exposePortsServiceHost != "" {
		exposePortsService, err := client.CoreV1().Services(curNs).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		log.Infof("Ports %v are open on host %s\n", util.ColorInfo(exposePortsService.Spec.Ports), util.ColorInfo(exposePortsServiceHost))
		o.Results.ExposeHost = exposePortsServiceHost
	}

	if o.Sync {
		syncOptions := &SyncOptions{
			CommonOptions: o.CommonOptions,
			Namespace:     ns,
			Pod:           pod.Name,
			Daemon:        true,
			Dir:           dir,
		}
		err = syncOptions.CreateKsync(client, ns, pod.Name, dir, workingDir, userName)
		if err != nil {
			return err
		}
	}

	var rshExec []string
	if create {
		//  Let install bash-completion to make life better
		log.Infof("Installing Bash Completion into DevPod\n")
		rshExec = append(rshExec, "yum install -q -y bash-completion bash-completion-extra")
	}
	if !o.Sync {
		// Try to clone the right git repo into the DevPod

		// First configure git credentials
		rshExec = append(rshExec, "jx step git credentials", "git config --global credential.helper store")

		// We only honor --import if --sync is not specified
		if o.Import {
			var importUrl string
			if o.ImportUrl == "" {
				gitInfo, err := o.FindGitInfo(dir)
				if err != nil {
					return err
				}
				importUrl = gitInfo.HttpCloneURL()
			} else {
				importUrl = o.ImportUrl
			}
			if importUrl != "" {
				dir := regexp.MustCompile(`(?m)^.*/(.*)\.git$`).FindStringSubmatch(importUrl)[1]
				rshExec = append(rshExec, fmt.Sprintf("if ! [ -d \"%s\" ]; then git clone %s; fi", dir, importUrl))
				rshExec = append(rshExec, fmt.Sprintf("cd %s", dir))
			}
		}
	}
	rshExec = append(rshExec, defaultRshCommand)

	options := &RshOptions{
		CommonOptions: o.CommonOptions,
		Namespace:     ns,
		Pod:           pod.Name,
		DevPod:        true,
		ExecCmd:       strings.Join(rshExec, " && "),
	}
	options.Args = []string{}
	return options.Run()
}

func (o *CreateDevPodOptions) getOrCreateEditEnvironment() (*v1.Environment, error) {
	var env *v1.Environment
	apisClient, err := o.Factory.CreateApiExtensionsClient()
	if err != nil {
		return env, err
	}
	err = kube.RegisterEnvironmentCRD(apisClient)
	if err != nil {
		return env, err
	}

	kubeClient, _, err := o.KubeClient()
	if err != nil {
		return env, err
	}

	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return env, err
	}
	u, err := user.Current()
	if err != nil {
		return env, err
	}
	env, err = kube.EnsureEditEnvironmentSetup(kubeClient, jxClient, ns, u.Username)
	if err != nil {
		return env, err
	}
	// lets ensure that we've installed the exposecontroller service in the namespace
	var flag bool
	editNs := env.Spec.Namespace
	flag, err = kube.IsDeploymentRunning(kubeClient, kube.DeploymentExposecontrollerService, editNs)
	if !flag || err != nil {
		log.Infof("Installing the ExposecontrollerService in the namespace: %s\n", util.ColorInfo(editNs))
		releaseName := editNs + "-es"
		err = o.installChart(releaseName, kube.ChartExposecontrollerService, "", editNs, true, nil)
	}
	return env, err
}

func (o *CreateDevPodOptions) guessDevPodLabel(dir string, labels []string) string {
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		rel, err := filepath.Rel(gopath, dir)
		if err == nil && rel != "" && !strings.HasPrefix(rel, "../") {
			return "go"
		}
	}
	root, _, err := o.Git().FindGitConfigDir(o.Dir)
	if err != nil {
		log.Warnf("Could not find a .git directory: %s\n", err)
	}
	answer := ""
	if root != "" {
		jenkinsfile := filepath.Join(root, "Jenkinsfile")
		exists, err := util.FileExists(jenkinsfile)
		if err != nil {
			log.Warnf("Could not find a Jenkinsfile at %s: %s\n", jenkinsfile, err)
		} else if exists {
			answer, err = FindDevPodLabelFromJenkinsfile(jenkinsfile, labels)
			if err != nil {
				log.Warnf("Could not extract the pod template label from Jenkinsfile at %s: %s\n", jenkinsfile, err)
			}

		}
	}
	return answer
}

// updateExposeController lets update the exposecontroller to expose any new Service resources created for this devpod
func (o *CreateDevPodOptions) updateExposeController(client kubernetes.Interface, devNs string, ns string) error {
	ingressConfig, err := kube.GetIngressConfig(client, devNs)
	if err != nil {
		return errors.Wrapf(err, "Failed to load ingress-config in namespace %s", devNs)
	}
	return o.runExposecontroller(ns, ns, ingressConfig)
}

// FindDevPodLabelFromJenkinsfile finds pod labels from a Jenkinsfile
func FindDevPodLabelFromJenkinsfile(filename string, labels []string) (string, error) {
	answer := ""
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return answer, err
	}
	r, err := regexp.Compile(`label\s+\"(.+)\"`)
	if err != nil {
		return answer, err
	}

	jenkinsXLabelPrefix := "jenkins-"
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		text := strings.TrimSpace(line)
		arr := r.FindStringSubmatch(text)
		if len(arr) > 1 {
			a := arr[1]
			if a != "" {
				if util.StringArrayIndex(labels, a) >= 0 {
					return a, nil
				}
				if strings.HasPrefix(a, jenkinsXLabelPrefix) {
					a = strings.TrimPrefix(a, jenkinsXLabelPrefix)
					if util.StringArrayIndex(labels, a) >= 0 {
						return a, nil
					}
				}
				return answer, fmt.Errorf("Cannot find pipeline agent %s in the list of available DevPods: %s", a, strings.Join(labels, ", "))
			}
		}
	}
	return answer, nil
}

func uniquePodName(names []string, prefix string) string {
	count := 1
	for {
		name := prefix
		if count > 1 {
			name += strconv.Itoa(count)
		}
		if util.StringArrayIndex(names, name) < 0 {
			return name
		}
		count++
	}
}

func ownerRef(pod *corev1.Pod) metav1.OwnerReference {
	controller := true
	return metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       "Service",
		Name:       pod.Name,
		UID:        pod.UID,
		Controller: &controller,
	}
}
