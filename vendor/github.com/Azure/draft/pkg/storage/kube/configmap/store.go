package configmap

import (
	"context"
	"time"

	"github.com/Azure/draft/pkg/storage"
	"github.com/golang/protobuf/ptypes"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// ConfigMaps represents a Kubernetes configmap storage engine for a storage.Object .
type ConfigMaps struct {
	impl corev1.ConfigMapInterface
}

// compile-time guarantee that *ConfigMaps implements storage.Store
var _ storage.Store = (*ConfigMaps)(nil)

// NewConfigMaps returns an implementation of storage.Store backed by kubernetes
// ConfigMap objects to store draft application build context.
func NewConfigMaps(impl corev1.ConfigMapInterface) *ConfigMaps {
	return &ConfigMaps{impl}
}

// DeleteBuilds deletes all draft builds for the application specified by appName.
//
// DeleteBuilds implements storage.Deleter.
func (s *ConfigMaps) DeleteBuilds(ctx context.Context, appName string) ([]*storage.Object, error) {
	builds, err := s.GetBuilds(ctx, appName)
	if err != nil {
		return nil, err
	}
	err = s.impl.Delete(appName, &metav1.DeleteOptions{})
	return builds, err
}

// DeleteBuild deletes the draft build given by buildID for the application specified by appName.
//
// DeleteBuild implements storage.Deleter.
func (s *ConfigMaps) DeleteBuild(ctx context.Context, appName, buildID string) (obj *storage.Object, err error) {
	var cfgmap *v1.ConfigMap
	if cfgmap, err = s.impl.Get(appName, metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, storage.NewErrAppStorageNotFound(appName)
		}
		return nil, err
	}
	if build, ok := cfgmap.Data[buildID]; ok {
		if obj, err = storage.DecodeString(build); err != nil {
			return nil, err
		}
		delete(cfgmap.Data, buildID)
		_, err = s.impl.Update(cfgmap)
		return obj, err
	}
	return nil, storage.NewErrAppBuildNotFound(appName, buildID)
}

// CreateBuild creates new storage for the application specified by appName to include build.
//
// If the configmap storage already exists for the application, ErrAppStorageExists is returned.
//
// CreateBuild implements storage.Creater.
func (s *ConfigMaps) CreateBuild(ctx context.Context, appName string, build *storage.Object) error {
	now, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		return err
	}
	build.CreatedAt = now

	cfgmap, err := newConfigMap(appName, build)
	if err != nil {
		return err
	}
	if _, err = s.impl.Create(cfgmap); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return storage.NewErrAppStorageExists(appName)
		}
		return err
	}
	return nil
}

// UpdateBuild updates the application configmap storage specified by appName to include build.
//
// If build does not exist, a new storage entry is created. Otherwise the existing storage
// is updated.
//
// UpdateBuild implements storage.Updater.
func (s *ConfigMaps) UpdateBuild(ctx context.Context, appName string, build *storage.Object) (err error) {
	var cfgmap *v1.ConfigMap
	if cfgmap, err = s.impl.Get(appName, metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return s.CreateBuild(ctx, appName, build)
		}
		return err
	}
	if _, ok := cfgmap.Data[build.BuildID]; ok {
		return storage.NewErrAppBuildExists(appName, build.BuildID)
	}
	if build.CreatedAt, err = ptypes.TimestampProto(time.Now()); err != nil {
		return err
	}
	content, err := storage.EncodeToString(build)
	if err != nil {
		return err
	}
	cfgmap.Data[build.BuildID] = content
	_, err = s.impl.Update(cfgmap)
	return err
}

// GetBuilds returns a slice of builds for the given app name.
//
// GetBuilds implements storage.Getter.
func (s *ConfigMaps) GetBuilds(ctx context.Context, appName string) (builds []*storage.Object, err error) {
	var cfgmap *v1.ConfigMap
	if cfgmap, err = s.impl.Get(appName, metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, storage.NewErrAppStorageNotFound(appName)
		}
		return nil, err
	}
	for _, obj := range cfgmap.Data {
		build, err := storage.DecodeString(obj)
		if err != nil {
			return nil, err
		}
		builds = append(builds, build)
	}
	return builds, nil
}

// GetBuild returns the build associated with buildID for the specified app name.
//
// GetBuild implements storage.Getter.
func (s *ConfigMaps) GetBuild(ctx context.Context, appName, buildID string) (obj *storage.Object, err error) {
	var cfgmap *v1.ConfigMap
	if cfgmap, err = s.impl.Get(appName, metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, storage.NewErrAppStorageNotFound(appName)
		}
		return nil, err
	}
	if data, ok := cfgmap.Data[buildID]; ok {
		if obj, err = storage.DecodeString(data); err != nil {
			return nil, err
		}
		return obj, nil
	}
	return nil, storage.NewErrAppBuildNotFound(appName, buildID)
}

// newConfigMap constructs a kubernetes ConfigMap object to store a build.
//
// Each configmap data entry is the base64 encoded string of a *storage.Object
// binary protobuf encoding.
func newConfigMap(appName string, build *storage.Object) (*v1.ConfigMap, error) {
	content, err := storage.EncodeToString(build)
	if err != nil {
		return nil, err
	}
	cfgmap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
			Labels: map[string]string{
				"heritage": "draft",
				"appname":  appName,
			},
		},
		Data: map[string]string{build.BuildID: content},
	}
	return cfgmap, nil
}
