// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// fakeHTTP is a stand-in for the net/http client so [NetHTTPDoer] can be driven
// without opening a socket.
type fakeHTTP struct {
	resp *http.Response
	err  error
	got  *http.Request
}

func (f *fakeHTTP) Do(req *http.Request) (*http.Response, error) {
	f.got = req
	return f.resp, f.err
}

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) { return 0, errors.New("read boom") }
func (errReadCloser) Close() error             { return nil }

func TestResolveDoer(t *testing.T) {
	// Distinguish the two candidate transports by the body they return.
	perCall := DoerFunc(func(*Request) (*RawResponse, error) { return raw(200, "", "per-call"), nil })
	client := DoerFunc(func(*Request) (*RawResponse, error) { return raw(200, "", "client"), nil })
	tag := func(d Doer) string { r, _ := d.Call(&Request{}); return r.Body }

	if got := tag(resolveDoer(perCall, client)); got != "per-call" {
		t.Fatalf("per-call transport should win, got %q", got)
	}
	if got := tag(resolveDoer(nil, client)); got != "client" {
		t.Fatalf("client transport should be used, got %q", got)
	}
	if _, ok := resolveDoer(nil, nil).(*NetHTTPDoer); !ok {
		t.Fatal("both nil should fall back to NetHTTP")
	}
}

func TestNetHTTPConstructor(t *testing.T) {
	d := NetHTTP()
	hc, ok := d.Client.(*http.Client)
	if !ok {
		t.Fatal("default client should be *http.Client")
	}
	if err := hc.CheckRedirect(nil, nil); err != http.ErrUseLastResponse {
		t.Fatalf("CheckRedirect=%v", err)
	}
}

func TestNetHTTPDoerCallSuccess(t *testing.T) {
	hdr := http.Header{}
	hdr.Add("Content-Type", "application/json")
	hdr.Add("X-Multi", "a")
	hdr.Add("X-Multi", "b")
	fake := &fakeHTTP{resp: &http.Response{
		StatusCode: 200,
		Header:     hdr,
		Body:       io.NopCloser(strings.NewReader("hello")),
	}}
	d := &NetHTTPDoer{Client: fake}
	req := &Request{Method: "POST", URL: "https://h/x", Headers: HeadersOf([2]string{"Accept", "text/plain"}), Body: "payload"}
	raw, err := d.Call(req)
	if err != nil {
		t.Fatal(err)
	}
	if raw.Code != 200 || raw.Body != "hello" {
		t.Fatalf("raw=%+v", raw)
	}
	if vs := raw.Headers.Values("X-Multi"); len(vs) != 2 {
		t.Fatalf("multi header not preserved: %v", vs)
	}
	// The request the fake saw carried our header and body.
	if fake.got.Header.Get("Accept") != "text/plain" {
		t.Fatalf("outgoing header missing")
	}
}

func TestNetHTTPDoerCallNewRequestError(t *testing.T) {
	d := &NetHTTPDoer{Client: &fakeHTTP{}}
	req := &Request{Method: "BAD METHOD", URL: "https://h/x", Headers: NewHeaders(), Body: ""}
	if _, err := d.Call(req); err == nil || !errors.Is(err, ErrError) {
		t.Fatalf("want NewRequest error, got %v", err)
	}
}

func TestNetHTTPDoerCallDoError(t *testing.T) {
	d := &NetHTTPDoer{Client: &fakeHTTP{err: errors.New("dial fail")}}
	req := &Request{Method: "GET", URL: "https://h/x", Headers: NewHeaders(), Body: "b"}
	if _, err := d.Call(req); err == nil {
		t.Fatalf("want Do error")
	}
}

func TestNetHTTPDoerCallReadError(t *testing.T) {
	fake := &fakeHTTP{resp: &http.Response{StatusCode: 200, Header: http.Header{}, Body: errReadCloser{}}}
	d := &NetHTTPDoer{Client: fake}
	req := &Request{Method: "GET", URL: "https://h/x", Headers: NewHeaders(), Body: ""}
	if _, err := d.Call(req); err == nil {
		t.Fatalf("want read error")
	}
}

func TestDoerFuncCall(t *testing.T) {
	called := false
	f := DoerFunc(func(*Request) (*RawResponse, error) { called = true; return raw(200, "", ""), nil })
	if _, err := f.Call(&Request{}); err != nil || !called {
		t.Fatalf("DoerFunc.Call not invoked")
	}
}

func TestNoRedirect(t *testing.T) {
	if err := noRedirect(nil, nil); err != http.ErrUseLastResponse {
		t.Fatalf("noRedirect=%v", err)
	}
}
