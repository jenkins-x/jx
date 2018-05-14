package azure

import "testing"

func TestGetRegistryName(t *testing.T) {
	input := "acrdevname.azurecr.io"
	expected := "acrdevname"
	output := getRegistryName(input)
	if output != expected {
		t.Errorf("getRegistryName: expected %v but got %v", expected, output)
	}
}
