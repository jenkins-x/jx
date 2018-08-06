/*
Copyright 2016 The Kubernetes Authors.

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

import "regexp"

var lre = regexp.MustCompile(`<([^>]*)>; *rel="([^"]*)"`)

// Parse Link headers, returning a map from Rel to URL.
// Only understands the URI and "rel" parameter. Very limited.
// See https://tools.ietf.org/html/rfc5988#section-5
func parseLinks(h string) map[string]string {
	links := map[string]string{}
	for _, m := range lre.FindAllStringSubmatch(h, 10) {
		if len(m) != 3 {
			continue
		}
		links[m[2]] = m[1]
	}
	return links
}
