package gke

func GetGoogleZones() []string {

	return []string{
		"asia-northeast1-a",
		"asia-northeast1-b",
		"asia-northeast1-c",
		"asia-southeast1-a",
		"asia-southeast1-b",
		"asia-east1-a",
		"asia-east1-b",
		"asia-east1-c",

		"australia-southeast1-a",
		"australia-southeast1-b",
		"australia-southeast1-c",

		"europe-west1-b",
		"europe-west1-c",
		"europe-west1-d",
		"europe-west2-a",
		"europe-west2-b",
		"europe-west2-c",
		"europe-west3-a",
		"europe-west3-b",
		"europe-west3-c",
		"europe-west4-b",
		"europe-west4-c",

		"northamerica-northeast1-a",
		"northamerica-northeast1-b",
		"northamerica-northeast1-c",

		"southamerica-east1-a",
		"southamerica-east1-a",
		"southamerica-east1-a",

		"us-west1-a",
		"us-west1-b",
		"us-west1-c",
		"us-east1-b",
		"us-east1-c",
		"us-east1-d",
		"us-central1-a",
		"us-central1-b",
		"us-central1-c",
		"us-central1-f",
		"us-east4-a",
		"us-east4-b",
		"us-east4-c",
	}
}

func GetGoogleMachineTypes() []string {

	return []string{
        "g1-small",
		"n1-standard-1",
		"n1-standard-2",
		"n1-standard-4",
		"n1-standard-8",
		"n1-standard-16",
		"n1-standard-32",
		"n1-standard-64",
		"n1-standard-96",
		"n1-highmem-2",
		"n1-highmem-4",
		"n1-highmem-8",
		"n1-highmem-16",
		"n1-highmem-32",
		"n1-highmem-64",
		"n1-highmem-96",
		"n1-highcpu-2",
		"n1-highcpu-4",
		"n1-highcpu-8",
		"n1-highcpu-16",
		"n1-highcpu-32",
		"n1-highcpu-64",
		"n1-highcpu-96",
	}
}
