package amazon

import "os"

const DefaultRegion = "us-west-2"

func ResolveRegion() string {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
		if region == "" {
			region = DefaultRegion
		}
	}
	return region
}

func ResolveRegionIfOptionEmpty(regionOption string) string {
	if regionOption != "" {
		return regionOption
	}
	return ResolveRegion()
}