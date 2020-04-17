// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	jenkinsiov1 "github.com/jenkins-x/jx/v2/pkg/apis/jenkins.io/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFacts implements FactInterface
type FakeFacts struct {
	Fake *FakeJenkinsV1
	ns   string
}

var factsResource = schema.GroupVersionResource{Group: "jenkins.io", Version: "v1", Resource: "facts"}

var factsKind = schema.GroupVersionKind{Group: "jenkins.io", Version: "v1", Kind: "Fact"}

// Get takes name of the fact, and returns the corresponding fact object, and an error if there is any.
func (c *FakeFacts) Get(name string, options v1.GetOptions) (result *jenkinsiov1.Fact, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(factsResource, c.ns, name), &jenkinsiov1.Fact{})

	if obj == nil {
		return nil, err
	}
	return obj.(*jenkinsiov1.Fact), err
}

// List takes label and field selectors, and returns the list of Facts that match those selectors.
func (c *FakeFacts) List(opts v1.ListOptions) (result *jenkinsiov1.FactList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(factsResource, factsKind, c.ns, opts), &jenkinsiov1.FactList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &jenkinsiov1.FactList{ListMeta: obj.(*jenkinsiov1.FactList).ListMeta}
	for _, item := range obj.(*jenkinsiov1.FactList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested facts.
func (c *FakeFacts) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(factsResource, c.ns, opts))

}

// Create takes the representation of a fact and creates it.  Returns the server's representation of the fact, and an error, if there is any.
func (c *FakeFacts) Create(fact *jenkinsiov1.Fact) (result *jenkinsiov1.Fact, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(factsResource, c.ns, fact), &jenkinsiov1.Fact{})

	if obj == nil {
		return nil, err
	}
	return obj.(*jenkinsiov1.Fact), err
}

// Update takes the representation of a fact and updates it. Returns the server's representation of the fact, and an error, if there is any.
func (c *FakeFacts) Update(fact *jenkinsiov1.Fact) (result *jenkinsiov1.Fact, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(factsResource, c.ns, fact), &jenkinsiov1.Fact{})

	if obj == nil {
		return nil, err
	}
	return obj.(*jenkinsiov1.Fact), err
}

// Delete takes name of the fact and deletes it. Returns an error if one occurs.
func (c *FakeFacts) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(factsResource, c.ns, name), &jenkinsiov1.Fact{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFacts) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(factsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &jenkinsiov1.FactList{})
	return err
}

// Patch applies the patch and returns the patched fact.
func (c *FakeFacts) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *jenkinsiov1.Fact, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(factsResource, c.ns, name, data, subresources...), &jenkinsiov1.Fact{})

	if obj == nil {
		return nil, err
	}
	return obj.(*jenkinsiov1.Fact), err
}
