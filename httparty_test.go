// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

import (
	"errors"
	"reflect"
	"testing"
)

// recordDoer is the transport stub used across the suite: it records every
// [Request] it is called with and returns the next canned step. No test opens a
// socket.
type recordDoer struct {
	reqs  []*Request
	steps []step
	n     int
}

type step struct {
	resp *RawResponse
	err  error
}

func (d *recordDoer) Call(req *Request) (*RawResponse, error) {
	d.reqs = append(d.reqs, req)
	s := d.steps[d.n]
	d.n++
	return s.resp, s.err
}

// raw builds a canned RawResponse with a single Content-Type header.
func raw(code int, ct, body string) *RawResponse {
	h := NewHeaders()
	if ct != "" {
		h.Set("Content-Type", ct)
	}
	return &RawResponse{Code: code, Body: body, Headers: h}
}

func boolp(b bool) *bool { return &b }

func single(r *RawResponse) *recordDoer { return &recordDoer{steps: []step{{resp: r}}} }

func TestModuleGetJSON(t *testing.T) {
	d := single(raw(200, "application/json", `{"a":1,"b":[2,3]}`))
	resp, err := Get("https://api.example.com/things",
		RequestOptions{Query: ParamsOf([2]string{"q", "ada lovelace"}), Transport: d})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Code() != 200 || !resp.Success() {
		t.Fatalf("code=%d", resp.Code())
	}
	if got := d.reqs[0].URL; got != "https://api.example.com/things?q=ada%20lovelace" {
		t.Fatalf("url=%q", got)
	}
	v, err := resp.Parsed()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"a": float64(1), "b": []any{float64(2), float64(3)}}
	if !reflect.DeepEqual(v, want) {
		t.Fatalf("parsed=%#v", v)
	}
	// Parsed is cached (second call takes the memoised path).
	if v2, _ := resp.Parsed(); !reflect.DeepEqual(v2, want) {
		t.Fatalf("parsed cache=%#v", v2)
	}
	if resp.Body() != `{"a":1,"b":[2,3]}` {
		t.Fatalf("body=%q", resp.Body())
	}
}

func TestClientDSLAndFormJSONBody(t *testing.T) {
	d := single(raw(201, "application/json", `{"ok":true}`))
	api := NewClient(func(c *Client) {
		c.BaseURI("https://api.example.com").
			Headers(HeadersOf([2]string{"Accept", "application/json"})).
			DefaultParams(ParamsOf([2]string{"api_key", "K"})).
			Format("json").
			Adapter(d)
	})
	resp, err := api.Post("/widgets", RequestOptions{Body: map[string]any{"name": "gadget"}})
	if err != nil {
		t.Fatal(err)
	}
	req := d.reqs[0]
	if req.Method != "POST" || req.Body != `{"name":"gadget"}` {
		t.Fatalf("req=%+v", req)
	}
	if ct, _ := req.Headers.Get("Content-Type"); ct != jsonContentType {
		t.Fatalf("ct=%q", ct)
	}
	if ac, _ := req.Headers.Get("Accept"); ac != "application/json" {
		t.Fatalf("accept=%q", ac)
	}
	if req.URL != "https://api.example.com/widgets?api_key=K" {
		t.Fatalf("url=%q", req.URL)
	}
	if v, _ := resp.Parsed(); !reflect.DeepEqual(v, map[string]any{"ok": true}) {
		t.Fatalf("parsed=%#v", v)
	}
}

