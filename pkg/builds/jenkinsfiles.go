package builds

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/jenkins-x/jx/pkg/config"
	corev1 "k8s.io/api/core/v1"
)

type JenkinsConverter struct {
	Indentation          string
	KubernetesPluginMode bool
	ProjectConfig        *config.ProjectConfig

	indentCount int
	buffer      bytes.Buffer
	writer      *bufio.Writer
}

// NewJenkinsConverter creates a new JenkinsConverter instance
func NewJenkinsConverter(projectConfig *config.ProjectConfig) *JenkinsConverter {
	answer := &JenkinsConverter{
		ProjectConfig: projectConfig,
		Indentation:   "  ",
	}
	answer.writer = bufio.NewWriter(&answer.buffer)
	return answer
}

func (j *JenkinsConverter) ToJenkinsfile() (string, error) {
	projectConfig := j.ProjectConfig
	pack := projectConfig.BuildPack
	j.startBlock("pipeline")

	j.startBlock("agent")
	if pack != "" {
		j.println(fmt.Sprintf(`agent 'jenkins-%s'`, pack))
	}
	j.endBlock()

	j.environmentBlock(projectConfig.Env)

	j.startBlock("stages")

	for _, branchBuild := range projectConfig.Builds {
		kind := branchBuild.Kind
		build := &branchBuild.Build

		name := branchBuild.Name
		if name == "" {
			name = strings.Title(kind)
		}
		// TODO hack!
		branchPattern := "PR-*"

		j.startBlock(fmt.Sprintf(`stage '%s'`, name))

		if branchPattern != "" {
			j.startBlock("when")
			j.println(fmt.Sprintf(`branch '%s'`, branchPattern))
			j.endBlock()
		}
		j.environmentBlock(branchBuild.Env)

		j.startBlock("step")
		j.startContainer()
		for _, step := range build.Steps {
			cmd := strings.Join(step.Args, " ")
			j.println(fmt.Sprintf(`sh "%s"`, cmd))
		}
		j.endContainer()

		j.endBlock()

		j.endBlock()
	}
	j.endBlock()
	j.endBlock()
	return j.String(), nil
}

func (j *JenkinsConverter) startBlock(blockHeader string) {
	j.println(blockHeader + " {")
	j.indentCount++
}

func (j *JenkinsConverter) endBlock() {
	j.indentCount--
	j.println("}")
}

func (j *JenkinsConverter) writeIndent() {
	for i := 0; i < j.indentCount; i++ {
		j.writeString(j.Indentation)
	}
}

func (j *JenkinsConverter) println(text string) {
	j.writeIndent()
	j.writeString(text)
	j.writeString("\n")
}

func (j *JenkinsConverter) printf(format string, args ...interface{}) {
	j.writeIndent()
	j.writeString(fmt.Sprintf(format, args...))
}

func (j *JenkinsConverter) String() string {
	j.writer.Flush()
	return j.buffer.String()
}

func (j *JenkinsConverter) writeString(text string) {
	j.writer.WriteString(text)
}

func (j *JenkinsConverter) startContainer() {
	pack := j.ProjectConfig.BuildPack
	if j.KubernetesPluginMode && pack != "" {
		j.startBlock(fmt.Sprintf(`container('%s')`, pack))
	}
}

func (j *JenkinsConverter) endContainer() {
	pack := j.ProjectConfig.BuildPack
	if j.KubernetesPluginMode && pack != "" {
		j.endBlock()
	}
}

func (j *JenkinsConverter) environmentBlock(envVars []corev1.EnvVar) {
	if len(envVars) > 0 {
		j.startBlock("environment")
		for _, env := range envVars {
			if env.Value != "" {
				j.println(fmt.Sprintf(`%s = "%s"`, env.Name, env.Value))
			}
		}
		j.endBlock()
	}

}
