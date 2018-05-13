package util

import (
	"net/http"
)

func handleErr(request *http.Request, response http.ResponseWriter, err error) {
	response.WriteHeader(http.StatusInternalServerError)
	response.Write([]byte(err.Error()))
}

func handleOk(response http.ResponseWriter, body []byte) {
	response.WriteHeader(http.StatusOK)
	response.Write(body)
}

type MethodMap map[string]string
type Router map[string]MethodMap

// Are you a mod or a rocker? I'm a
type mocker func(http.ResponseWriter, *http.Request)

// @param dataDir Location of test data json file
// @param router  Should map a URL path to a map that maps a method to a JSON response file name. Conceptually: (url, method) -> file
// See pkg/gits/bitbucket_test.go for an example.
func GetMockAPIResponseFromFile(dataDir string, route MethodMap) mocker {

	return func(response http.ResponseWriter, request *http.Request) {
		fileName := route[request.Method]

		obj, err := LoadBytes(dataDir, fileName)

		if err != nil {
			handleErr(request, response, err)
		}

		handleOk(response, obj)
	}
}
