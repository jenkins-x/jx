// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// DefaultDialer is a dialer with all fields set to the default zero values.
	DefaultDialer = &Dialer{
		Proxy: http.ProxyFromEnvironment,
	}

	// ErrBadHandshake is returned when the server response to opening handshake is
	// invalid.
	ErrBadHandshake       = errors.New("websocket: bad handshake")
	errInvalidCompression = errors.New("websocket: invalid compression negotiation")
	errMalformedRequest   = errors.New("malformed HTTP request")
	errMalformedURL       = errors.New("malformed ws or wss URL")
)

// A Dialer contains options for connecting to WebSocket server.
type Dialer struct {
	// NetDial specifies the dial function for creating TCP connections. If
	// NetDial is nil, net.Dial is used.
	NetDial func(network, addr string) (net.Conn, error)

	// Proxy specifies a function to return a proxy for a given
	// Request. If the function returns a non-nil error, the
	// request is aborted with the provided error.
	// If Proxy is nil or returns a nil *URL, no proxy is used.
	Proxy func(*http.Request) (*url.URL, error)

	// TLSClientConfig specifies the TLS configuration to use with tls.Client.
	// If nil, the default configuration is used.
	TLSClientConfig *tls.Config

	// HandshakeTimeout specifies the duration for the handshake to complete.
	HandshakeTimeout time.Duration

	// Input and output buffer sizes. If the buffer size is zero, then a
	// default value of 4096 is used.
	ReadBufferSize, WriteBufferSize int

	// Subprotocols specifies the client's requested subprotocols.
	Subprotocols []string

	// EnableCompression specifies if the client should attempt to negotiate
	// per message compression (RFC 7692). Setting this value to true does not
	// guarantee that compression will be supported. Currently only "no context
	// takeover" modes are supported.
	EnableCompression bool
}

func hostPortNoPort(u *url.URL) (hostPort, hostNoPort string) {
	hostPort = u.Host
	hostNoPort = u.Host
	if i := strings.LastIndex(u.Host, ":"); i > strings.LastIndex(u.Host, "]") {
		hostNoPort = hostNoPort[:i]
	} else {
		switch u.Scheme {
		case "wss":
			hostPort += ":443"
		case "https":
			hostPort += ":443"
		default:
			hostPort += ":80"
		}
	}
	return hostPort, hostNoPort
}

// DefaultRequest is a request with all fields set to the defaults for a
// websocket connection with the given URL.
func DefaultRequest(u *url.URL) *http.Request {
	return &http.Request{
		Method:     "GET",
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       u.Host,
	}
}

