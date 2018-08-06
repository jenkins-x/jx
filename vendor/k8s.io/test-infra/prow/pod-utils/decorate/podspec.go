/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package decorate

import (
	"fmt"
	"path"
	"sort"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/test-infra/prow/clonerefs"
	"k8s.io/test-infra/prow/entrypoint"
	"k8s.io/test-infra/prow/gcsupload"
	"k8s.io/test-infra/prow/initupload"
	"k8s.io/test-infra/prow/kube"
	"k8s.io/test-infra/prow/pod-utils/clone"
	"k8s.io/test-infra/prow/pod-utils/downwardapi"
	"k8s.io/test-infra/prow/pod-utils/wrapper"
	"k8s.io/test-infra/prow/sidecar"
)

const (
	logMountName            = "logs"
	logMountPath            = "/logs"
	artifactsEnv            = "ARTIFACTS"
	artifactsPath           = logMountPath + "/artifacts"
	codeMountName           = "code"
	codeMountPath           = "/home/prow/go"
	gopathEnv               = "GOPATH"
	toolsMountName          = "tools"
	toolsMountPath          = "/tools"
	gcsCredentialsMountName = "gcs-credentials"
	gcsCredentialsMountPath = "/secrets/gcs"
	sshKeysMountNamePrefix  = "ssh-keys"
	sshKeysMountPathPrefix  = "/secrets/ssh"
)

// Labels returns a string slice with label consts from kube.
func Labels() []string {
	return []string{kube.ProwJobTypeLabel, kube.CreatedByProw, kube.ProwJobIDLabel}
}

// VolumeMounts returns a string slice with *MountName consts in it.
func VolumeMounts() []string {
	return []string{logMountName, codeMountName, toolsMountName, gcsCredentialsMountName}
}

// VolumeMountPaths returns a string slice with *MountPath consts in it.
func VolumeMountPaths() []string {
	return []string{logMountPath, codeMountPath, toolsMountPath, gcsCredentialsMountPath}
}

// ProwJobToPod converts a ProwJob to a Pod that will run the tests.
func ProwJobToPod(pj kube.ProwJob, buildID string) (*v1.Pod, error) {
	if pj.Spec.PodSpec == nil {
		return nil, fmt.Errorf("prowjob %q lacks a pod spec", pj.Name)
	}

	rawEnv, err := downwardapi.EnvForSpec(downwardapi.NewJobSpec(pj.Spec, buildID, pj.Name))
	if err != nil {
		return nil, err
	}

	spec := pj.Spec.PodSpec.DeepCopy()
	spec.RestartPolicy = "Never"
	spec.Containers[0].Name = kube.TestContainerName

	if pj.Spec.DecorationConfig == nil {
		spec.Containers[0].Env = append(spec.Containers[0].Env, kubeEnv(rawEnv)...)
	} else {
		if err := decorate(spec, &pj, rawEnv); err != nil {
			return nil, fmt.Errorf("error decorating podspec: %v", err)
		}
	}

	podLabels := make(map[string]string)
	for k, v := range pj.ObjectMeta.Labels {
		podLabels[k] = v
	}
	podLabels[kube.CreatedByProw] = "true"
	podLabels[kube.ProwJobTypeLabel] = string(pj.Spec.Type)
	podLabels[kube.ProwJobIDLabel] = pj.ObjectMeta.Name
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   pj.ObjectMeta.Name,
			Labels: podLabels,
			Annotations: map[string]string{
				kube.ProwJobAnnotation: pj.Spec.Job,
			},
		},
		Spec: *spec,
	}, nil
}

