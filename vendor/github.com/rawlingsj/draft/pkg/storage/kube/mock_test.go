package kube

import (
	"testing"

	"k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/api"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/Azure/draft/pkg/storage"
)

// MockConfigMaps mocks a kubernetes ConfigMapsInterface.
//
// For use in testing only.
type MockConfigMaps struct {
	corev1.ConfigMapInterface
	cfgmaps map[string]*v1.ConfigMap
}

// NewConfigMapsWithMocks initializes a new ConfigMaps store initialized
// with kubernetes ConfigMap objects created from the provided entries.
func NewConfigMapsWithMocks(t *testing.T, entries ...struct {
	appName string
	objects []*storage.Object
}) *ConfigMaps {
	var mock MockConfigMaps
	mock.Init(t, entries...)
	return NewConfigMaps(&mock)
}

// Init initializes the MockConfigMaps mock with the set of storage objects.
func (mock *MockConfigMaps) Init(t *testing.T, entries ...struct {
	appName string
	objects []*storage.Object
}) {
	mock.cfgmaps = make(map[string]*v1.ConfigMap)
	for _, entry := range entries {
		var cfgmap *v1.ConfigMap
		for _, object := range entry.objects {
			if cfgmap != nil {
				if _, ok := cfgmap.Data[object.BuildID]; !ok {
					content, err := storage.EncodeToString(object)
					if err != nil {
						t.Fatalf("failed to encode storage object: %v", err)
					}
					cfgmap.Data[object.BuildID] = content
				}
			} else {
				var err error
				if cfgmap, err = newConfigMap(entry.appName, object); err != nil {
					t.Fatalf("failed to create configmap: %v", err)
				}
			}
		}
		mock.cfgmaps[entry.appName] = cfgmap
	}
}

// Get returns the ConfigMap by name.
func (mock *MockConfigMaps) Get(name string, options metav1.GetOptions) (*v1.ConfigMap, error) {
	cfgmap, ok := mock.cfgmaps[name]
	if !ok {
		return nil, apierrors.NewNotFound(api.Resource("tests"), name)
	}
	return cfgmap, nil
}

// Create creates a new ConfigMap.
func (mock *MockConfigMaps) Create(cfgmap *v1.ConfigMap) (*v1.ConfigMap, error) {
	name := cfgmap.ObjectMeta.Name
	if object, ok := mock.cfgmaps[name]; ok {
		return object, apierrors.NewAlreadyExists(api.Resource("tests"), name)
	}
	mock.cfgmaps[name] = cfgmap
	return cfgmap, nil
}

// Update updates a ConfigMap.
func (mock *MockConfigMaps) Update(cfgmap *v1.ConfigMap) (*v1.ConfigMap, error) {
	name := cfgmap.ObjectMeta.Name
	mock.cfgmaps[name] = cfgmap
	return cfgmap, nil
}

// Delete deletes a ConfigMap by name.
func (mock *MockConfigMaps) Delete(name string, opts *metav1.DeleteOptions) error {
	if _, ok := mock.cfgmaps[name]; !ok {
		return apierrors.NewNotFound(api.Resource("tests"), name)
	}
	delete(mock.cfgmaps, name)
	return nil
}
