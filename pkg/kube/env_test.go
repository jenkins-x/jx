package kube

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestSortEnvironments(t *testing.T) {
	environments := []v1.Environment{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "c",
			},
			Spec: v1.EnvironmentSpec{
				Order: 100,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "z",
			},
			Spec: v1.EnvironmentSpec{
				Order: 5,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "d",
			},
			Spec: v1.EnvironmentSpec{
				Order: 100,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "a",
			},
			Spec: v1.EnvironmentSpec{
				Order: 150,
			},
		},
	}

	SortEnvironments(environments)

	assert.Equal(t, "z", environments[0].Name, "Environment 0")
	assert.Equal(t, "c", environments[1].Name, "Environment 1")
	assert.Equal(t, "d", environments[2].Name, "Environment 2")
	assert.Equal(t, "a", environments[3].Name, "Environment 3")
}

func TestSortEnvironments2(t *testing.T) {
	environments := []v1.Environment{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "dev",
			},
			Spec: v1.EnvironmentSpec{
				Order: 0,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "prod",
			},
			Spec: v1.EnvironmentSpec{
				Order: 200,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "staging",
			},
			Spec: v1.EnvironmentSpec{
				Order: 100,
			},
		},
	}

	SortEnvironments(environments)

	assert.Equal(t, "dev", environments[0].Name, "Environment 0")
	assert.Equal(t, "staging", environments[1].Name, "Environment 1")
	assert.Equal(t, "prod", environments[2].Name, "Environment 2")
}


func TestReplaceMakeVariable(t *testing.T) {
	lines := []string{"FOO", "NAMESPACE:=\"abc\"", "BAR", "NAMESPACE := \"abc\"", }

	actual := append([]string{}, lines...)
	expectedValue := "\"changed\""
	replaceMakeVariable(actual, "NAMESPACE", expectedValue)

	assert.Equal(t, "FOO", actual[0], "line 0")
	assert.Equal(t, "NAMESPACE := " + expectedValue, actual[1], "line 1")
	assert.Equal(t, "BAR", actual[2], "line 2")
	assert.Equal(t, "NAMESPACE := " + expectedValue, actual[3], "line 3")
}


