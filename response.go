// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

// Response models HTTParty::Response: the status code, the raw body, the
// response headers, and the content-type-aware [Response.Parsed] value. It is
// what every verb returns on a completed request.
type Response struct {
	code    int
	body    string
	headers *Headers
	// format is the forced parse format from the :format option / format DSL
	// ("" derives the format from the Content-Type).
	format string

	parsed     any
	parsedErr  error
	parsedDone bool
}

// newResponse wraps a completed [RawResponse], recording the forced parse format
// (empty to derive from the Content-Type).
func newResponse(raw *RawResponse, format string) *Response {
	return &Response{code: raw.Code, body: raw.Body, headers: raw.Headers, format: format}
}

// Code returns the HTTP status code (HTTParty::Response#code).
func (r *Response) Code() int { return r.code }

// Body returns the raw, unparsed response body (HTTParty::Response#body).
func (r *Response) Body() string { return r.body }

// Headers returns the response headers (HTTParty::Response#headers).
func (r *Response) Headers() *Headers { return r.headers }

// Success reports whether the status is a 2xx (HTTParty::Response#success?).
func (r *Response) Success() bool { return r.code >= 200 && r.code < 300 }

// Parsed returns the parsed body (HTTParty::Response#parsed_response): a JSON
// body becomes a map/slice/scalar, an XML body a nested map, and any other
// content type (html, csv, plain, unknown) the raw body string. The forced
// :format option overrides the Content-Type. A blank body yields the raw
// (blank) string. The result is computed once and cached. A malformed JSON/XML
// body returns a non-nil error, as HTTParty's parser raises.
func (r *Response) Parsed() (any, error) {
	if r.parsedDone {
		return r.parsed, r.parsedErr
	}
	format := normalizeParseFormat(r.format)
	if r.format == "" {
		ct, _ := r.headers.Get("Content-Type")
		format = deriveFormat(ct)
	}
	r.parsed, r.parsedErr = parseBody(format, r.body)
	r.parsedDone = true
	return r.parsed, r.parsedErr
}
