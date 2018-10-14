package accountv2

import (
	"encoding/json"
	"reflect"
	"strings"
)

type GenericPaginatedResourcesHandler struct {
	resourceType reflect.Type
}

func NewAccountPaginatedResources(resource interface{}) GenericPaginatedResourcesHandler {
	return GenericPaginatedResourcesHandler{
		resourceType: reflect.TypeOf(resource),
	}
}

func (pr GenericPaginatedResourcesHandler) Resources(bytes []byte, curURL string) ([]interface{}, string, error) {
	var paginatedResources = struct {
		NextUrl        string          `json:"next_url"`
		ResourcesBytes json.RawMessage `json:"resources"`
	}{}

	err := json.Unmarshal(bytes, &paginatedResources)

	slicePtr := reflect.New(reflect.SliceOf(pr.resourceType))
	dc := json.NewDecoder(strings.NewReader(string(paginatedResources.ResourcesBytes)))
	dc.UseNumber()
	err = dc.Decode(slicePtr.Interface())
	slice := reflect.Indirect(slicePtr)

	contents := make([]interface{}, 0, slice.Len())
	for i := 0; i < slice.Len(); i++ {
		contents = append(contents, slice.Index(i).Interface())
	}

	return contents, paginatedResources.NextUrl, err
}
