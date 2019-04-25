package util

import (
	"bytes"
	"net/url"
	"strings"
)

// UrlJoin joins the given paths so that there is only ever one '/' character between the paths
func UrlJoin(paths ...string) string {
	var buffer bytes.Buffer
	last := len(paths) - 1
	for i, path := range paths {
		p := path
		if i > 0 {
			buffer.WriteString("/")
			p = strings.TrimPrefix(p, "/")
		}
		if i < last {
			p = strings.TrimSuffix(p, "/")
		}
		buffer.WriteString(p)
	}
	return buffer.String()
}

// UrlHostNameWithoutPort returns the host name without any port of the given URL like string
func UrlHostNameWithoutPort(rawUri string) (string, error) {
	if strings.Index(rawUri, ":/") > 0 {
		u, err := url.Parse(rawUri)
		if err != nil {
			return "", err
		}
		rawUri = u.Host
	}

	// must be a crazy kind of string so lets do our best
	slice := strings.Split(rawUri, ":")
	idx := 0
	if len(slice) > 1 {
		if len(slice) > 2 {
			idx = 1
		}
		return strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(slice[idx], "/"), "/"), "/"), nil
	}
	return rawUri, nil
}

// URLEqual verifies if URLs are equal
func URLEqual(url1, url2 string) bool {
	return url1 == url2 || strings.TrimSuffix(url1, "/") == strings.TrimSuffix(url2, "/")
}

// StripCredentialsFromURL strip credentials from URL
func StripCredentialsFromURL(u *url.URL) string {
	pass, hasPassword := u.User.Password()
	userName := u.User.Username()
	if hasPassword {
		textToReplace := pass + "@"
		if userName != "" {
			textToReplace = userName + ":" + textToReplace
		}
		return strings.Replace(u.String(), textToReplace, "", 1)
	}
	return u.String()
}

// UrlToHostName converts the given URL to a host name returning the error string if its not a URL
func UrlToHostName(svcUrl string) string {
	host := ""
	if svcUrl != "" {
		u, err := url.Parse(svcUrl)
		if err != nil {
			host = err.Error()
		} else {
			host = u.Host
		}
	}
	return host
}
