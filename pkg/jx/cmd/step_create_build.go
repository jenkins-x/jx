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
	buildapi "github.com/knative/build/pkg/apis/build/v1alpha1"
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

// NewCmdStepCreateBuild Creates a new Command object
func NewCmdStepCreateBuild(f Factory, in terminal.FileReader, out terminal.FileWriter, errOut io.Writer) *cobra.Command {
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
		Use:     "build",
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

func (o *StepCreateBuildOptions) generateBuild(projectConfig *config.ProjectConfig, build *config.BranchBuild) (*buildapi.Build, error) {
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
	answer := &buildapi.Build{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "build.knative.dev/v1alpha1",
			Kind:       "Build",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: kube.ToValidName(buildName),
		},
		Spec: buildapi.BuildSpec{
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
