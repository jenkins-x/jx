package iks

import (
	"strconv"
	"strings"

	"github.com/IBM-Cloud/bluemix-go/api/container/containerv1"
)

func GetRegions(regions Regions) ([]string, error) {

	regionarr, err := regions.GetRegions()

	if err != nil {
		return nil, err
	}
	strregions := make([]string, len(regionarr))

	for i, region := range regionarr {
		strregions[i] = region.Name
	}

	return strregions, nil

}

func GetZones(region Region, zones Zones) ([]string, error) {
	zonearr, err := zones.GetZones(region)

	if err != nil {
		return nil, err
	}
	strzones := make([]string, len(zonearr))

	for i, zone := range zonearr {
		strzones[i] = zone.ID
	}

	return strzones, nil
}

func GetKubeVersions(versions containerv1.KubeVersions) ([]string, string, error) {
	target := containerv1.ClusterTargetHeader{}
	versionarr, err := versions.List(target)
	var def string

	if err != nil {
		return nil, "", err
	}

	strversions := make([]string, len(versionarr))

	for i, version := range versionarr {
		strversions[i] = strconv.Itoa(version.Major) + "." + strconv.Itoa(version.Minor) + "." + strconv.Itoa(version.Patch)
		if version.Default {
			def = strversions[i]
		}
	}
	return strversions, def, nil
}

func GetMachineTypes(zone Zone, machinetypes MachineTypes) ([]string, error) {
	machinetypesarr, err := machinetypes.GetMachineTypes(zone)

	if err != nil {
		return nil, err
	}
	strmachinetype := make([]string, len(machinetypesarr))

	for i, machinetype := range machinetypesarr {
		strmachinetype[i] = machinetype.Name
	}

	return strmachinetype, nil
}

func GetPrivateVLANs(zone Zone, vlans VLANs) ([]string, error) {
	vlansarr, err := vlans.GetVLANs(zone)

	if err != nil {
		return nil, err
	}
	strvlans := make([]string, len(vlansarr))

	var i = 0
	for _, vlan := range vlansarr {
		if strings.Compare(vlan.Type, "private") == 0 {
			strvlans[i] = vlan.ID
			i++
		}
	}

	return strvlans[0:i], nil
}

func GetPublicVLANs(zone Zone, vlans VLANs) ([]string, error) {
	vlansarr, err := vlans.GetVLANs(zone)

	if err != nil {
		return nil, err
	}
	strvlans := make([]string, len(vlansarr))

	var i = 0
	for _, vlan := range vlansarr {
		if strings.Compare(vlan.Type, "public") == 0 {
			strvlans[i] = vlan.ID
			i++
		}
	}

	return strvlans[0:i], nil
}
