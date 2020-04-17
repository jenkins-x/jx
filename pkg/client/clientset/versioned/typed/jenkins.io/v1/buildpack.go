// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/jenkins-x/jx/v2/pkg/apis/jenkins.io/v1"
	scheme "github.com/jenkins-x/jx/v2/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// BuildPacksGetter has a method to return a BuildPackInterface.
// A group's client should implement this interface.
type BuildPacksGetter interface {
	BuildPacks(namespace string) BuildPackInterface
}

// BuildPackInterface has methods to work with BuildPack resources.
type BuildPackInterface interface {
	Create(*v1.BuildPack) (*v1.BuildPack, error)
	Update(*v1.BuildPack) (*v1.BuildPack, error)
	Delete(name string, options *metav1.DeleteOptions) error
	DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error
	Get(name string, options metav1.GetOptions) (*v1.BuildPack, error)
	List(opts metav1.ListOptions) (*v1.BuildPackList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.BuildPack, err error)
	BuildPackExpansion
}

// buildPacks implements BuildPackInterface
type buildPacks struct {
	client rest.Interface
	ns     string
}

// newBuildPacks returns a BuildPacks
func newBuildPacks(c *JenkinsV1Client, namespace string) *buildPacks {
	return &buildPacks{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the buildPack, and returns the corresponding buildPack object, and an error if there is any.
func (c *buildPacks) Get(name string, options metav1.GetOptions) (result *v1.BuildPack, err error) {
	result = &v1.BuildPack{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("buildpacks").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of BuildPacks that match those selectors.
func (c *buildPacks) List(opts metav1.ListOptions) (result *v1.BuildPackList, err error) {
	result = &v1.BuildPackList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("buildpacks").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested buildPacks.
func (c *buildPacks) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("buildpacks").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a buildPack and creates it.  Returns the server's representation of the buildPack, and an error, if there is any.
func (c *buildPacks) Create(buildPack *v1.BuildPack) (result *v1.BuildPack, err error) {
	result = &v1.BuildPack{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("buildpacks").
		Body(buildPack).
		Do().
		Into(result)
	return
}

// Update takes the representation of a buildPack and updates it. Returns the server's representation of the buildPack, and an error, if there is any.
func (c *buildPacks) Update(buildPack *v1.BuildPack) (result *v1.BuildPack, err error) {
	result = &v1.BuildPack{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("buildpacks").
		Name(buildPack.Name).
		Body(buildPack).
		Do().
		Into(result)
	return
}

// Delete takes name of the buildPack and deletes it. Returns an error if one occurs.
func (c *buildPacks) Delete(name string, options *metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("buildpacks").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *buildPacks) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("buildpacks").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched buildPack.
func (c *buildPacks) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.BuildPack, err error) {
	result = &v1.BuildPack{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("buildpacks").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
