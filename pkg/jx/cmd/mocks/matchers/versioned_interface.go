// Code generated by pegomock. DO NOT EDIT.
package matchers

import (
	versioned "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/petergtz/pegomock"
	"reflect"
)

func AnyVersionedInterface() versioned.Interface {
	pegomock.RegisterMatcher(pegomock.NewAnyMatcher(reflect.TypeOf((*(versioned.Interface))(nil)).Elem()))
	var nullValue versioned.Interface
	return nullValue
}

func EqVersionedInterface(value versioned.Interface) versioned.Interface {
	pegomock.RegisterMatcher(&pegomock.EqMatcher{Value: value})
	var nullValue versioned.Interface
	return nullValue
}
