// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

import (
	"encoding/json"
	"net/url"
)

// Client is a configured HTTParty client, modelling a Ruby class that does
// `include HTTParty` together with its class-level DSL: [Client.BaseURI]
// (base_uri), [Client.Headers] (headers), [Client.DefaultParams]
// (default_params), [Client.BasicAuth] (basic_auth) and [Client.Format]
// (format). Its verb methods ([Client.Get], [Client.Post], …) issue requests
// bound to that configuration. The package-level verb functions ([Get], [Post],
// …) are the module methods (HTTParty.get, …), backed by a zero-value client.
type Client struct {
	baseURI       string
	headers       *Headers
	defaultParams *Params
	basicAuth     *BasicAuth
	format        string
	transport     Doer
}

// NewClient builds a [Client], optionally configured by a block (mirroring the
// class-level DSL applied when a class does `include HTTParty`).
func NewClient(block ...func(*Client)) *Client {
	c := &Client{}
	if len(block) > 0 && block[0] != nil {
		block[0](c)
	}
	return c
}

// BaseURI sets the base URI that request paths resolve against (base_uri).
func (c *Client) BaseURI(uri string) *Client { c.baseURI = uri; return c }

// Headers sets the default headers merged into every request (headers).
func (c *Client) Headers(h *Headers) *Client { c.headers = h; return c }

// DefaultParams sets the default query params merged into every request
// (default_params).
func (c *Client) DefaultParams(p *Params) *Client { c.defaultParams = p; return c }

// BasicAuth sets default HTTP Basic credentials (basic_auth).
func (c *Client) BasicAuth(user, pass string) *Client {
	c.basicAuth = &BasicAuth{Username: user, Password: pass}
	return c
}

// Format forces the default response parse format (format): "json", "xml",
// "html", "csv" or "plain".
func (c *Client) Format(f string) *Client { c.format = f; return c }

// Adapter sets the client's default transport [Doer] (the host seam); a per-call
// RequestOptions.Transport still overrides it. Tests inject a stub here or per
// call.
func (c *Client) Adapter(d Doer) *Client { c.transport = d; return c }

// Get issues a GET request (HTTParty.get / .get on an including class).
func (c *Client) Get(path string, opts ...RequestOptions) (*Response, error) {
	return c.perform("GET", path, firstOpt(opts))
}

// Head issues a HEAD request (HTTParty.head).
func (c *Client) Head(path string, opts ...RequestOptions) (*Response, error) {
	return c.perform("HEAD", path, firstOpt(opts))
}

// Options issues an OPTIONS request (HTTParty.options).
func (c *Client) Options(path string, opts ...RequestOptions) (*Response, error) {
	return c.perform("OPTIONS", path, firstOpt(opts))
}

// Post issues a POST request (HTTParty.post).
func (c *Client) Post(path string, opts ...RequestOptions) (*Response, error) {
	return c.perform("POST", path, firstOpt(opts))
}

// Put issues a PUT request (HTTParty.put).
func (c *Client) Put(path string, opts ...RequestOptions) (*Response, error) {
	return c.perform("PUT", path, firstOpt(opts))
}

// Patch issues a PATCH request (HTTParty.patch).
func (c *Client) Patch(path string, opts ...RequestOptions) (*Response, error) {
	return c.perform("PATCH", path, firstOpt(opts))
}

// Delete issues a DELETE request (HTTParty.delete).
func (c *Client) Delete(path string, opts ...RequestOptions) (*Response, error) {
	return c.perform("DELETE", path, firstOpt(opts))
}

// defaultMaxRedirects is HTTParty's default redirect limit (options[:limit]).
const defaultMaxRedirects = 5

// Content types set when encoding a structured body.
const (
	formContentType = "application/x-www-form-urlencoded"
	jsonContentType = "application/json"
)

