// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

import "testing"

func TestEscape(t *testing.T) {
	cases := map[string]string{
		"plain.text-_~": "plain.text-_~",
		"hello world":   "hello%20world",
		"a&b=c":         "a%26b%3Dc",
		"é":             "%C3%A9",
		"/?#[]@":        "%2F%3F%23%5B%5D%40",
	}
	for in, want := range cases {
		if got := Escape(in); got != want {
			t.Fatalf("Escape(%q)=%q want %q", in, got, want)
		}
	}
}

func TestUnescape(t *testing.T) {
	cases := map[string]string{
		"plain":         "plain", // fast path (no % or +)
		"hello%20world": "hello world",
		"a+b":           "a b",
		"%C3%A9":        "é",
		"bad%2":         "bad%2",       // truncated -> literal
		"bad%zz more":   "bad%zz more", // invalid hex -> literal '%'
		"a%2bb":         "a+b",         // lowercase hex digit
		"%c3%a9":        "é",           // lowercase hex bytes
	}
	for in, want := range cases {
		if got := Unescape(in); got != want {
			t.Fatalf("Unescape(%q)=%q want %q", in, got, want)
		}
	}
}

func TestBuildQueryAndParseQuery(t *testing.T) {
	p := ParamsOf([2]string{"b", "2"}, [2]string{"a", "x y"}, [2]string{"c", "a&b"})
	if got := BuildQuery(p); got != "b=2&a=x%20y&c=a%26b" {
		t.Fatalf("BuildQuery=%q", got)
	}
	// ParseQuery: leading '?', empty segment skipped, bare key, duplicate wins.
	q := ParseQuery("?a=1&&b&a=2&c=x%26y")
	if v, _ := q.Get("a"); v != "2" {
		t.Fatalf("a=%q", v)
	}
	if v, ok := q.Get("b"); !ok || v != "" {
		t.Fatalf("bare key b=%q ok=%v", v, ok)
	}
	if v, _ := q.Get("c"); v != "x&y" {
		t.Fatalf("c=%q", v)
	}
	if ParseQuery("").Len() != 0 {
		t.Fatal("empty query")
	}
}

func TestBasicHeaderFrom(t *testing.T) {
	if got := BasicHeaderFrom("aladdin", "opensesame"); got != "Basic YWxhZGRpbjpvcGVuc2VzYW1l" {
		t.Fatalf("basic=%q", got)
	}
}
