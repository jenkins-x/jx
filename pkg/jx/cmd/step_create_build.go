package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
	"gopkg.in/AlecAivazis/survey.v1/terminal"

	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/jx/cmd/templates"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	createBuildLong = templates.LongDesc(`
		Creates a Knative build resource for a project
`)

	createBuildExample = templates.Examples(`
		# create a Knative build and render to the console
		jx step create build

		# create a Knative build
		jx step create build -o mybuild.yaml

			`)
)

// StepCreateBuildOptions contains the command line flags
type StepCreateBuildOptions struct {
	StepOptions

	Dir              string
	OutputDir        string
	OutputFilePrefix string
	BranchKind       string
	BuildNumber      int
}

// NewCmdCreateBuild Creates a new Command object
func NewCmdCreateBuild(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
	options := &StepCreateBuildOptions{
		StepOptions: StepOptions{
			CommonOptions: CommonOptions{
				Factory: f,
				In:      in,
				Out:     out,
				Err:     errOut,
			},
		},
	}

	cmd := &cobra.Command{
		Use:     "create build",
		Short:   "Creates a Knative build resource for a project",
		Long:    createBuildLong,
		Example: createBuildExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			CheckErr(err)
		},
	}
	options.addCommonFlags(cmd)

	cmd.Flags().StringVarP(&options.Dir, "dir", "d", "", "The directory to query to find the projects .git directory")
	cmd.Flags().StringVarP(&options.BranchKind, "kind", "k", "", "The kind of build such as 'release' or 'pullRequest' otherwise all of the builds are created")
	cmd.Flags().IntVarP(&options.BuildNumber, "build-number", "n", 1, "Which build number to use. <= 0 are ignored")
	cmd.Flags().StringVarP(&options.OutputDir, "output-dir", "o", "", "The directory where the generated build yaml files will be output to")
	cmd.Flags().StringVarP(&options.OutputFilePrefix, "output-prefix", "p", "build-", "The file name prefix used in the generated build files if output-dir is enabled")
	return cmd
}

// Run implements this command
func (o *StepCreateBuildOptions) Run() error {
	pc, _, err := config.LoadProjectConfig(o.Dir)
	if err != nil {
		return err
	}

	// TODO load the build pack jenkins-x to add any default build kinds?

	for _, branchBuild := range pc.Builds {
		if o.BranchKind != "" && branchBuild.Kind != o.BranchKind {
			continue
		}
		build, err := o.generateBuild(pc, branchBuild)
		if err != nil {
			return err
		}
		data, err := yaml.Marshal(build)
		if err != nil {
			return err
		}
		if data == nil {
			return fmt.Errorf("Could not marshal build to yaml")
		}

		outDir := o.OutputDir
		if outDir != "" {
			err = os.MkdirAll(outDir, DefaultWritePermissions)
			if err != nil {
				return err
			}
			output := filepath.Join(outDir, "build-"+branchBuild.Kind+".yml")
			err = ioutil.WriteFile(output, data, DefaultWritePermissions)
			if err != nil {
				return err
			}
		} else {
			log.Info(string(data))
		}
	}
	return err
}

func (o *StepCreateBuildOptions) generateBuild(projectConfig *config.ProjectConfig, build *config.BranchBuild) (*Build, error) {
	dir := o.Dir
	var err error
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	_, projectName := filepath.Split(dir)
	buildName := projectName
	buildNumber := o.BuildNumber
	if buildNumber > 0 {
		buildName = buildName + strconv.Itoa(buildNumber)
	}
	steps := []corev1.Container{}
	answer := &Build{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "build.knative.dev/v1alpha1",
			Kind:       "Build",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: kube.ToValidName(buildName),
		},
		Spec: BuildSpec{
			Steps: steps,
		},
	}

	// TODO load default steps from build pack?
	defaultImage := ""

	podTemplate, err := o.loadPodTemplate(projectConfig.BuildPack)
	if err != nil {
		return answer, err
	}
	for _, step := range build.Build.Steps {
		step2 := step
		if step2.Image == "" {
			step2.Image = defaultImage
		}
		if step2.Image == "" {
			buildPack := projectConfig.BuildPack
			if buildPack == "" {
				return answer, fmt.Errorf("No build pack defined in the configuration file: %s", config.ProjectConfigFileName)
			}
			containers := podTemplate.Spec.Containers
			if len(containers) > 0 {
				step2.Image = containers[0].Image
			}
			if step2.Image == "" {
				return answer, fmt.Errorf("No container image defined in the pod template for build pack %s", buildPack)
			}
		}
		if step2.Image != "" {
			defaultImage = step2.Image
		}

		err = o.addCommonSettings(&step2, projectConfig, build, podTemplate)
		if err != nil {
			return answer, err
		}

		steps = append(steps, step2)
	}
	answer.Spec.Steps = steps
	return answer, nil
}

func (o *StepCreateBuildOptions) loadPodTemplate(buildPack string) (*corev1.Pod, error) {
	if buildPack == "" {
		return nil, nil
	}
	answer := &corev1.Pod{}

	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		return answer, err
	}
	configMapName := kube.ConfigMapJenkinsPodTemplates
	cm, err := kubeClient.CoreV1().ConfigMaps(ns).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		return answer, errors.Wrapf(err, "failed to find ConfigMap %s in namespace %s", configMapName, ns)
	}

	podTemplateYaml := ""
	if cm.Data != nil {
		podTemplateYaml = cm.Data[buildPack]
	}
	if podTemplateYaml == "" {
		return answer, fmt.Errorf("No pod template is defiend in ConfigMap %s for build pack %s", configMapName, buildPack)
	}
	err = yaml.Unmarshal([]byte(podTemplateYaml), answer)
	return answer, err
}

func (o *StepCreateBuildOptions) addCommonSettings(container *corev1.Container, projectConfig *config.ProjectConfig, branchBuild *config.BranchBuild, podTemplate *corev1.Pod) error {
	build := &branchBuild.Build
	for _, env := range branchBuild.Env {
		if kube.GetEnvVar(container, env.Name) == nil {
			container.Env = append(container.Env, env)
		}
	}
	if podTemplate != nil {
		containers := podTemplate.Spec.Containers
		if len(containers) > 0 {
			c := containers[0]
			if !branchBuild.ExcludePodTemplateEnv {
				for _, env := range c.Env {
					if kube.GetEnvVar(container, env.Name) == nil {
						container.Env = append(c.Env, env)
					}
				}
			}
			if !branchBuild.ExcludePodTemplateVolumes {
				for _, v := range podTemplate.Spec.Volumes {
					if kube.GetVolume(&build.Volumes, v.Name) == nil {
						build.Volumes = append(build.Volumes, v)
					}
					for _, vm := range c.VolumeMounts {
						if vm.Name == v.Name {
							if kube.GetVolumeMount(&container.VolumeMounts, vm.Name) == nil {
								container.VolumeMounts = append(container.VolumeMounts, vm)
							}
						}
					}
				}
			}
		}
	}
	return nil
}

// TODO replace with the actual Knative build vendored ASAP!
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Build represents a build of a container image. A Build is made up of a
// source, and a set of steps. Steps can mount volumes to share data between
// themselves. A build may be created by instantiating a BuildTemplate.
type Build struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BuildSpec   `json:"spec"`
	Status BuildStatus `json:"status"`
}

// BuildSpec is the spec for a Build resource.
type BuildSpec struct {
	// TODO: Generation does not work correctly with CRD. They are scrubbed
	// by the APIserver (https://github.com/kubernetes/kubernetes/issues/58778)
	// So, we add Generation here. Once that gets fixed, remove this and use
	// ObjectMeta.Generation instead.
	// +optional
	Generation int64 `json:"generation,omitempty"`

	// Source specifies the input to the build.
	Source *SourceSpec `json:"source,omitempty"`

	// Steps are the steps of the build; each step is run sequentially with the
	// source mounted into /workspace.
	Steps []corev1.Container `json:"steps,omitempty"`

	// Volumes is a collection of volumes that are available to mount into the
	// steps of the build.
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// The name of the service account as which to run this build.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Template, if specified, references a BuildTemplate resource to use to
	// populate fields in the build, and optional Arguments to pass to the
	// template.
	Template *TemplateInstantiationSpec `json:"template,omitempty"`

	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// TemplateInstantiationSpec specifies how a BuildTemplate is instantiated into
// a Build.
type TemplateInstantiationSpec struct {
	// Name references the BuildTemplate resource to use.
	//
	// The template is assumed to exist in the Build's namespace.
	Name string `json:"name"`

	// Arguments, if specified, lists values that should be applied to the
	// parameters specified by the template.
	Arguments []ArgumentSpec `json:"arguments,omitempty"`

	// Env, if specified will provide variables to all build template steps.
	// This will override any of the template's steps environment variables.
	Env []corev1.EnvVar `json:"env,omitempty"`
}

// ArgumentSpec defines the actual values to use to populate a template's
// parameters.
type ArgumentSpec struct {
	// Name is the name of the argument.
	Name string `json:"name"`
	// Value is the value of the argument.
	Value string `json:"value"`
	// TODO(jasonhall): ValueFrom?
}

// SourceSpec defines the input to the Build
type SourceSpec struct {
	// Git represents source in a Git repository.
	Git *GitSourceSpec `json:"git,omitempty"`

	// GCS represents source in Google Cloud Storage.
	GCS *GCSSourceSpec `json:"gcs,omitempty"`

	// Custom indicates that source should be retrieved using a custom
	// process defined in a container invocation.
	Custom *corev1.Container `json:"custom,omitempty"`

	// SubPath specifies a path within the fetched source which should be
	// built. This option makes parent directories *inaccessible* to the
	// build steps. (The specific source type may, in fact, not even fetch
	// files not in the SubPath.)
	SubPath string `json:"subPath,omitempty"`
}

// GitSourceSpec describes a Git repo source input to the Build.
type GitSourceSpec struct {
	// URL of the Git repository to clone from.
	Url string `json:"url"`

	// Git revision (branch, tag, commit SHA or ref) to clone.  See
	// https://git-scm.com/docs/gitrevisions#_specifying_revisions for more
	// information.
	Revision string `json:"revision"`
}

// GCSSourceSpec describes source input to the Build in the form of an archive,
// or a source manifest describing files to fetch.
type GCSSourceSpec struct {
	// Type declares the style of source to fetch.
	Type GCSSourceType `json:"type,omitempty"`

	// Location specifies the location of the source archive or manifest file.
	Location string `json:"location,omitempty"`
}

// GCSSourceType defines a type of GCS source fetch.
type GCSSourceType string

const (
	// GCSArchive indicates that source should be fetched from a typical archive file.
	GCSArchive GCSSourceType = "Archive"

	// GCSManifest indicates that source should be fetched using a
	// manifest-based protocol which enables incremental source upload.
	GCSManifest GCSSourceType = "Manifest"
)

// BuildProvider defines a build execution implementation.
type BuildProvider string

const (
	// GoogleBuildProvider indicates that this build was performed with Google Cloud Build.
	GoogleBuildProvider BuildProvider = "Google"
	// ClusterBuildProvider indicates that this build was performed on-cluster.
	ClusterBuildProvider BuildProvider = "Cluster"
)

// BuildStatus is the status for a Build resource
type BuildStatus struct {
	Builder BuildProvider `json:"builder,omitempty"`

	// Cluster provides additional information if the builder is Cluster.
	Cluster *ClusterSpec `json:"cluster,omitempty"`
	// Google provides additional information if the builder is Google.
	Google *GoogleSpec `json:"google,omitempty"`

	// StartTime is the time the build started.
	StartTime metav1.Time `json:"startTime,omitEmpty"`
	// CompletionTime is the time the build completed.
	CompletionTime metav1.Time `json:"completionTime,omitEmpty"`

	// StepStates describes the state of each build step container.
	StepStates []corev1.ContainerState `json:"stepStates,omitEmpty"`
	// Conditions describes the set of conditions of this build.
	Conditions []BuildCondition `json:"conditions,omitempty"`

	StepsCompleted []string `json:"stepsCompleted"`
}

// ClusterSpec provides information about the on-cluster build, if applicable.
type ClusterSpec struct {
	// Namespace is the namespace in which the pod is running.
	Namespace string `json:"namespace"`
	// PodName is the name of the pod responsible for executing this build's steps.
	PodName string `json:"podName"`
}

// GoogleSpec provides information about the GCB build, if applicable.
type GoogleSpec struct {
	// Operation is the unique name of the GCB API Operation for the build.
	Operation string `json:"operation"`
}

// BuildConditionType defines types of build conditions.
type BuildConditionType string

// BuildSucceeded is set when the build is running, and becomes True when the
// build finishes successfully.
//
// If the build is ongoing, its status will be Unknown. If it fails, its status
// will be False.
const BuildSucceeded BuildConditionType = "Succeeded"

// BuildCondition defines a readiness condition for a Build.
// See: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#typical-status-properties
type BuildCondition struct {
	// Type is the type of the condition.
	Type BuildConditionType `json:"state"`

	// Status is one of True, False or Unknown.
	Status corev1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// Reason is a one-word CamelCase reason for the condition's last
	// transition.
	// +optional
	Reason string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`

	// Message is a human-readable message indicating details about the
	// last transition.
	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BuildList is a list of Build resources
type BuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items is the list of Build items in this list.
	Items []Build `json:"items"`
}

// GetCondition returns the Condition matching the given type.
func (bs *BuildStatus) GetCondition(t BuildConditionType) *BuildCondition {
	for _, cond := range bs.Conditions {
		if cond.Type == t {
			return &cond
		}
	}
	return nil
}

// SetCondition sets the condition, unsetting previous conditions with the same
// type as necessary.
func (b *BuildStatus) SetCondition(newCond *BuildCondition) {
	if newCond == nil {
		return
	}

	t := newCond.Type
	var conditions []BuildCondition
	for _, cond := range b.Conditions {
		if cond.Type != t {
			conditions = append(conditions, cond)
		}
	}
	conditions = append(conditions, *newCond)
	b.Conditions = conditions
}

// RemoveCondition removes any condition with the given type.
func (b *BuildStatus) RemoveCondition(t BuildConditionType) {
	var conditions []BuildCondition
	for _, cond := range b.Conditions {
		if cond.Type != t {
			conditions = append(conditions, cond)
		}
	}
	b.Conditions = conditions
}

// GetGeneration returns the generation number of this object.
func (b *Build) GetGeneration() int64 { return b.Spec.Generation }

// SetGeneration sets the generation number of this object.
func (b *Build) SetGeneration(generation int64) { b.Spec.Generation = generation }
