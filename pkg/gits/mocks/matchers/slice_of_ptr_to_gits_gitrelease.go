// Code generated by pegomock. DO NOT EDIT.
package matchers

import (
	"reflect"

	gits "github.com/jenkins-x/jx/v2/pkg/gits"
	"github.com/petergtz/pegomock"
)

func AnySliceOfPtrToGitsGitRelease() []*gits.GitRelease {
	pegomock.RegisterMatcher(pegomock.NewAnyMatcher(reflect.TypeOf((*([]*gits.GitRelease))(nil)).Elem()))
	var nullValue []*gits.GitRelease
	return nullValue
}

func EqSliceOfPtrToGitsGitRelease(value []*gits.GitRelease) []*gits.GitRelease {
	pegomock.RegisterMatcher(&pegomock.EqMatcher{Value: value})
	var nullValue []*gits.GitRelease
	return nullValue
}
