// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

// Request is the prepared HTTP request handed to the transport [Doer]: the verb
// method, the fully-built URL (base_uri + path + query), the resolved headers
// and the encoded string body, plus the per-call [RequestOptions] a host
// transport may consult (e.g. Timeout). It models the state HTTParty::Request
// has assembled by the time it calls Net::HTTP.
type Request struct {
	// Method is the upper-case HTTP method ("GET", "POST", …).
	Method string
	// URL is the fully-built request URL.
	URL string
	// Headers are the outgoing request headers.
	Headers *Headers
	// Body is the encoded request body (already form- or JSON-encoded, or a
	// caller-supplied raw string; empty for bodiless requests).
	Body string
	// Options carries the per-call settings (see [RequestOptions]).
	Options RequestOptions
}

// RawResponse is the wire response a [Doer] produces for a single round-trip:
// the status code, the raw (unparsed) body, and the response headers. HTTParty's
// content-type-aware parsing and redirect following happen in this package
// around the Doer, so a Doer performs exactly one request and never follows a
// redirect itself.
type RawResponse struct {
	// Code is the HTTP status code.
	Code int
	// Body is the raw response body as delivered by the transport.
	Body string
	// Headers are the response headers (repeats preserved via [Headers.Add]).
	Headers *Headers
}

// BasicAuth carries HTTP Basic credentials, mirroring HTTParty's
// :basic_auth => { username:, password: } option and the basic_auth class DSL.
type BasicAuth struct {
	Username string
	Password string
}

// RequestOptions is the per-call options bag, modelling the options Hash HTTParty
// accepts on every verb (HTTParty.get(url, query:, body:, headers:, basic_auth:,
// timeout:, follow_redirects:, format:)). The zero value is valid: an empty
// options set issues a plain request.
type RequestOptions struct {
	// Query are the query-string parameters (the :query option), merged after
	// the client's default_params.
	Query *Params
	// Body is the request body (the :body option): a raw string is sent as-is; a
	// [*Params] is form-encoded (application/x-www-form-urlencoded); any other
	// value is JSON-encoded (application/json).
	Body any
	// Headers are per-call request headers (the :headers option), merged over the
	// client's default headers.
	Headers *Headers
	// BasicAuth sets HTTP Basic credentials (the :basic_auth option); it overrides
	// the client's basic_auth for this call.
	BasicAuth *BasicAuth
	// Timeout is the request timeout in seconds (the :timeout option); metadata a
	// host transport may honour (this package opens no socket itself).
	Timeout int
	// FollowRedirects controls 3xx following (the :follow_redirects option);
	// nil means the HTTParty default of true.
	FollowRedirects *bool
	// MaxRedirects caps the redirect chain (the :limit option); <= 0 means the
	// HTTParty default of 5.
	MaxRedirects int
	// Format forces the response parser (the :format option: "json", "xml",
	// "html", "csv", "plain"); empty derives the format from the Content-Type.
	Format string
	// Transport injects the [Doer] for this call (the host seam); nil falls back
	// to the client's transport, then to [NetHTTP]. Tests inject a stub here.
	Transport Doer
}
