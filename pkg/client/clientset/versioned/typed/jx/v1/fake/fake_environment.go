package fake

import (
	jx_v1 "github.com/jenkins-x/jx/pkg/apis/jx/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeEnvironments implements EnvironmentInterface
type FakeEnvironments struct {
	Fake *FakeApiV1
	ns   string
}

var environmentsResource = schema.GroupVersionResource{Group: "api.jenkins.io", Version: "v1", Resource: "environments"}

var environmentsKind = schema.GroupVersionKind{Group: "api.jenkins.io", Version: "v1", Kind: "Environment"}

// Get takes name of the environment, and returns the corresponding environment object, and an error if there is any.
func (c *FakeEnvironments) Get(name string, options v1.GetOptions) (result *jx_v1.Environment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(environmentsResource, c.ns, name), &jx_v1.Environment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*jx_v1.Environment), err
}

// List takes label and field selectors, and returns the list of Environments that match those selectors.
func (c *FakeEnvironments) List(opts v1.ListOptions) (result *jx_v1.EnvironmentList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(environmentsResource, environmentsKind, c.ns, opts), &jx_v1.EnvironmentList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &jx_v1.EnvironmentList{}
	for _, item := range obj.(*jx_v1.EnvironmentList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested environments.
func (c *FakeEnvironments) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(environmentsResource, c.ns, opts))

}

// Create takes the representation of a environment and creates it.  Returns the server's representation of the environment, and an error, if there is any.
func (c *FakeEnvironments) Create(environment *jx_v1.Environment) (result *jx_v1.Environment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(environmentsResource, c.ns, environment), &jx_v1.Environment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*jx_v1.Environment), err
}

// Update takes the representation of a environment and updates it. Returns the server's representation of the environment, and an error, if there is any.
func (c *FakeEnvironments) Update(environment *jx_v1.Environment) (result *jx_v1.Environment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(environmentsResource, c.ns, environment), &jx_v1.Environment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*jx_v1.Environment), err
}

// Delete takes name of the environment and deletes it. Returns an error if one occurs.
func (c *FakeEnvironments) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(environmentsResource, c.ns, name), &jx_v1.Environment{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeEnvironments) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(environmentsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &jx_v1.EnvironmentList{})
	return err
}

// Patch applies the patch and returns the patched environment.
func (c *FakeEnvironments) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *jx_v1.Environment, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(environmentsResource, c.ns, name, data, subresources...), &jx_v1.Environment{})

	if obj == nil {
		return nil, err
	}
	return obj.(*jx_v1.Environment), err
}
