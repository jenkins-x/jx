package testing

import (
	"fmt"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/testhelper"
	fake "github.com/gophercloud/gophercloud/testhelper/client"
)

// ExtraSpecsGetBody provides a GET result of the extra_specs for a flavor
const ExtraSpecsGetBody = `
{
    "extra_specs" : {
        "hw:cpu_policy": "CPU-POLICY",
        "hw:cpu_thread_policy": "CPU-THREAD-POLICY"
    }
}
`

// ExtraSpecGetBody provides a GET result of a particular extra_spec for a flavor
const GetExtraSpecBody = `
{
    "hw:cpu_policy": "CPU-POLICY"
}
`

// ExtraSpecs is the expected extra_specs returned from GET on a flavor's extra_specs
var ExtraSpecs = map[string]string{
	"hw:cpu_policy":        "CPU-POLICY",
	"hw:cpu_thread_policy": "CPU-THREAD-POLICY",
}

// ExtraSpec is the expected extra_spec returned from GET on a flavor's extra_specs
var ExtraSpec = map[string]string{
	"hw:cpu_policy": "CPU-POLICY",
}

func HandleExtraSpecsListSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/flavors/1/os-extra_specs", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, ExtraSpecsGetBody)
	})
}

func HandleExtraSpecGetSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/flavors/1/os-extra_specs/hw:cpu_policy", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, GetExtraSpecBody)
	})
}

func HandleExtraSpecsCreateSuccessfully(t *testing.T) {
	th.Mux.HandleFunc("/flavors/1/os-extra_specs", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "X-Auth-Token", fake.TokenID)
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestJSONRequest(t, r, `{
				"extra_specs": {
					"hw:cpu_policy":        "CPU-POLICY",
					"hw:cpu_thread_policy": "CPU-THREAD-POLICY"
				}
			}`)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, ExtraSpecsGetBody)
	})
}
