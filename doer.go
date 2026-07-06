// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

import (
	"io"
	"net/http"
	"strings"
)

// Doer is the transport host seam. Given a prepared [Request], a Doer performs a
// single HTTP round-trip and returns the [RawResponse], or an error. It never
// follows redirects — this package's client loop does that around the Doer — so
// the core is fully exercisable in tests against an in-process stub and opens no
// socket itself. The default production Doer is [NetHTTP], backed by net/http; a
// host (rbgo) wires the real transport, and tests inject a [DoerFunc].
type Doer interface {
	Call(req *Request) (*RawResponse, error)
}

// DoerFunc adapts a function to the [Doer] interface — the convenient way to
// inject a stub transport in tests or a custom transport in a host.
type DoerFunc func(req *Request) (*RawResponse, error)

// Call invokes f(req).
func (f DoerFunc) Call(req *Request) (*RawResponse, error) { return f(req) }

// resolveDoer selects the transport for a call: the per-call Transport wins,
// then the client's transport, then the default [NetHTTP]. It is the single
// point where the default net/http transport is materialised, so a request that
// injects a stub never constructs (or touches) the real transport.
func resolveDoer(perCall, client Doer) Doer {
	if perCall != nil {
		return perCall
	}
	if client != nil {
		return client
	}
	return NetHTTP()
}

// httpClient is the minimal net/http surface [NetHTTPDoer] depends on,
// indirected so tests can drive the doer's request-building and
// response/error mapping without opening a socket.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// NetHTTPDoer is the default [Doer]: it turns a [Request] into a net/http
// request, executes it with its Client, and maps the response (or a transport
// failure) back to a [RawResponse]. It models HTTParty's use of Net::HTTP.
type NetHTTPDoer struct {
	// Client performs the request; [NetHTTP] wires an http.Client that does not
	// auto-follow redirects (this package follows them explicitly).
	Client httpClient
}

// NetHTTP returns the default net/http-backed [Doer]. Its http.Client is
// configured not to follow redirects itself ([noRedirect]) because redirect
// following is handled by the client loop around the Doer.
func NetHTTP() *NetHTTPDoer {
	return &NetHTTPDoer{Client: &http.Client{CheckRedirect: noRedirect}}
}

// noRedirect tells net/http to hand back the redirect response unchanged
// (http.ErrUseLastResponse) instead of following it.
func noRedirect(_ *http.Request, _ []*http.Request) error {
	return http.ErrUseLastResponse
}

// Call performs the HTTP round-trip for req with net/http.
func (d *NetHTTPDoer) Call(req *Request) (*RawResponse, error) {
	var body io.Reader
	if req.Body != "" {
		body = strings.NewReader(req.Body)
	}
	hr, err := http.NewRequest(req.Method, req.URL, body)
	if err != nil {
		return nil, &Error{Kind: KindError, Message: err.Error(), Cause: err}
	}
	for _, p := range req.Headers.Pairs() {
		hr.Header.Add(p.Key, p.Val)
	}

	resp, err := d.Client.Do(hr)
	if err != nil {
		return nil, &Error{Kind: KindError, Message: err.Error(), Cause: err}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &Error{Kind: KindError, Message: err.Error(), Cause: err}
	}
	return &RawResponse{
		Code:    resp.StatusCode,
		Body:    string(raw),
		Headers: headersFromHTTP(resp.Header),
	}, nil
}

// headersFromHTTP converts an http.Header into a [Headers], preserving each
// value of a repeated field (via [Headers.Add]) so a duplicate Location header
// on a redirect is still detectable.
func headersFromHTTP(h http.Header) *Headers {
	out := NewHeaders()
	for k, vs := range h {
		for _, v := range vs {
			out.Add(k, v)
		}
	}
	return out
}