func TestFormBodyAndClientBasicAuth(t *testing.T) {
	d := single(raw(200, "text/plain", "ok"))
	api := NewClient().Adapter(d)
	api.BasicAuth("aladdin", "opensesame")
	_, err := api.Put("https://h.example/x", RequestOptions{
		Body: ParamsOf([2]string{"a", "b c"}, [2]string{"d", "e&f"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	req := d.reqs[0]
	if req.Body != "a=b%20c&d=e%26f" {
		t.Fatalf("form body=%q", req.Body)
	}
	if ct, _ := req.Headers.Get("Content-Type"); ct != formContentType {
		t.Fatalf("ct=%q", ct)
	}
	if a, _ := req.Headers.Get("Authorization"); a != "Basic YWxhZGRpbjpvcGVuc2VzYW1l" {
		t.Fatalf("auth=%q", a)
	}
}

func TestPerCallBasicAuthOverridesAndPresetSkips(t *testing.T) {
	// Per-call basic auth wins over the client's.
	d := single(raw(200, "text/plain", ""))
	api := NewClient().Adapter(d).BasicAuth("client", "secret")
	if _, err := api.Get("https://h/x", RequestOptions{BasicAuth: &BasicAuth{Username: "u", Password: "p"}}); err != nil {
		t.Fatal(err)
	}
	if a, _ := d.reqs[0].Headers.Get("Authorization"); a != BasicHeaderFrom("u", "p") {
		t.Fatalf("auth=%q", a)
	}
	// A preset Authorization header is not overwritten by basic_auth.
	d2 := single(raw(200, "text/plain", ""))
	api2 := NewClient().Adapter(d2).BasicAuth("client", "secret")
	if _, err := api2.Get("https://h/x", RequestOptions{Headers: HeadersOf([2]string{"Authorization", "Bearer tok"})}); err != nil {
		t.Fatal(err)
	}
	if a, _ := d2.reqs[0].Headers.Get("Authorization"); a != "Bearer tok" {
		t.Fatalf("auth=%q", a)
	}
}

func TestStringBodyVerbatimAndNilBody(t *testing.T) {
	d := &recordDoer{steps: []step{{resp: raw(200, "text/plain", "")}, {resp: raw(200, "text/plain", "")}}}
	api := NewClient().Adapter(d)
	if _, err := api.Patch("https://h/x", RequestOptions{Body: "raw-payload"}); err != nil {
		t.Fatal(err)
	}
	if d.reqs[0].Body != "raw-payload" {
		t.Fatalf("body=%q", d.reqs[0].Body)
	}
	if d.reqs[0].Headers.Has("Content-Type") {
		t.Fatalf("string body must not force a content type")
	}
	if _, err := api.Delete("https://h/x"); err != nil {
		t.Fatal(err)
	}
	if d.reqs[1].Body != "" {
		t.Fatalf("nil body=%q", d.reqs[1].Body)
	}
}

func TestJSONBodyMarshalError(t *testing.T) {
	d := single(raw(200, "text/plain", ""))
	api := NewClient().Adapter(d)
	_, err := api.Post("https://h/x", RequestOptions{Body: make(chan int)})
	if err == nil || !errors.Is(err, ErrError) {
		t.Fatalf("want marshal error, got %v", err)
	}
}

func TestUnsupportedFormat(t *testing.T) {
	_, err := Get("https://h/x", RequestOptions{Format: "yaml", Transport: single(raw(200, "", ""))})
	if err == nil || !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("want UnsupportedFormat, got %v", err)
	}
}

func TestUnsupportedURIScheme(t *testing.T) {
	// Bad scheme.
	if _, err := Get("ftp://h/x", RequestOptions{Transport: single(raw(200, "", ""))}); !errors.Is(err, ErrUnsupportedURIScheme) {
		t.Fatalf("want scheme error, got %v", err)
	}
	// Unparsable path (control character).
	if _, err := Get("http://h/\x7f", RequestOptions{Transport: single(raw(200, "", ""))}); !errors.Is(err, ErrUnsupportedURIScheme) {
		t.Fatalf("want parse error, got %v", err)
	}
	// Unparsable base_uri.
	bad := NewClient().BaseURI("http://\x7f").Adapter(single(raw(200, "", "")))
	if _, err := bad.Get("x"); !errors.Is(err, ErrUnsupportedURIScheme) {
		t.Fatalf("want base parse error, got %v", err)
	}
}

func TestTransportError(t *testing.T) {
	boom := errors.New("dial fail")
	d := &recordDoer{steps: []step{{err: boom}}}
	_, err := Get("https://h/x", RequestOptions{Transport: d})
	if !errors.Is(err, boom) {
		t.Fatalf("want transport error, got %v", err)
	}
}

func TestRedirectFollowedKeepsMethod(t *testing.T) {
	r1 := raw(302, "", "")
	r1.Headers.Set("Location", "/moved")
	d := &recordDoer{steps: []step{{resp: r1}, {resp: raw(200, "application/json", `{"done":true}`)}}}
	resp, err := Post("https://h/old", RequestOptions{Body: "keepme", Transport: d})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Code() != 200 {
		t.Fatalf("code=%d", resp.Code())
	}
	if d.reqs[1].URL != "https://h/moved" {
		t.Fatalf("redirect url=%q", d.reqs[1].URL)
	}
	if d.reqs[1].Method != "POST" || d.reqs[1].Body != "keepme" {
		t.Fatalf("302 must keep method+body: %s %q", d.reqs[1].Method, d.reqs[1].Body)
	}
}

func TestRedirect303BecomesGet(t *testing.T) {
	r1 := raw(303, "", "")
	r1.Headers.Set("Location", "https://other/x")
	d := &recordDoer{steps: []step{{resp: r1}, {resp: raw(200, "text/plain", "ok")}}}
	if _, err := Post("https://h/old", RequestOptions{Body: "drop", Transport: d}); err != nil {
		t.Fatal(err)
	}
	if d.reqs[1].Method != "GET" || d.reqs[1].Body != "" {
		t.Fatalf("303 must switch to GET+empty body: %s %q", d.reqs[1].Method, d.reqs[1].Body)
	}
	if d.reqs[1].URL != "https://other/x" {
		t.Fatalf("abs redirect url=%q", d.reqs[1].URL)
	}
}

func TestRedirectNotFollowedWhenDisabled(t *testing.T) {
	r1 := raw(302, "text/plain", "see other")
	r1.Headers.Set("Location", "/moved")
	d := &recordDoer{steps: []step{{resp: r1}}}
	resp, err := Get("https://h/x", RequestOptions{FollowRedirects: boolp(false), Transport: d})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Code() != 302 {
		t.Fatalf("code=%d", resp.Code())
	}
}

func TestRedirectNoLocationReturnsResponse(t *testing.T) {
	d := single(raw(304, "text/plain", "")) // 304, no Location -> not followed
	resp, err := Get("https://h/x", RequestOptions{Transport: d})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Code() != 304 {
		t.Fatalf("code=%d", resp.Code())
	}
}

func TestRedirectionTooDeep(t *testing.T) {
	r := raw(302, "", "")
	r.Headers.Set("Location", "/again")
	d := &recordDoer{steps: []step{{resp: r}, {resp: r}}}
	_, err := Get("https://h/x", RequestOptions{MaxRedirects: 1, Transport: d})
	if !errors.Is(err, ErrRedirectionTooDeep) || !IsResponseError(err) {
		t.Fatalf("want RedirectionTooDeep, got %v", err)
	}
}

func TestDuplicateLocationHeader(t *testing.T) {
	r := raw(302, "", "")
	r.Headers.Add("Location", "/a")
	r.Headers.Add("Location", "/b")
	d := &recordDoer{steps: []step{{resp: r}}}
	_, err := Get("https://h/x", RequestOptions{Transport: d})
	if !errors.Is(err, ErrDuplicateLocationHeader) {
		t.Fatalf("want DuplicateLocationHeader, got %v", err)
	}
	var e *Error
	if !errors.As(err, &e) || e.Response == nil {
		t.Fatalf("expected response context on error")
	}
}

func TestRedirectBadLocationScheme(t *testing.T) {
	r := raw(302, "", "")
	r.Headers.Set("Location", "gopher://evil/x")
	d := &recordDoer{steps: []step{{resp: r}}}
	if _, err := Get("https://h/x", RequestOptions{Transport: d}); !errors.Is(err, ErrUnsupportedURIScheme) {
		t.Fatalf("want scheme error, got %v", err)
	}
	// Unparsable Location.
	r2 := raw(302, "", "")
	r2.Headers.Set("Location", "\x7f")
	d2 := &recordDoer{steps: []step{{resp: r2}}}
	if _, err := Get("https://h/x", RequestOptions{Transport: d2}); !errors.Is(err, ErrUnsupportedURIScheme) {
		t.Fatalf("want loc parse error, got %v", err)
	}
}

func TestNilResponseHeadersTolerated(t *testing.T) {
	d := &recordDoer{steps: []step{{resp: &RawResponse{Code: 200, Body: "hi"}}}} // Headers nil
	resp, err := Get("https://h/x", RequestOptions{Transport: d})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Code() != 200 {
		t.Fatalf("code=%d", resp.Code())
	}
}

func TestFollowRedirectsExplicitTrue(t *testing.T) {
	r1 := raw(301, "", "")
	r1.Headers.Set("Location", "/two")
	d := &recordDoer{steps: []step{{resp: r1}, {resp: raw(200, "text/plain", "x")}}}
	resp, err := Get("https://h/one", RequestOptions{FollowRedirects: boolp(true), Transport: d})
	if err != nil || resp.Code() != 200 {
		t.Fatalf("resp=%v err=%v", resp, err)
	}
}

func TestClientFormatFallback(t *testing.T) {
	// Client format applies when the per-call format is empty; forced format
	// overrides the Content-Type (here a JSON body served as text/plain).
	d := single(raw(200, "text/plain", `{"k":"v"}`))
	api := NewClient().Format("json").Adapter(d)
	resp, err := api.Get("https://h/x")
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := resp.Parsed(); !reflect.DeepEqual(v, map[string]any{"k": "v"}) {
		t.Fatalf("parsed=%#v", v)
	}
}

func TestNewClientNilBlock(t *testing.T) {
	c := NewClient(nil)
	if c.baseURI != "" {
		t.Fatalf("nil block should be a no-op")
	}
}

// TestAllVerbs exercises every module-level verb and its Client counterpart, all
// against the stub transport.
func TestAllVerbs(t *testing.T) {
	newStub := func() *recordDoer { return single(raw(200, "text/plain", "ok")) }

	modVerbs := []func(string, ...RequestOptions) (*Response, error){Get, Head, Options, Post, Put, Patch, Delete}
	for _, v := range modVerbs {
		if _, err := v("https://h/x", RequestOptions{Transport: newStub()}); err != nil {
			t.Fatal(err)
		}
	}

	c := NewClient()
	cliVerbs := []func(string, ...RequestOptions) (*Response, error){c.Get, c.Head, c.Options, c.Post, c.Put, c.Patch, c.Delete}
	for _, v := range cliVerbs {
		if _, err := v("https://h/x", RequestOptions{Transport: newStub()}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResponseSuccessRange(t *testing.T) {
	if r := newResponse(raw(204, "", ""), ""); !r.Success() {
		t.Fatal("204 should be success")
	}
	if r := newResponse(raw(404, "", ""), ""); r.Success() {
		t.Fatal("404 should not be success")
	}
	r := newResponse(raw(200, "", ""), "")
	if r.Headers() == nil {
		t.Fatal("headers accessor")
	}
}
