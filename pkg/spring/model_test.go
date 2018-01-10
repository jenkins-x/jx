package spring

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpringBootModel(t *testing.T) {
	model, err := LoadSpringBoot()
	assert.Nil(t, err)

	fmt.Printf("Loaded spring model %#v\n", model)

	//assert.Equal(t, "http://foo.bar/whatnot/thingy", UrlJoin("http://foo.bar", "whatnot", "thingy"))
}
