package cmd_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jenkins-x/jx/pkg/cmd"
	"github.com/stretchr/testify/assert"
)

const text = `{"params":{"fin":false,"headers":[":status: 302","server: nginx/1.13.12","date: Mon, 10 Sep 2018 12:40:13 GMT","content-type: text/html; charset=utf-8","content-length: 24","location: /","set-cookie: sso-core_csrf=; Path=/; Domain=cd.jenkins-x.io; Expires=Mon, 10 Sep 2018 11:40:13 GMT; HttpOnly; Secure\u0000sso-core=ZW1haWw6Y29zbWluLmNvam9jYXJAZ214LmNoIHVzZXI6fGJiZW9heG1ycFdWMCs2VGNjSVlOWTlOTEx3cmxJcCtIMnF1MzgxQzkxc2FqL3R2SUR6enlDRE1YNmJCY2ZxSG9PNiszMDRnY1RFSDgvNUY1NlMwR21kMDRXcyt6KzQyVllXbVcraGJqN25rRzFtc3JEMHNsa3J3U1pLeVJrZmJBTkJsS0NJd0QzUFlUcUxsbjJEL09Kdng5VUpoSHFsSy93ZERGUFhRNDl6T3NPUlRHUm0wS1Y5TzFIU3l3dk5nS3dJRDl3SDgwZFNVYmFsUGljMVNHNStPcm9mdlQ5NXRneWljOWRTYjdDN0NqTDJSb2padVU3ZU5ocnp1RkRUZjRTL2ttVDNOeG5PYmZabXdPQkVGQlNrRlM5RVBBREFUakJsUFNTb3MrajExcWJ2YVViTGJGMTFVZWhQK0RRV2QrYWtkb2EvUk13eU1wMnczemxPU3BSUFRZcitad1AyQnU1OVg0ckdKRU1vY3owdXdYb25Sck9ydXpiRlV6aVYyc1EwWFhLVGRVR2dOWFA4UUVnakVQSUxKRU5ydzg0S3FxblJDa1ZCMzl3dG94OE9PQy9vdHVLUXlLa2lCczdiTThSSmlzM0lNNm5pK1kyZCtvYjhqbXMwMTNLNElEUGRrR1NyMklUNzdQVTFHa2xXd2V5K1VZNjhSVjFnV0Y1Q0lyc0ZEN2FkNW9WZEZKSFVkN0tpUFhXZGxvcFJ3MWJsTExYNGpCVUY1b094WkpRZk54dlN5Yi9tczJOcnQvbzEzcDJHTzdIMmFaOVArczJJWXJTaTlSaWcvcDcrNHdyanRBSlJjUkJ6anZ3Y0pxaFhjY29wd0pGWWFpREE4MkdZcU51ZDZyZUhsdHdBTW4yQUNOalUyOHM0RDRYSHY4YTlSeDNReDZjN2VVU3ZFZVdsMExCUEIxWE52Wk5lNUJFSmd6eklIOHdBQ0NGQm1BVkJlWGtJVXUvd1paenlUekhKcUZ6emdmLzBGTHU2NzZOZkxuVHY0ajkvbUh2TVc5b3k4M3FEWVBDSkVEU051TWtyUUZzZGExUU94eWJQbXVuK1JhOWNRd1B3ZzNLRWJod2R6Z3VuMXZPaWVsNi9jZ28yQVZlTmdkNzBtMXR5WVdXTGdpQklxdzlsWCtXREkwNTc4YUtJRDltU2R3R1hkbjBBb0pMcDFPYnNvaVY1WW16SU9FZktXTjEvUnV1ZnJhSWNCWHBwNmlZZktnUU9hWmJKc1BXQTdKZFRNQmRkLzY3bndXZVM4WnUwTHdkUlBobVlUN2pyYlJ3a1BxOERTU2RYVmEvT1VvWlRrZU5IQWx3L2ROc2tab0xnek5pVlVoelE1TWdWUE5VRHhSUVV6VkNUZlhHTytoMWxaTG4wbXA0QlVEbDJzUnh3PT18MTUzNjY2OTYxMnw=|1536583213|wtRmBn2qPxwS7DyzBpItswrNMaU=; Path=/; Domain=cd.jenkins-x.io; Expires=Mon, 17 Sep 2018 12:40:13 GMT; HttpOnly; Secure","strict-transport-security: max-age=15724800; includeSubDomains"],"stream_id":3},"phase":0,"source":{"id":56,"type":9},"time":"27661324","type":177},`
const ssoCookie = "ZW1haWw6Y29zbWluLmNvam9jYXJAZ214LmNoIHVzZXI6fGJiZW9heG1ycFdWMCs2VGNjSVlOWTlOTEx3cmxJcCtIMnF1MzgxQzkxc2FqL3R2SUR6enlDRE1YNmJCY2ZxSG9PNiszMDRnY1RFSDgvNUY1NlMwR21kMDRXcyt6KzQyVllXbVcraGJqN25rRzFtc3JEMHNsa3J3U1pLeVJrZmJBTkJsS0NJd0QzUFlUcUxsbjJEL09Kdng5VUpoSHFsSy93ZERGUFhRNDl6T3NPUlRHUm0wS1Y5TzFIU3l3dk5nS3dJRDl3SDgwZFNVYmFsUGljMVNHNStPcm9mdlQ5NXRneWljOWRTYjdDN0NqTDJSb2padVU3ZU5ocnp1RkRUZjRTL2ttVDNOeG5PYmZabXdPQkVGQlNrRlM5RVBBREFUakJsUFNTb3MrajExcWJ2YVViTGJGMTFVZWhQK0RRV2QrYWtkb2EvUk13eU1wMnczemxPU3BSUFRZcitad1AyQnU1OVg0ckdKRU1vY3owdXdYb25Sck9ydXpiRlV6aVYyc1EwWFhLVGRVR2dOWFA4UUVnakVQSUxKRU5ydzg0S3FxblJDa1ZCMzl3dG94OE9PQy9vdHVLUXlLa2lCczdiTThSSmlzM0lNNm5pK1kyZCtvYjhqbXMwMTNLNElEUGRrR1NyMklUNzdQVTFHa2xXd2V5K1VZNjhSVjFnV0Y1Q0lyc0ZEN2FkNW9WZEZKSFVkN0tpUFhXZGxvcFJ3MWJsTExYNGpCVUY1b094WkpRZk54dlN5Yi9tczJOcnQvbzEzcDJHTzdIMmFaOVArczJJWXJTaTlSaWcvcDcrNHdyanRBSlJjUkJ6anZ3Y0pxaFhjY29wd0pGWWFpREE4MkdZcU51ZDZyZUhsdHdBTW4yQUNOalUyOHM0RDRYSHY4YTlSeDNReDZjN2VVU3ZFZVdsMExCUEIxWE52Wk5lNUJFSmd6eklIOHdBQ0NGQm1BVkJlWGtJVXUvd1paenlUekhKcUZ6emdmLzBGTHU2NzZOZkxuVHY0ajkvbUh2TVc5b3k4M3FEWVBDSkVEU051TWtyUUZzZGExUU94eWJQbXVuK1JhOWNRd1B3ZzNLRWJod2R6Z3VuMXZPaWVsNi9jZ28yQVZlTmdkNzBtMXR5WVdXTGdpQklxdzlsWCtXREkwNTc4YUtJRDltU2R3R1hkbjBBb0pMcDFPYnNvaVY1WW16SU9FZktXTjEvUnV1ZnJhSWNCWHBwNmlZZktnUU9hWmJKc1BXQTdKZFRNQmRkLzY3bndXZVM4WnUwTHdkUlBobVlUN2pyYlJ3a1BxOERTU2RYVmEvT1VvWlRrZU5IQWx3L2ROc2tab0xnek5pVlVoelE1TWdWUE5VRHhSUVV6VkNUZlhHTytoMWxaTG4wbXA0QlVEbDJzUnh3PT18MTUzNjY2OTYxMnw=|1536583213|wtRmBn2qPxwS7DyzBpItswrNMaU="

