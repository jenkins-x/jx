package util

import (
	"crypto/sha1"
	"encoding/base64"
)

// BasicAuth encodes the provided user name and password as basic auth credentials
func BasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// HashPassword hashes the given password with SHA1
func HashPassword(password string) string {
	s := sha1.New()
	s.Write([]byte(password))
	passwordSum := s.Sum(nil)
	return base64.StdEncoding.EncodeToString(passwordSum)
}
