package iks

import (
	"fmt"
	"strings"

	"github.com/IBM-Cloud/bluemix-go/client"
)

type Zone struct {
	ID    string
	Metro string
}

type Zones interface {
	GetZones(region Region) ([]Zone, error)
	GetZone(zone string, region Region) (*Zone, error)
}

type zone struct {
	*client.Client
	zones map[string][]Zone
}

func newZonesAPI(c *client.Client) Zones {
	return &zone{
		Client: c,
		zones:  nil,
	}
}

func (v *zone) fetch(region Region) error {
	if v.zones == nil {
		v.zones = make(map[string][]Zone)
	}
	if _, ok := v.zones[region.Name]; !ok {
		zones := []Zone{}
		m := make(map[string]string, 1)
		m["X-Region"] = region.Name
		_, err := v.Client.Get("/v1/zones", &zones, m)
		if err != nil {
			return err
		}
		v.zones[region.Name] = zones
	}
	return nil
}

func (v *zone) GetZones(region Region) ([]Zone, error) {
	if err := v.fetch(region); err != nil {
		return nil, err
	}
	return v.zones[region.Name], nil
}

func (v *zone) GetZone(zonearg string, region Region) (*Zone, error) {
	if err := v.fetch(region); err != nil {
		return nil, err
	}

	for _, zone := range v.zones[region.Name] {
		if strings.Compare(zonearg, zone.ID) == 0 {
			return &zone, nil
		}
	}
	return nil, fmt.Errorf("region %q not found in %q", zonearg, region.Name)
}
