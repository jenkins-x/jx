/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package github

import (
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

// ValidateWebhook ensures that the provided request conforms to the
// format of a Github webhook and the payload can be validated with
// the provided hmac secret. It returns the event type, the event guid,
// the payload of the request, whether the webhook is valid or not,
// and finally the resultant HTTP status code
func ValidateWebhook(w http.ResponseWriter, r *http.Request, hmacSecret []byte) (string, string, []byte, bool, int) {
	defer r.Body.Close()

	// Our health check uses GET, so just kick back a 200.
	if r.Method == http.MethodGet {
		return "", "", nil, false, http.StatusOK
	}

	// Header checks: It must be a POST with an event type and a signature.
	if r.Method != http.MethodPost {
		responseHttpError(w, http.StatusMethodNotAllowed, "405 Method not allowed")
		return "", "", nil, false, http.StatusMethodNotAllowed
	}
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		responseHttpError(w, http.StatusBadRequest, "400 Bad Request: Missing X-GitHub-Event Header")
		return "", "", nil, false, http.StatusBadRequest
	}
	eventGUID := r.Header.Get("X-GitHub-Delivery")
	if eventGUID == "" {
		responseHttpError(w, http.StatusBadRequest, "400 Bad Request: Missing X-GitHub-Delivery Header")
		return "", "", nil, false, http.StatusBadRequest
	}
	sig := r.Header.Get("X-Hub-Signature")
	if sig == "" {
		responseHttpError(w, http.StatusForbidden, "403 Forbidden: Missing X-Hub-Signature")
		return "", "", nil, false, http.StatusForbidden
	}
	contentType := r.Header.Get("content-type")
	if contentType != "application/json" {
		responseHttpError(w, http.StatusBadRequest, "400 Bad Request: Hook only accepts content-type: application/json - please reconfigure this hook on GitHub")
		return "", "", nil, false, http.StatusBadRequest
	}
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		responseHttpError(w, http.StatusInternalServerError, "500 Internal Server Error: Failed to read request body")
		return "", "", nil, false, http.StatusInternalServerError
	}
	// Validate the payload with our HMAC secret.
	if !ValidatePayload(payload, sig, hmacSecret) {
		responseHttpError(w, http.StatusForbidden, "403 Forbidden: Invalid X-Hub-Signature")
		return "", "", nil, false, http.StatusForbidden
	}

	return eventType, eventGUID, payload, true, http.StatusOK
}

func responseHttpError(w http.ResponseWriter, statusCode int, response string) {
	logrus.WithFields(logrus.Fields{
		"response":    response,
		"status-code": statusCode,
	}).Debug(response)
	http.Error(w, response, statusCode)
}
