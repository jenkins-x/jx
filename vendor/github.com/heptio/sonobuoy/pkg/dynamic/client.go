package dynamic

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// A scoped down meta.MetadataAccessor
type MetadataAccessor interface {
	Namespace(runtime.Object) (string, error)
	Name(runtime.Object) (string, error)
	ResourceVersion(runtime.Object) (string, error)
}

// A scoped down meta.RESTMapper
type mapper interface {
	RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error)
}

// APIHelper wraps the client-go dynamic client and exposes a simple interface.
type APIHelper struct {
	Client   dynamic.Interface
	Mapper   mapper
	Accessor MetadataAccessor
}

// NewAPIHelperFromRESTConfig creates a new APIHelper with default objects
// from client-go.
func NewAPIHelperFromRESTConfig(cfg *rest.Config) (*APIHelper, error) {
	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not create dynamic client")
	}
	discover, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "could not create discovery client")
	}
	groupResources, err := restmapper.GetAPIGroupResources(discover)
	if err != nil {
		return nil, errors.Wrap(err, "could not get api group resources")
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	return NewAPIHelper(dynClient, mapper, meta.NewAccessor())
}

// NewAPIHelper returns an APIHelper with the internals instantiated.
func NewAPIHelper(dyn dynamic.Interface, mapper mapper, accessor MetadataAccessor) (*APIHelper, error) {
	return &APIHelper{
		Client:   dyn,
		Mapper:   mapper,
		Accessor: accessor,
	}, nil
}

// CreateObject attempts to create any kubernetes object.
func (a *APIHelper) CreateObject(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	restMapping, err := a.Mapper.RESTMapping(obj.GroupVersionKind().GroupKind(), obj.GroupVersionKind().Version)
	if err != nil {
		return nil, errors.Wrap(err, "could not get restMapping")
	}
	name, err := a.Accessor.Name(obj)
	if err != nil {
		return nil, errors.Wrap(err, "could not get name for object")
	}
	namespace, err := a.Accessor.Namespace(obj)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't get namespace for object %s", name)
	}

	rsc := a.Client.Resource(restMapping.Resource)
	if rsc == nil {
		return nil, errors.New("failed to get a resource interface")
	}
	ri := rsc.Namespace(namespace)
	return ri.Create(obj)
}

// Name returns the name of the kubernetes object.
func (a *APIHelper) Name(obj *unstructured.Unstructured) (string, error) {
	return a.Accessor.Name(obj)
}

// Namespace returns the namespace of the kubernetes object.
func (a *APIHelper) Namespace(obj *unstructured.Unstructured) (string, error) {
	return a.Accessor.Namespace(obj)
}

// ResourceVersion returns the resource version of a kubernetes object.
func (a *APIHelper) ResourceVersion(obj *unstructured.Unstructured) (string, error) {
	return a.Accessor.ResourceVersion(obj)
}
