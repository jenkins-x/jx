package inprocess

import (
	"context"

	"github.com/Azure/draft/pkg/storage"
)

// Store is an inprocess storage engine for draft.
type Store struct {
	// builds is mapping of app name to storage objects.
	builds map[string][]*storage.Object
}

// compile-time guarantee that *Store implements storage.Store
var _ storage.Store = (*Store)(nil)

// NewStore returns a new inprocess memory Store for storing draft application context.
func NewStore() *Store {
	return &Store{builds: make(map[string][]*storage.Object)}
}

// DeleteBuilds deletes all draft builds for the application specified by appName.
//
// DeleteBuilds implements storage.Deleter.
func (s *Store) DeleteBuilds(ctx context.Context, appName string) ([]*storage.Object, error) {
	h, ok := s.builds[appName]
	if !ok {
		return nil, storage.NewErrAppStorageNotFound(appName)
	}
	delete(s.builds, appName)
	return h, nil
}

// DeleteBuild deletes the draft build given by buildID for the application specified by appName.
//
// DeleteBuild implements storage.Deleter.
func (s *Store) DeleteBuild(ctx context.Context, appName, buildID string) (*storage.Object, error) {
	h, ok := s.builds[appName]
	if !ok {
		return nil, storage.NewErrAppStorageNotFound(appName)
	}
	for i, o := range h {
		if buildID == o.BuildID {
			s.builds[appName] = append(h[:i], h[i+1:]...)
			return o, nil
		}
	}
	return nil, storage.NewErrAppBuildNotFound(appName, buildID)
}

// CreateBuild creates new storage for the application specified by appName to include build.
//
// If storage already exists for the application, ErrAppStorageExists is returned.
//
// CreateBuild implements storage.Creater.
func (s *Store) CreateBuild(ctx context.Context, appName string, build *storage.Object) error {
	if _, ok := s.builds[appName]; ok {
		return storage.NewErrAppStorageExists(appName)
	}
	s.builds[appName] = []*storage.Object{build}
	return nil
}

// UpdateBuild updates the application storage specified by appName to include build.
//
// If build does not exist, a new storage entry is created. Otherwise the existing storage
// is updated.
//
// UpdateBuild implements storage.Updater.
func (s *Store) UpdateBuild(ctx context.Context, appName string, build *storage.Object) error {
	if _, ok := s.builds[appName]; !ok {
		return s.CreateBuild(ctx, appName, build)
	}
	s.builds[appName] = append(s.builds[appName], build)
	// TODO(fibonacci1729): deduplication of builds.
	return nil
}

// GetBuilds returns a slice of builds for the given app name.
//
// GetBuilds implements storage.Getter.
func (s *Store) GetBuilds(ctx context.Context, appName string) ([]*storage.Object, error) {
	h, ok := s.builds[appName]
	if !ok {
		return nil, storage.NewErrAppStorageNotFound(appName)
	}
	return h, nil
}

// GetBuild returns the build associated with buildID for the specified app name.
//
// GetBuild implements storage.Getter.
func (s *Store) GetBuild(ctx context.Context, appName, buildID string) (*storage.Object, error) {
	h, ok := s.builds[appName]
	if !ok {
		return nil, storage.NewErrAppStorageNotFound(appName)
	}
	for _, o := range h {
		if buildID == o.BuildID {
			return o, nil
		}
	}
	return nil, storage.NewErrAppBuildNotFound(appName, buildID)
}