// perform builds the request from the client's configuration and the per-call
// options, then runs the request/redirect loop against the transport [Doer],
// returning the finished [Response]. It models HTTParty::Request#perform:
// validating the format and URI scheme, merging headers/params/auth, encoding
// the body, and following redirects up to the limit.
func (c *Client) perform(method, path string, opt RequestOptions) (*Response, error) {
	format := opt.Format
	if format == "" {
		format = c.format
	}
	if format != "" && !supportedFormat(format) {
		return nil, &Error{Kind: KindUnsupportedFormat, Message: "unsupported format: " + format}
	}

	base, err := c.buildURL(path, opt)
	if err != nil {
		return nil, err
	}

	headers := c.baseHeaders().Merge(opt.Headers)
	if auth := c.authFor(opt); auth != nil && !headers.Has("Authorization") {
		headers.Set("Authorization", BasicHeaderFrom(auth.Username, auth.Password))
	}

	body, err := encodeBody(opt.Body, headers)
	if err != nil {
		return nil, err
	}

	doer := resolveDoer(opt.Transport, c.transport)
	follow := opt.FollowRedirects == nil || *opt.FollowRedirects
	limit := opt.MaxRedirects
	if limit <= 0 {
		limit = defaultMaxRedirects
	}

	curMethod, curURL, curBody := method, base, body
	for redirects := 0; ; redirects++ {
		req := &Request{Method: curMethod, URL: curURL.String(), Headers: headers, Body: curBody, Options: opt}
		raw, err := doer.Call(req)
		if err != nil {
			return nil, err
		}
		if raw.Headers == nil {
			raw.Headers = NewHeaders()
		}
		if follow && isRedirect(raw.Code) {
			if loc := raw.Headers.Values("Location"); len(loc) > 0 {
				resp := newResponse(raw, format)
				if len(loc) > 1 {
					return nil, newResponseError(KindDuplicateLocationHeader, "duplicate Location header", resp)
				}
				if redirects >= limit {
					return nil, newResponseError(KindRedirectionTooDeep, "redirection too deep", resp)
				}
				next, err := resolveRedirect(curURL, loc[0])
				if err != nil {
					return nil, err
				}
				curURL = next
				if raw.Code == 303 {
					curMethod, curBody = "GET", ""
				}
				continue
			}
		}
		return newResponse(raw, format), nil
	}
}

// buildURL resolves path against the client's base_uri and appends the merged
// query params (base-URL query, then default_params, then the per-call :query),
// mirroring HTTParty's URI construction. It returns HTTParty::UnsupportedURIScheme
// when the resulting scheme is not http(s) or the URL cannot be parsed.
func (c *Client) buildURL(path string, opt RequestOptions) (*url.URL, error) {
	ref, err := url.Parse(path)
	if err != nil {
		return nil, newSchemeError(path, err)
	}
	u := ref
	if c.baseURI != "" {
		base, err := url.Parse(c.baseURI)
		if err != nil {
			return nil, newSchemeError(c.baseURI, err)
		}
		u = base.ResolveReference(ref)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, newSchemeError(u.String(), nil)
	}
	merged := ParseQuery(u.RawQuery)
	if c.defaultParams != nil {
		merged = merged.Merge(c.defaultParams)
	}
	if opt.Query != nil {
		merged = merged.Merge(opt.Query)
	}
	u.RawQuery = BuildQuery(merged)
	return u, nil
}

// baseHeaders returns a fresh clone of the client's default headers (or an empty
// set when it has none) so a request never mutates the client's headers.
func (c *Client) baseHeaders() *Headers {
	if c.headers == nil {
		return NewHeaders()
	}
	return c.headers.Clone()
}

// authFor returns the Basic credentials for a call: the per-call :basic_auth
// wins over the client's basic_auth.
func (c *Client) authFor(opt RequestOptions) *BasicAuth {
	if opt.BasicAuth != nil {
		return opt.BasicAuth
	}
	return c.basicAuth
}

// encodeBody encodes a request body and sets the Content-Type when it isn't
// already present: a string is sent verbatim; a [*Params] is form-encoded; any
// other value is JSON-encoded. A value that cannot be marshalled to JSON returns
// an HTTParty error.
func encodeBody(body any, h *Headers) (string, error) {
	switch v := body.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case *Params:
		h.SetDefault("Content-Type", formContentType)
		return BuildQuery(v), nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return "", &Error{Kind: KindError, Message: err.Error(), Cause: err}
		}
		h.SetDefault("Content-Type", jsonContentType)
		return string(data), nil
	}
}

// resolveRedirect resolves a Location value against the current URL and returns
// the next URL, rejecting a non-http(s) target or an unparsable Location with
// HTTParty::UnsupportedURIScheme.
func resolveRedirect(base *url.URL, loc string) (*url.URL, error) {
	ref, err := url.Parse(loc)
	if err != nil {
		return nil, newSchemeError(loc, err)
	}
	u := base.ResolveReference(ref)
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, newSchemeError(u.String(), nil)
	}
	return u, nil
}

// isRedirect reports whether a status code is a 3xx redirection.
func isRedirect(code int) bool { return code >= 300 && code < 400 }

// newSchemeError builds an HTTParty::UnsupportedURIScheme error for u.
func newSchemeError(u string, cause error) *Error {
	return &Error{Kind: KindUnsupportedURIScheme, Message: "unsupported URI scheme: " + u, Cause: cause}
}

// firstOpt returns the first optional options value or the zero value.
func firstOpt(opts []RequestOptions) RequestOptions {
	if len(opts) > 0 {
		return opts[0]
	}
	return RequestOptions{}
}
