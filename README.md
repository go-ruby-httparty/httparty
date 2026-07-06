<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-httparty/brand/main/social/go-ruby-httparty-httparty.png" alt="go-ruby-httparty/httparty" width="720"></p>

# httparty — go-ruby-httparty

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-httparty.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) model of the behaviour of Ruby's
[`httparty`](https://github.com/jnunemaker/httparty) gem** — the popular HTTP
client DSL over `Net::HTTP`. It reproduces the module verb methods, the `include
HTTParty` class DSL, the content-type-aware response, and the deterministic
request building (query / body / header encoding, Basic auth, redirect
following) HTTParty performs around a transport — **without any Ruby runtime**.

It is the HTTParty client for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a **standalone,
reusable** module — a sibling of
[go-ruby-faraday](https://github.com/go-ruby-faraday/faraday),
[go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) and
[go-ruby-erb](https://github.com/go-ruby-erb/erb).

> **What it is — and isn't.** Everything HTTParty does *around* the wire is
> deterministic and needs **no interpreter**, so it lives here as pure Go:
> building the request URL, encoding the body (form or JSON), applying Basic
> auth, following 3xx redirects, and parsing the response by content type. The
> **HTTP round-trip itself is a host seam**: the `Doer` performs one transport
> request. The default production `Doer` is `NetHTTP`, backed by `net/http`;
> **tests inject a `DoerFunc` stub and the core opens no socket itself.** A
> future rbgo binding wires the real transport and maps Ruby's HTTParty surface
> onto this API.

## Features

Faithful model of the `httparty` gem's client, validated against the gem where
it is installed:

- **Module verbs** — `Get`/`Head`/`Options`/`Post`/`Put`/`Patch`/`Delete`, each
  taking a URL and an optional `RequestOptions` (the options Hash:
  `Query`, `Body`, `Headers`, `BasicAuth`, `Timeout`, `FollowRedirects`,
  `MaxRedirects`, `Format`).
- **`include HTTParty` DSL** — `NewClient(...)` with `BaseURI` (base_uri),
  `Headers` (headers), `DefaultParams` (default_params), `BasicAuth`
  (basic_auth) and `Format` (format), plus the same verb methods bound to that
  configuration.
- **`Response`** — `Code`, `Body`, `Headers`, `Success` and the content-type
  aware `Parsed()`: a JSON body → `map`/slice/scalar, an XML body → nested
  `map`, and any other type (html/csv/plain/unknown) → the raw body string,
  mirroring HTTParty's `parsed_response`. A forced `Format` overrides the
  Content-Type.
- **Body / query encoding** — a raw string is sent as-is, a `*Params` is
  form-encoded (`application/x-www-form-urlencoded`), any other value is
  JSON-encoded (`application/json`); query params are order-preserving and
  escaped with `ERB::Util.url_encode` semantics.
- **Redirect following** — 3xx `Location` following up to a limit, with the
  HTTParty behaviour of switching to `GET` on `303`; over-deep chains and
  duplicate `Location` headers raise the matching error.
- **Transport seam** — `RequestOptions.Transport` / `Client.Adapter(Doer)`;
  `NetHTTP()` is the default net/http transport (redirect-following disabled so
  the client loop controls it), a `DoerFunc` a test stub. **The core never opens
  a socket.**
- **Error tree** — `HTTParty::Error` → `UnsupportedFormat`,
  `UnsupportedURIScheme`, `ResponseError` (`RedirectionTooDeep`,
  `DuplicateLocationHeader`), matched with `errors.Is` against the `Err*`
  sentinels (a superclass matches its subclasses).

CGO-free, dependency-free (stdlib only), **100% test coverage**, `gofmt` +
`go vet` clean, and green across the six 64-bit Go targets (amd64, arm64,
riscv64, loong64, ppc64le, **s390x** — big-endian).

## Install

```sh
go get github.com/go-ruby-httparty/httparty
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-ruby-httparty/httparty"
)

func main() {
	// Module method (HTTParty.get).
	resp, err := httparty.Get("https://api.example.com/users",
		httparty.RequestOptions{Query: httparty.ParamsOf([2]string{"q", "ada"})})
	if err != nil {
		// a *httparty.Error: errors.Is(err, httparty.ErrUnsupportedURIScheme), etc.
		return
	}
	fmt.Println(resp.Code(), resp.Success())
	v, _ := resp.Parsed() // JSON -> map[string]any, XML -> map, else string
	fmt.Println(v)

	// `include HTTParty` class with the DSL.
	api := httparty.NewClient(func(c *httparty.Client) {
		c.BaseURI("https://api.example.com").
			Headers(httparty.HeadersOf([2]string{"Accept", "application/json"})).
			BasicAuth("user", "pass").
			Format("json")
	})
	created, err := api.Post("/widgets", httparty.RequestOptions{
		Body: map[string]any{"name": "gadget"}, // JSON-encoded
	})
	if err != nil {
		return
	}
	_ = errors.Is // keep the import in this snippet
	fmt.Println(created.Code())
}
```

### Injecting a transport (tests / hosts)

```go
resp, _ := httparty.Get("https://api.example.com/ping", httparty.RequestOptions{
	Format: "json",
	Transport: httparty.DoerFunc(func(req *httparty.Request) (*httparty.RawResponse, error) {
		h := httparty.HeadersOf([2]string{"Content-Type", "application/json"})
		return &httparty.RawResponse{Code: 200, Body: `{"ok":true}`, Headers: h}, nil
	}),
})
// resp.Parsed() == map[string]any{"ok": true}
```

## Value model

| gem                                          | this package                                       |
| -------------------------------------------- | -------------------------------------------------- |
| `HTTParty.get/post/... (url, options)`       | `Get/Post/...(url, RequestOptions{...})`           |
| `include HTTParty` + `base_uri`, `headers`   | `NewClient(func(c){ c.BaseURI(...).Headers(...) })`|
| `basic_auth`, `default_params`, `format`     | `c.BasicAuth(...)`, `c.DefaultParams(...)`, `c.Format(...)` |
| options `:query` / `:body` / `:headers`      | `RequestOptions.Query` / `.Body` / `.Headers`      |
| options `:basic_auth` / `:timeout`           | `RequestOptions.BasicAuth` / `.Timeout`            |
| options `:follow_redirects` / `:format`      | `RequestOptions.FollowRedirects` / `.Format`       |
| `HTTParty::Response#code/body/parsed_response` | `(*Response).Code()/Body()/Parsed()`             |
| `Net::HTTP` transport                        | `Doer`; `NetHTTP()` default (host seam)            |
| `HTTParty::Error` subtree                    | `*Error` + `Err*` sentinels (`errors.Is`)          |

## Tests & coverage

The suite pairs deterministic, ruby-free tests (which alone hold coverage at
**100%**, so the qemu cross-arch and Windows lanes pass the gate) with a
**differential oracle** against the reference `httparty` gem: query-value
escaping (vs `ERB::Util.url_encode`), JSON parsing (vs `HTTParty::Parser.call`),
and Basic-auth header construction are diffed **byte-for-byte** against the gem.
The oracle scripts `$stdout.binmode` and skip themselves where the gem is
absent. **No test opens a socket** — the transport is stubbed everywhere.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-httparty/httparty authors.
