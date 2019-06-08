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

func TestGetManagedZoneName(t *testing.T) {
	t.Parallel()
	d := getManagedZoneName("wine.cheese.co.uk")
	assert.Equal(t, d, "wine-cheese-co-uk-zone")

	d = getManagedZoneName("planes.n.trains.com")
	assert.Equal(t, d, "planes-n-trains-com-zone")
}
