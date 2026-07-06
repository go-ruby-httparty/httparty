// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package httparty is a pure-Go (CGO-free) model of the behaviour of Ruby's
// `httparty` gem — the popular HTTP client DSL over Net::HTTP: the module verb
// methods (HTTParty.get/post/put/patch/delete/head/options), the `include
// HTTParty` class DSL (base_uri, headers, default_params, basic_auth, format),
// the content-type-aware Response, and the deterministic request building
// (query/body/header encoding, Basic auth, redirect following) HTTParty performs
// around the transport.
//
// # What it is — and isn't
//
// Everything HTTParty does *around* the wire is deterministic and needs no Ruby
// interpreter, so it lives here as pure Go: building the request URL (path
// resolved against base_uri, order-preserving escaped query strings), encoding
// the body (form or JSON), applying Basic auth, following 3xx redirects, and
// parsing the response by content type. The HTTP round-trip itself is a host
// seam: the [Doer] performs one transport request. The default production Doer
// is [NetHTTP], backed by net/http; tests inject a [DoerFunc] stub and the core
// opens no socket itself. A future rbgo binding wires the real transport and
// maps Ruby's HTTParty surface onto this API.
//
// # Two entry points
//
// The package-level verb functions are the module methods:
//
//	resp, err := httparty.Get("https://api.example.com/users",
//		httparty.RequestOptions{Query: httparty.ParamsOf([2]string{"q", "ada"})})
//	if err != nil { /* an *httparty.Error */ }
//	_ = resp.Code()          // 200
//	_ = resp.Success()       // true for 2xx
//	v, _ := resp.Parsed()    // JSON body -> map[string]any, XML -> map, else string
//
// A [Client] models a class that does `include HTTParty` with its DSL:
//
//	api := httparty.NewClient(func(c *httparty.Client) {
//		c.BaseURI("https://api.example.com").
//			Headers(httparty.HeadersOf([2]string{"Accept", "application/json"})).
//			BasicAuth("user", "pass").
//			Format("json")
//	})
//	resp, err := api.Post("/widgets", httparty.RequestOptions{
//		Body: map[string]any{"name": "gadget"}, // JSON-encoded
//	})
//
// # Value model
//
// Query params and url-encoded bodies are carried as an ordered string→string
// [Params]; headers as a case-insensitive, multi-valued [Headers]. A request
// body is a raw string, a [*Params] (form), or any value (JSON). The [Response]
// exposes Code/Body/Headers/Success and the content-type-aware [Response.Parsed].
// Errors are an [Error] tree matched with errors.Is against the Err* sentinels.
package httparty
