package containerregistry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

const (
	// GrantTypeAccessRefreshToken is the keyword used by the azure API that we are
	// authenticating using a refresh token.
	GrantTypeAccessRefreshToken = "access_token_refresh_token"
	// DockerTokenLoginUsernameGUID is the username for `docker login` when using
	// Azure Container Registry with a refresh token.
	DockerTokenLoginUsernameGUID = "00000000-0000-0000-0000-000000000000"
)

var client = &http.Client{}

// AuthDirective is the auth challenge returned from an Azure Container Registry.
type AuthDirective struct {
	Service string
	Realm   string
}

// AuthResponse is the auth response returned from an Azure Container Registry.
type AuthResponse struct {
	RefreshToken string `json:"refresh_token"`
}

// ReceiveChallengeFromLoginServer makes a request against an Azure Container Registry to
// retrieve the auth challenge. This is used in conjunction with PerformTokenExchange to
// authenticate against an Azure Container Registry instance.
func ReceiveChallengeFromLoginServer(serverAddress string) (*AuthDirective, error) {
	challengeURL := url.URL{
		Scheme: "https",
		Host:   serverAddress,
		Path:   "v2/",
	}
	var err error
	var r *http.Request
	r, _ = http.NewRequest("GET", challengeURL.String(), nil)

	var challenge *http.Response
	if challenge, err = client.Do(r); err != nil {
		return nil, fmt.Errorf("Error reaching registry endpoint %s, error: %s", challengeURL.String(), err)
	}
	defer challenge.Body.Close()

	if challenge.StatusCode != 401 {
		return nil, fmt.Errorf("Registry did not issue a valid AAD challenge, status: %d", challenge.StatusCode)
	}

	var authHeader []string
	var ok bool
	if authHeader, ok = challenge.Header["Www-Authenticate"]; !ok {
		return nil, fmt.Errorf("Challenge response does not contain header 'Www-Authenticate'")
	}

	if len(authHeader) != 1 {
		return nil, fmt.Errorf("Registry did not issue a valid AAD challenge, authenticate header [%s]",
			strings.Join(authHeader, ", "))
	}

	authSections := strings.SplitN(authHeader[0], " ", 2)
	authType := strings.ToLower(authSections[0])
	var authParams *map[string]string
	if authParams, err = parseAssignments(authSections[1]); err != nil {
		return nil, fmt.Errorf("Unable to understand the contents of Www-Authenticate header %s", authSections[1])
	}

	// verify headers
	if !strings.EqualFold("Bearer", authType) {
		return nil, fmt.Errorf("Www-Authenticate: expected realm: Bearer, actual: %s", authType)
	}
	if len((*authParams)["service"]) == 0 {
		return nil, fmt.Errorf("Www-Authenticate: missing header \"service\"")
	}
	if len((*authParams)["realm"]) == 0 {
		return nil, fmt.Errorf("Www-Authenticate: missing header \"realm\"")
	}

	return &AuthDirective{
		Service: (*authParams)["service"],
		Realm:   (*authParams)["realm"],
	}, nil
}

// PerformTokenExchange makes a request against an Azure Container Registry to
// send the auth challenge, returning an auth token. This is used in conjunction
// with ReceiveChallengeFromLoginServer to authenticate against an Azure
// Container Registry instance.
func PerformTokenExchange(
	serverAddress string,
	directive *AuthDirective,
	tenant string,
	accessToken string) (string, error) {
	var err error
	data := url.Values{
		"service":       []string{directive.Service},
		"grant_type":    []string{GrantTypeAccessRefreshToken},
		"access_token":  []string{accessToken},
		"refresh_token": []string{accessToken},
		"tenant":        []string{tenant},
	}

	var realmURL *url.URL
	if realmURL, err = url.Parse(directive.Realm); err != nil {
		return "", fmt.Errorf("Www-Authenticate: invalid realm %s", directive.Realm)
	}
	authEndpoint := fmt.Sprintf("%s://%s/oauth2/exchange", realmURL.Scheme, realmURL.Host)

	datac := data.Encode()
	var r *http.Request
	r, _ = http.NewRequest("POST", authEndpoint, bytes.NewBufferString(datac))
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", strconv.Itoa(len(datac)))

	var exchange *http.Response
	if exchange, err = client.Do(r); err != nil {
		return "", fmt.Errorf("Www-Authenticate: failed to reach auth url %s", authEndpoint)
	}

	defer exchange.Body.Close()
	if exchange.StatusCode != 200 {
		return "", fmt.Errorf("Www-Authenticate: auth url %s responded with status code %d", authEndpoint, exchange.StatusCode)
	}

	var content []byte
	if content, err = ioutil.ReadAll(exchange.Body); err != nil {
		return "", fmt.Errorf("Www-Authenticate: error reading response from %s", authEndpoint)
	}

	var authResp AuthResponse
	if err = json.Unmarshal(content, &authResp); err != nil {
		return "", fmt.Errorf("Www-Authenticate: unable to read response %s", content)
	}

	return authResp.RefreshToken, nil
}

// Try and parse a string of assignments in the form of:
// key1 = value1, key2 = "value 2", key3 = ""
// Note: this method and handle quotes but does not handle escaping of quotes
func parseAssignments(statements string) (*map[string]string, error) {
	var cursor int
	result := make(map[string]string)
	var errorMsg = fmt.Errorf("malformed header value: %s", statements)
	for {
		// parse key
		equalIndex := nextOccurrence(statements, cursor, "=")
		if equalIndex == -1 {
			return nil, errorMsg
		}
		key := strings.TrimSpace(statements[cursor:equalIndex])

		// parse value
		cursor = nextNoneSpace(statements, equalIndex+1)
		if cursor == -1 {
			return nil, errorMsg
		}
		// case: value is quoted
		if statements[cursor] == '"' {
			cursor = cursor + 1
			// like I said, not handling escapes, but this will skip any comma that's
			// within the quotes which is somewhat more likely
			closeQuoteIndex := nextOccurrence(statements, cursor, "\"")
			if closeQuoteIndex == -1 {
				return nil, errorMsg
			}
			value := statements[cursor:closeQuoteIndex]
			result[key] = value

			commaIndex := nextNoneSpace(statements, closeQuoteIndex+1)
			if commaIndex == -1 {
				// no more comma, done
				return &result, nil
			} else if statements[commaIndex] != ',' {
				// expect comma immediately after close quote
				return nil, errorMsg
			} else {
				cursor = commaIndex + 1
			}
		} else {
			commaIndex := nextOccurrence(statements, cursor, ",")
			endStatements := commaIndex == -1
			var untrimmed string
			if endStatements {
				untrimmed = statements[cursor:commaIndex]
			} else {
				untrimmed = statements[cursor:]
			}
			value := strings.TrimSpace(untrimmed)

			if len(value) == 0 {
				// disallow empty value without quote
				return nil, errorMsg
			}

			result[key] = value

			if endStatements {
				return &result, nil
			}
			cursor = commaIndex + 1
		}
	}
}

func nextOccurrence(str string, start int, sep string) int {
	if start >= len(str) {
		return -1
	}
	offset := strings.Index(str[start:], sep)
	if offset == -1 {
		return -1
	}
	return offset + start
}

func nextNoneSpace(str string, start int) int {
	if start >= len(str) {
		return -1
	}
	offset := strings.IndexFunc(str[start:], func(c rune) bool { return !unicode.IsSpace(c) })
	if offset == -1 {
		return -1
	}
	return offset + start
}
