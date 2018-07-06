package gke

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetRegionFromZone(t *testing.T) {
	r := GetRegionFromZone("europe-west1-b")
	assert.Equal(t, r, "europe-west1")

	r = GetRegionFromZone("uswest1-d")
	assert.Equal(t, r, "uswest1")
}