// Dial creates a new client connection using the provided HTTP request.
//
// If the WebSocket handshake fails, ErrBadHandshake is returned along with a
// non-nil *http.Response so that callers can handle redirects, authentication,
// etcetera. The response body may not contain the entire response and does not
// need to be closed by the application.
func (d *Dialer) Dial(req *http.Request) (*Conn, *http.Response, error) {

	if d == nil {
		d = &Dialer{
			Proxy: http.ProxyFromEnvironment,
		}
	}

	if req == nil {
		// A request must be set before calling Dial().
		return nil, nil, errMalformedRequest
	}

	if req.URL == nil {
		// A websocket URI must be set before calling Dial().
		return nil, nil, errMalformedURL
	}

	challengeKey, err := generateChallengeKey()
	if err != nil {
		return nil, nil, err
	}

	switch req.URL.Scheme {
	case "ws":
		req.URL.Scheme = "http"
	case "wss":
		req.URL.Scheme = "https"
	default:
		return nil, nil, errMalformedURL
	}

	if req.URL.User != nil {
		// User name and password are not allowed in websocket URIs.
		return nil, nil, errMalformedURL
	}

	// Do some sanity checking on the request headers
	for k, _ := range req.Header {
		switch {
		case k == "Upgrade" ||
			k == "Connection" ||
			k == "Sec-Websocket-Key" ||
			k == "Sec-Websocket-Version" ||
			k == "Sec-Websocket-Extensions" ||
			(k == "Sec-Websocket-Protocol" && len(d.Subprotocols) > 0):
			return nil, nil, errors.New("websocket: duplicate header not allowed: " + k)
		}
	}

	// Set the request headers using the capitalization for names and values in
	// RFC examples. Although the capitalization shouldn't matter, there are
	// servers that depend on it. The Header.Set method is not used because the
	// method canonicalizes the header names.
	req.Header["Upgrade"] = []string{"websocket"}
	req.Header["Connection"] = []string{"Upgrade"}
	req.Header["Sec-WebSocket-Key"] = []string{challengeKey}
	req.Header["Sec-WebSocket-Version"] = []string{"13"}
	if len(d.Subprotocols) > 0 {
		req.Header["Sec-WebSocket-Protocol"] = []string{strings.Join(d.Subprotocols, ", ")}
	}

	if d.EnableCompression {
		req.Header.Set("Sec-Websocket-Extensions", "permessage-deflate; server_no_context_takeover; client_no_context_takeover")
	}

	hostPort, hostNoPort := hostPortNoPort(req.URL)

	var proxyURL *url.URL
	// Check wether the proxy method has been configured
	if d.Proxy != nil {
		proxyURL, err = d.Proxy(req)
	}
	if err != nil {
		return nil, nil, err
	}

	var targetHostPort string
	if proxyURL != nil {
		targetHostPort, _ = hostPortNoPort(proxyURL)
	} else {
		targetHostPort = hostPort
	}

	var deadline time.Time
	if d.HandshakeTimeout != 0 {
		deadline = time.Now().Add(d.HandshakeTimeout)
	}

	netDial := d.NetDial
	if netDial == nil {
		netDialer := &net.Dialer{Deadline: deadline}
		netDial = netDialer.Dial
	}

	netConn, err := netDial("tcp", targetHostPort)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		if netConn != nil {
			netConn.Close()
		}
	}()

	if err := netConn.SetDeadline(deadline); err != nil {
		return nil, nil, err
	}

	if proxyURL != nil {
		connectHeader := make(http.Header)
		if user := proxyURL.User; user != nil {
			proxyUser := user.Username()
			if proxyPassword, passwordSet := user.Password(); passwordSet {
				credential := base64.StdEncoding.EncodeToString([]byte(proxyUser + ":" + proxyPassword))
				connectHeader.Set("Proxy-Authorization", "Basic "+credential)
			}
		}
		connectReq := &http.Request{
			Method: "CONNECT",
			URL:    &url.URL{Opaque: hostPort},
			Host:   hostPort,
			Header: connectHeader,
		}

		connectReq.Write(netConn)

		// Read response.
		// Okay to use and discard buffered reader here, because
		// TLS server will not speak until spoken to.
		br := bufio.NewReader(netConn)
		resp, err := http.ReadResponse(br, connectReq)
		if err != nil {
			return nil, nil, err
		}
		if resp.StatusCode != 200 {
			f := strings.SplitN(resp.Status, " ", 2)
			return nil, nil, errors.New(f[1])
		}
	}

	if req.URL.Scheme == "https" {
		cfg := cloneTLSConfig(d.TLSClientConfig)
		if cfg.ServerName == "" {
			cfg.ServerName = hostNoPort
		}
		tlsConn := tls.Client(netConn, cfg)
		netConn = tlsConn
		if err := tlsConn.Handshake(); err != nil {
			return nil, nil, err
		}
		if !cfg.InsecureSkipVerify {
			if err := tlsConn.VerifyHostname(cfg.ServerName); err != nil {
				return nil, nil, err
			}
		}
	}

	conn := newConn(netConn, false, d.ReadBufferSize, d.WriteBufferSize)

	if err := req.Write(netConn); err != nil {
		return nil, nil, err
	}

	resp, err := http.ReadResponse(conn.br, req)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != 101 ||
		!strings.EqualFold(resp.Header.Get("Upgrade"), "websocket") ||
		!strings.EqualFold(resp.Header.Get("Connection"), "upgrade") ||
		resp.Header.Get("Sec-Websocket-Accept") != computeAcceptKey(challengeKey) {
		// Before closing the network connection on return from this
		// function, slurp up some of the response to aid application
		// debugging.
		buf := make([]byte, 1024)
		n, _ := io.ReadFull(resp.Body, buf)
		resp.Body = ioutil.NopCloser(bytes.NewReader(buf[:n]))
		return nil, resp, ErrBadHandshake
	}

	for _, ext := range parseExtensions(req.Header) {
		if ext[""] != "permessage-deflate" {
			continue
		}
		_, snct := ext["server_no_context_takeover"]
		_, cnct := ext["client_no_context_takeover"]
		if !snct || !cnct {
			return nil, resp, errInvalidCompression
		}
		conn.newCompressionWriter = compressNoContextTakeover
		conn.newDecompressionReader = decompressNoContextTakeover
		break
	}

	resp.Body = ioutil.NopCloser(bytes.NewReader([]byte{}))
	conn.subprotocol = resp.Header.Get("Sec-Websocket-Protocol")

	netConn.SetDeadline(time.Time{})
	netConn = nil // to avoid close in defer.
	return conn, resp, nil
}

// cloneTLSConfig clones all public fields except the fields
// SessionTicketsDisabled and SessionTicketKey. This avoids copying the
// sync.Mutex in the sync.Once and makes it safe to call cloneTLSConfig on a
// config in active use.
func cloneTLSConfig(cfg *tls.Config) *tls.Config {
	if cfg == nil {
		return &tls.Config{}
	}
	return &tls.Config{
		Rand:                     cfg.Rand,
		Time:                     cfg.Time,
		Certificates:             cfg.Certificates,
		NameToCertificate:        cfg.NameToCertificate,
		GetCertificate:           cfg.GetCertificate,
		RootCAs:                  cfg.RootCAs,
		NextProtos:               cfg.NextProtos,
		ServerName:               cfg.ServerName,
		ClientAuth:               cfg.ClientAuth,
		ClientCAs:                cfg.ClientCAs,
		InsecureSkipVerify:       cfg.InsecureSkipVerify,
		CipherSuites:             cfg.CipherSuites,
		PreferServerCipherSuites: cfg.PreferServerCipherSuites,
		ClientSessionCache:       cfg.ClientSessionCache,
		MinVersion:               cfg.MinVersion,
		MaxVersion:               cfg.MaxVersion,
		CurvePreferences:         cfg.CurvePreferences,
	}
}