func TestExtractSsoCookie(t *testing.T) {
	extratedCookie := cmd.ExtractSsoCookie(text)
	assert.Equal(t, ssoCookie, extratedCookie)
}

func newTestServer(fn func(http.ResponseWriter, *http.Request)) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc(cmd.UserOnboardingEndpoint, fn)
	server := httptest.NewServer(mux)
	return server
}

func TestOnboardUser(t *testing.T) {
	login := cmd.Login{
		Data: cmd.UserLoginInfo{
			Ca:     "test",
			Login:  "test",
			Server: "test",
			Token:  "test",
		},
	}
	cookie := "test"
	server := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(404)
		}
		providedCookie, err := r.Cookie(cmd.SsoCookieName)
		assert.NoError(t, err, "sso cookie should be present in request")
		assert.Equal(t, cookie, providedCookie.Value)
		output, err := json.Marshal(&login)
		assert.NoError(t, err)
		w.WriteHeader(http.StatusCreated)
		w.Write(output)
	})
	defer server.Close()

	cmd := &cmd.LoginOptions{
		URL: server.URL,
	}
	userInfoLogin, err := cmd.OnboardUser(cookie)
	assert.NoError(t, err)
	assert.NotNil(t, userInfoLogin)
	assert.Equal(t, login.Data, *userInfoLogin)

	userInfoLogin, err = cmd.OnboardUser("")
	assert.Error(t, err)
	assert.Nil(t, userInfoLogin)
}
