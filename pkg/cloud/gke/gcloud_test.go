package gke

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRegionFromZone(t *testing.T) {
	t.Parallel()
	r := GetRegionFromZone("europe-west1-b")
	assert.Equal(t, r, "europe-west1")

	r = GetRegionFromZone("uswest1-d")
	assert.Equal(t, r, "uswest1")
}
