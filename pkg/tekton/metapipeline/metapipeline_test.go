package metapipeline

import (
	jenkinsv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/apps"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_empty_steps_if_app_does_not_define_pipeline_extension(t *testing.T) {
	testApp := jenkinsv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{apps.AppTypeLabel: apps.PipelineExtension.String()},
		},
		Spec: jenkinsv1.AppSpec{},
	}

	out := log.CaptureOutput(func() {
		steps, err := BuildExtensionSteps([]jenkinsv1.App{testApp})
		assert.NoError(t, err)
		assert.Empty(t, steps)
	})

	assert.Contains(t, out, "WARNING: Skipping app")
}

func Test_create_pipeline_steps_for_extending_app(t *testing.T) {
	testApp := jenkinsv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{apps.AppTypeLabel: apps.PipelineExtension.String()},
		},
		Spec: jenkinsv1.AppSpec{
			PipelineExtension: &jenkinsv1.PipelineExtension{
				Name:  "testapp",
				Image: "jenkinsxio/testapp",
			},
		},
	}

	out := log.CaptureOutput(func() {
		steps, err := BuildExtensionSteps([]jenkinsv1.App{testApp})
		assert.NoError(t, err)
		assert.Len(t, steps, 1)
	})

	assert.Empty(t, out)
}
