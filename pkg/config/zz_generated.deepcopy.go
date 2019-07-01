// +build !ignore_autogenerated

// Code generated by deepcopy-gen. DO NOT EDIT.

package config

import (
	jenkinsfile "github.com/jenkins-x/jx/pkg/jenkinsfile"
	v1 "k8s.io/api/core/v1"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AddonConfig) DeepCopyInto(out *AddonConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AddonConfig.
func (in *AddonConfig) DeepCopy() *AddonConfig {
	if in == nil {
		return nil
	}
	out := new(AddonConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdminSecretsConfig) DeepCopyInto(out *AdminSecretsConfig) {
	*out = *in
	if in.ChartMuseum != nil {
		in, out := &in.ChartMuseum, &out.ChartMuseum
		if *in == nil {
			*out = nil
		} else {
			*out = new(ChartMuseum)
			**out = **in
		}
	}
	if in.Grafana != nil {
		in, out := &in.Grafana, &out.Grafana
		if *in == nil {
			*out = nil
		} else {
			*out = new(Grafana)
			**out = **in
		}
	}
	if in.Jenkins != nil {
		in, out := &in.Jenkins, &out.Jenkins
		if *in == nil {
			*out = nil
		} else {
			*out = new(Jenkins)
			**out = **in
		}
	}
	if in.Nexus != nil {
		in, out := &in.Nexus, &out.Nexus
		if *in == nil {
			*out = nil
		} else {
			*out = new(Nexus)
			**out = **in
		}
	}
	if in.PipelineSecrets != nil {
		in, out := &in.PipelineSecrets, &out.PipelineSecrets
		if *in == nil {
			*out = nil
		} else {
			*out = new(PipelineSecrets)
			**out = **in
		}
	}
	if in.KanikoSecret != nil {
		in, out := &in.KanikoSecret, &out.KanikoSecret
		if *in == nil {
			*out = nil
		} else {
			*out = new(KanikoSecret)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdminSecretsConfig.
func (in *AdminSecretsConfig) DeepCopy() *AdminSecretsConfig {
	if in == nil {
		return nil
	}
	out := new(AdminSecretsConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdminSecretsFlags) DeepCopyInto(out *AdminSecretsFlags) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdminSecretsFlags.
func (in *AdminSecretsFlags) DeepCopy() *AdminSecretsFlags {
	if in == nil {
		return nil
	}
	out := new(AdminSecretsFlags)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AdminSecretsService) DeepCopyInto(out *AdminSecretsService) {
	*out = *in
	in.Secrets.DeepCopyInto(&out.Secrets)
	out.Flags = in.Flags
	out.ingressAuth = in.ingressAuth
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AdminSecretsService.
func (in *AdminSecretsService) DeepCopy() *AdminSecretsService {
	if in == nil {
		return nil
	}
	out := new(AdminSecretsService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BasicAuth) DeepCopyInto(out *BasicAuth) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BasicAuth.
func (in *BasicAuth) DeepCopy() *BasicAuth {
	if in == nil {
		return nil
	}
	out := new(BasicAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChartMuseum) DeepCopyInto(out *ChartMuseum) {
	*out = *in
	out.ChartMuseumEnv = in.ChartMuseumEnv
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChartMuseum.
func (in *ChartMuseum) DeepCopy() *ChartMuseum {
	if in == nil {
		return nil
	}
	out := new(ChartMuseum)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChartMuseumEnv) DeepCopyInto(out *ChartMuseumEnv) {
	*out = *in
	out.ChartMuseumSecret = in.ChartMuseumSecret
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChartMuseumEnv.
func (in *ChartMuseumEnv) DeepCopy() *ChartMuseumEnv {
	if in == nil {
		return nil
	}
	out := new(ChartMuseumEnv)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChartMuseumSecret) DeepCopyInto(out *ChartMuseumSecret) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChartMuseumSecret.
func (in *ChartMuseumSecret) DeepCopy() *ChartMuseumSecret {
	if in == nil {
		return nil
	}
	out := new(ChartMuseumSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ChatConfig) DeepCopyInto(out *ChatConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ChatConfig.
func (in *ChatConfig) DeepCopy() *ChatConfig {
	if in == nil {
		return nil
	}
	out := new(ChatConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnabledConfig) DeepCopyInto(out *EnabledConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnabledConfig.
func (in *EnabledConfig) DeepCopy() *EnabledConfig {
	if in == nil {
		return nil
	}
	out := new(EnabledConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EnvironmentConfig) DeepCopyInto(out *EnvironmentConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EnvironmentConfig.
func (in *EnvironmentConfig) DeepCopy() *EnvironmentConfig {
	if in == nil {
		return nil
	}
	out := new(EnvironmentConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExposeController) DeepCopyInto(out *ExposeController) {
	*out = *in
	out.Config = in.Config
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExposeController.
func (in *ExposeController) DeepCopy() *ExposeController {
	if in == nil {
		return nil
	}
	out := new(ExposeController)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExposeControllerConfig) DeepCopyInto(out *ExposeControllerConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExposeControllerConfig.
func (in *ExposeControllerConfig) DeepCopy() *ExposeControllerConfig {
	if in == nil {
		return nil
	}
	out := new(ExposeControllerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Grafana) DeepCopyInto(out *Grafana) {
	*out = *in
	out.GrafanaSecret = in.GrafanaSecret
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Grafana.
func (in *Grafana) DeepCopy() *Grafana {
	if in == nil {
		return nil
	}
	out := new(Grafana)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrafanaSecret) DeepCopyInto(out *GrafanaSecret) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrafanaSecret.
func (in *GrafanaSecret) DeepCopy() *GrafanaSecret {
	if in == nil {
		return nil
	}
	out := new(GrafanaSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelmValuesConfig) DeepCopyInto(out *HelmValuesConfig) {
	*out = *in
	if in.ExposeController != nil {
		in, out := &in.ExposeController, &out.ExposeController
		if *in == nil {
			*out = nil
		} else {
			*out = new(ExposeController)
			(*in).DeepCopyInto(*out)
		}
	}
	in.Jenkins.DeepCopyInto(&out.Jenkins)
	out.Prow = in.Prow
	out.PipelineSecrets = in.PipelineSecrets
	if in.ControllerBuild != nil {
		in, out := &in.ControllerBuild, &out.ControllerBuild
		if *in == nil {
			*out = nil
		} else {
			*out = new(EnabledConfig)
			**out = **in
		}
	}
	if in.ControllerWorkflow != nil {
		in, out := &in.ControllerWorkflow, &out.ControllerWorkflow
		if *in == nil {
			*out = nil
		} else {
			*out = new(EnabledConfig)
			**out = **in
		}
	}
	if in.DockerRegistryEnabled != nil {
		in, out := &in.DockerRegistryEnabled, &out.DockerRegistryEnabled
		if *in == nil {
			*out = nil
		} else {
			*out = new(EnabledConfig)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelmValuesConfig.
func (in *HelmValuesConfig) DeepCopy() *HelmValuesConfig {
	if in == nil {
		return nil
	}
	out := new(HelmValuesConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HelmValuesConfigService) DeepCopyInto(out *HelmValuesConfigService) {
	*out = *in
	in.Config.DeepCopyInto(&out.Config)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HelmValuesConfigService.
func (in *HelmValuesConfigService) DeepCopy() *HelmValuesConfigService {
	if in == nil {
		return nil
	}
	out := new(HelmValuesConfigService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Image) DeepCopyInto(out *Image) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Image.
func (in *Image) DeepCopy() *Image {
	if in == nil {
		return nil
	}
	out := new(Image)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IssueTrackerConfig) DeepCopyInto(out *IssueTrackerConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IssueTrackerConfig.
func (in *IssueTrackerConfig) DeepCopy() *IssueTrackerConfig {
	if in == nil {
		return nil
	}
	out := new(IssueTrackerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Jenkins) DeepCopyInto(out *Jenkins) {
	*out = *in
	out.JenkinsSecret = in.JenkinsSecret
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Jenkins.
func (in *Jenkins) DeepCopy() *Jenkins {
	if in == nil {
		return nil
	}
	out := new(Jenkins)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JenkinsAdminSecret) DeepCopyInto(out *JenkinsAdminSecret) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JenkinsAdminSecret.
func (in *JenkinsAdminSecret) DeepCopy() *JenkinsAdminSecret {
	if in == nil {
		return nil
	}
	out := new(JenkinsAdminSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JenkinsGiteaServersValuesConfig) DeepCopyInto(out *JenkinsGiteaServersValuesConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JenkinsGiteaServersValuesConfig.
func (in *JenkinsGiteaServersValuesConfig) DeepCopy() *JenkinsGiteaServersValuesConfig {
	if in == nil {
		return nil
	}
	out := new(JenkinsGiteaServersValuesConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JenkinsGithubServersValuesConfig) DeepCopyInto(out *JenkinsGithubServersValuesConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JenkinsGithubServersValuesConfig.
func (in *JenkinsGithubServersValuesConfig) DeepCopy() *JenkinsGithubServersValuesConfig {
	if in == nil {
		return nil
	}
	out := new(JenkinsGithubServersValuesConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JenkinsPipelineSecretsValuesConfig) DeepCopyInto(out *JenkinsPipelineSecretsValuesConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JenkinsPipelineSecretsValuesConfig.
func (in *JenkinsPipelineSecretsValuesConfig) DeepCopy() *JenkinsPipelineSecretsValuesConfig {
	if in == nil {
		return nil
	}
	out := new(JenkinsPipelineSecretsValuesConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JenkinsServersGlobalConfig) DeepCopyInto(out *JenkinsServersGlobalConfig) {
	*out = *in
	if in.EnvVars != nil {
		in, out := &in.EnvVars, &out.EnvVars
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JenkinsServersGlobalConfig.
func (in *JenkinsServersGlobalConfig) DeepCopy() *JenkinsServersGlobalConfig {
	if in == nil {
		return nil
	}
	out := new(JenkinsServersGlobalConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JenkinsServersValuesConfig) DeepCopyInto(out *JenkinsServersValuesConfig) {
	*out = *in
	if in.Gitea != nil {
		in, out := &in.Gitea, &out.Gitea
		*out = make([]JenkinsGiteaServersValuesConfig, len(*in))
		copy(*out, *in)
	}
	if in.GHE != nil {
		in, out := &in.GHE, &out.GHE
		*out = make([]JenkinsGithubServersValuesConfig, len(*in))
		copy(*out, *in)
	}
	in.Global.DeepCopyInto(&out.Global)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JenkinsServersValuesConfig.
func (in *JenkinsServersValuesConfig) DeepCopy() *JenkinsServersValuesConfig {
	if in == nil {
		return nil
	}
	out := new(JenkinsServersValuesConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *JenkinsValuesConfig) DeepCopyInto(out *JenkinsValuesConfig) {
	*out = *in
	in.Servers.DeepCopyInto(&out.Servers)
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		if *in == nil {
			*out = nil
		} else {
			*out = new(bool)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new JenkinsValuesConfig.
func (in *JenkinsValuesConfig) DeepCopy() *JenkinsValuesConfig {
	if in == nil {
		return nil
	}
	out := new(JenkinsValuesConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KanikoSecret) DeepCopyInto(out *KanikoSecret) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KanikoSecret.
func (in *KanikoSecret) DeepCopy() *KanikoSecret {
	if in == nil {
		return nil
	}
	out := new(KanikoSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Nexus) DeepCopyInto(out *Nexus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Nexus.
func (in *Nexus) DeepCopy() *Nexus {
	if in == nil {
		return nil
	}
	out := new(Nexus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PipelineSecrets) DeepCopyInto(out *PipelineSecrets) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PipelineSecrets.
func (in *PipelineSecrets) DeepCopy() *PipelineSecrets {
	if in == nil {
		return nil
	}
	out := new(PipelineSecrets)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Preview) DeepCopyInto(out *Preview) {
	*out = *in
	if in.Image != nil {
		in, out := &in.Image, &out.Image
		if *in == nil {
			*out = nil
		} else {
			*out = new(Image)
			**out = **in
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Preview.
func (in *Preview) DeepCopy() *Preview {
	if in == nil {
		return nil
	}
	out := new(Preview)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PreviewEnvironmentConfig) DeepCopyInto(out *PreviewEnvironmentConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PreviewEnvironmentConfig.
func (in *PreviewEnvironmentConfig) DeepCopy() *PreviewEnvironmentConfig {
	if in == nil {
		return nil
	}
	out := new(PreviewEnvironmentConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PreviewValuesConfig) DeepCopyInto(out *PreviewValuesConfig) {
	*out = *in
	if in.ExposeController != nil {
		in, out := &in.ExposeController, &out.ExposeController
		if *in == nil {
			*out = nil
		} else {
			*out = new(ExposeController)
			(*in).DeepCopyInto(*out)
		}
	}
	if in.Preview != nil {
		in, out := &in.Preview, &out.Preview
		if *in == nil {
			*out = nil
		} else {
			*out = new(Preview)
			(*in).DeepCopyInto(*out)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PreviewValuesConfig.
func (in *PreviewValuesConfig) DeepCopy() *PreviewValuesConfig {
	if in == nil {
		return nil
	}
	out := new(PreviewValuesConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProjectConfig) DeepCopyInto(out *ProjectConfig) {
	*out = *in
	if in.Env != nil {
		in, out := &in.Env, &out.Env
		*out = make([]v1.EnvVar, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PreviewEnvironments != nil {
		in, out := &in.PreviewEnvironments, &out.PreviewEnvironments
		if *in == nil {
			*out = nil
		} else {
			*out = new(PreviewEnvironmentConfig)
			**out = **in
		}
	}
	if in.IssueTracker != nil {
		in, out := &in.IssueTracker, &out.IssueTracker
		if *in == nil {
			*out = nil
		} else {
			*out = new(IssueTrackerConfig)
			**out = **in
		}
	}
	if in.Chat != nil {
		in, out := &in.Chat, &out.Chat
		if *in == nil {
			*out = nil
		} else {
			*out = new(ChatConfig)
			**out = **in
		}
	}
	if in.Wiki != nil {
		in, out := &in.Wiki, &out.Wiki
		if *in == nil {
			*out = nil
		} else {
			*out = new(WikiConfig)
			**out = **in
		}
	}
	if in.Addons != nil {
		in, out := &in.Addons, &out.Addons
		*out = make([]*AddonConfig, len(*in))
		for i := range *in {
			if (*in)[i] == nil {
				(*out)[i] = nil
			} else {
				(*out)[i] = new(AddonConfig)
				(*in)[i].DeepCopyInto((*out)[i])
			}
		}
	}
	if in.PipelineConfig != nil {
		in, out := &in.PipelineConfig, &out.PipelineConfig
		if *in == nil {
			*out = nil
		} else {
			*out = new(jenkinsfile.PipelineConfig)
			(*in).DeepCopyInto(*out)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProjectConfig.
func (in *ProjectConfig) DeepCopy() *ProjectConfig {
	if in == nil {
		return nil
	}
	out := new(ProjectConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProwValuesConfig) DeepCopyInto(out *ProwValuesConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProwValuesConfig.
func (in *ProwValuesConfig) DeepCopy() *ProwValuesConfig {
	if in == nil {
		return nil
	}
	out := new(ProwValuesConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RequirementsConfig) DeepCopyInto(out *RequirementsConfig) {
	*out = *in
	if in.Environments != nil {
		in, out := &in.Environments, &out.Environments
		*out = make([]EnvironmentConfig, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RequirementsConfig.
func (in *RequirementsConfig) DeepCopy() *RequirementsConfig {
	if in == nil {
		return nil
	}
	out := new(RequirementsConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WikiConfig) DeepCopyInto(out *WikiConfig) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WikiConfig.
func (in *WikiConfig) DeepCopy() *WikiConfig {
	if in == nil {
		return nil
	}
	out := new(WikiConfig)
	in.DeepCopyInto(out)
	return out
}