func decorate(spec *kube.PodSpec, pj *kube.ProwJob, rawEnv map[string]string) error {
	rawEnv[artifactsEnv] = artifactsPath
	rawEnv[gopathEnv] = codeMountPath
	logMount := kube.VolumeMount{
		Name:      logMountName,
		MountPath: logMountPath,
	}
	logVolume := kube.Volume{
		Name: logMountName,
		VolumeSource: kube.VolumeSource{
			EmptyDir: &kube.EmptyDirVolumeSource{},
		},
	}

	codeMount := kube.VolumeMount{
		Name:      codeMountName,
		MountPath: codeMountPath,
	}
	codeVolume := kube.Volume{
		Name: codeMountName,
		VolumeSource: kube.VolumeSource{
			EmptyDir: &kube.EmptyDirVolumeSource{},
		},
	}

	toolsMount := kube.VolumeMount{
		Name:      toolsMountName,
		MountPath: toolsMountPath,
	}
	toolsVolume := kube.Volume{
		Name: toolsMountName,
		VolumeSource: kube.VolumeSource{
			EmptyDir: &kube.EmptyDirVolumeSource{},
		},
	}

	gcsCredentialsMount := kube.VolumeMount{
		Name:      gcsCredentialsMountName,
		MountPath: gcsCredentialsMountPath,
	}
	gcsCredentialsVolume := kube.Volume{
		Name: gcsCredentialsMountName,
		VolumeSource: kube.VolumeSource{
			Secret: &kube.SecretSource{
				SecretName: pj.Spec.DecorationConfig.GCSCredentialsSecret,
			},
		},
	}

	var sshKeysVolumes []kube.Volume
	var cloneLog string
	var refs []*kube.Refs
	if pj.Spec.Refs != nil {
		refs = append(refs, pj.Spec.Refs)
	}
	refs = append(refs, pj.Spec.ExtraRefs...)
	willCloneRefs := len(refs) > 0 && !pj.Spec.DecorationConfig.SkipCloning
	if willCloneRefs {
		var sshKeyMode int32 = 0400 // this is octal, so symbolic ref is `u+r`
		var sshKeysMounts []kube.VolumeMount
		var sshKeyPaths []string
		for _, secret := range pj.Spec.DecorationConfig.SSHKeySecrets {
			name := fmt.Sprintf("%s-%s", sshKeysMountNamePrefix, secret)
			keyPath := path.Join(sshKeysMountPathPrefix, secret)
			sshKeyPaths = append(sshKeyPaths, keyPath)
			sshKeysMounts = append(sshKeysMounts, kube.VolumeMount{
				Name:      name,
				MountPath: keyPath,
				ReadOnly:  true,
			})
			sshKeysVolumes = append(sshKeysVolumes, kube.Volume{
				Name: name,
				VolumeSource: kube.VolumeSource{
					Secret: &kube.SecretSource{
						SecretName:  secret,
						DefaultMode: &sshKeyMode,
					},
				},
			})
		}

		cloneLog = fmt.Sprintf("%s/clone.json", logMountPath)
		cloneConfigEnv, err := clonerefs.Encode(clonerefs.Options{
			SrcRoot:      codeMountPath,
			Log:          cloneLog,
			GitUserName:  clonerefs.DefaultGitUserName,
			GitUserEmail: clonerefs.DefaultGitUserEmail,
			GitRefs:      refs,
			KeyFiles:     sshKeyPaths,
		})
		if err != nil {
			return fmt.Errorf("could not encode clone configuration as JSON: %v", err)
		}

		spec.InitContainers = append(spec.InitContainers, kube.Container{
			Name:         "clonerefs",
			Image:        pj.Spec.DecorationConfig.UtilityImages.CloneRefs,
			Command:      []string{"/clonerefs"},
			Env:          kubeEnv(map[string]string{clonerefs.JSONConfigEnvVar: cloneConfigEnv}),
			VolumeMounts: append([]kube.VolumeMount{logMount, codeMount}, sshKeysMounts...),
		})
	}
	gcsOptions := gcsupload.Options{
		// TODO: pass the artifact dir here too once we figure that out
		GCSConfiguration:   pj.Spec.DecorationConfig.GCSConfiguration,
		GcsCredentialsFile: fmt.Sprintf("%s/service-account.json", gcsCredentialsMountPath),
		DryRun:             false,
	}

	initUploadOptions := initupload.Options{
		Options: &gcsOptions,
	}
	if willCloneRefs {
		initUploadOptions.Log = cloneLog
	}
	initUploadConfigEnv, err := initupload.Encode(initUploadOptions)
	if err != nil {
		return fmt.Errorf("could not encode initupload configuration as JSON: %v", err)
	}

	entrypointLocation := fmt.Sprintf("%s/entrypoint", toolsMountPath)

	spec.InitContainers = append(spec.InitContainers,
		kube.Container{
			Name:    "initupload",
			Image:   pj.Spec.DecorationConfig.UtilityImages.InitUpload,
			Command: []string{"/initupload"},
			Env: kubeEnv(map[string]string{
				initupload.JSONConfigEnvVar: initUploadConfigEnv,
				downwardapi.JobSpecEnv:      rawEnv[downwardapi.JobSpecEnv], // TODO: shouldn't need this?
			}),
			VolumeMounts: []kube.VolumeMount{logMount, gcsCredentialsMount},
		},
		kube.Container{
			Name:         "place-tools",
			Image:        pj.Spec.DecorationConfig.UtilityImages.Entrypoint,
			Command:      []string{"/bin/cp"},
			Args:         []string{"/entrypoint", entrypointLocation},
			VolumeMounts: []kube.VolumeMount{toolsMount},
		},
	)

	wrapperOptions := wrapper.Options{
		ProcessLog: fmt.Sprintf("%s/process-log.txt", logMountPath),
		MarkerFile: fmt.Sprintf("%s/marker-file.txt", logMountPath),
	}
	entrypointConfigEnv, err := entrypoint.Encode(entrypoint.Options{
		Args:        append(spec.Containers[0].Command, spec.Containers[0].Args...),
		Options:     &wrapperOptions,
		Timeout:     pj.Spec.DecorationConfig.Timeout,
		GracePeriod: pj.Spec.DecorationConfig.GracePeriod,
		ArtifactDir: artifactsPath,
	})
	if err != nil {
		return fmt.Errorf("could not encode entrypoint configuration as JSON: %v", err)
	}
	allEnv := rawEnv
	allEnv[entrypoint.JSONConfigEnvVar] = entrypointConfigEnv

	spec.Containers[0].Command = []string{entrypointLocation}
	spec.Containers[0].Args = []string{}
	spec.Containers[0].Env = append(spec.Containers[0].Env, kubeEnv(allEnv)...)
	spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts, logMount, toolsMount)

	gcsOptions.Items = append(gcsOptions.Items, artifactsPath)
	sidecarConfigEnv, err := sidecar.Encode(sidecar.Options{
		GcsOptions:     &gcsOptions,
		WrapperOptions: &wrapperOptions,
	})
	if err != nil {
		return fmt.Errorf("could not encode sidecar configuration as JSON: %v", err)
	}

	spec.Containers = append(spec.Containers, kube.Container{
		Name:    "sidecar",
		Image:   pj.Spec.DecorationConfig.UtilityImages.Sidecar,
		Command: []string{"/sidecar"},
		Env: kubeEnv(map[string]string{
			sidecar.JSONConfigEnvVar: sidecarConfigEnv,
			downwardapi.JobSpecEnv:   rawEnv[downwardapi.JobSpecEnv], // TODO: shouldn't need this?
		}),
		VolumeMounts: []kube.VolumeMount{logMount, gcsCredentialsMount},
	})
	spec.Volumes = append(spec.Volumes, logVolume, toolsVolume, gcsCredentialsVolume)

	if willCloneRefs {
		spec.Containers[0].WorkingDir = clone.PathForRefs(codeMountPath, refs[0])
		spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts, codeMount)
		spec.Volumes = append(spec.Volumes, append(sshKeysVolumes, codeVolume)...)
	}

	return nil
}

// kubeEnv transforms a mapping of environment variables
// into their serialized form for a PodSpec, sorting by
// the name of the env vars
func kubeEnv(environment map[string]string) []v1.EnvVar {
	var keys []string
	for key := range environment {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var kubeEnvironment []v1.EnvVar
	for _, key := range keys {
		kubeEnvironment = append(kubeEnvironment, v1.EnvVar{
			Name:  key,
			Value: environment[key],
		})
	}

	return kubeEnvironment
}
