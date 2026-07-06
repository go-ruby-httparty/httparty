// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

import (
	"reflect"
	"testing"
)

func TestDeriveFormat(t *testing.T) {
	cases := map[string]string{
		"application/json":                 "json",
		"text/json":                        "json",
		"application/vnd.api+json":         "json",
		"application/json; charset=utf-8":  "json",
		"text/xml":                         "xml",
		"application/xml":                  "xml",
		"application/rss+xml":              "xml",
		"text/html":                        "",
		"text/plain":                       "",
		"application/octet-stream":         "",
		"":                                 "",
		"  Application/JSON ; boundary=x ": "json",
	}
	for ct, want := range cases {
		if got := deriveFormat(ct); got != want {
			t.Fatalf("deriveFormat(%q)=%q want %q", ct, got, want)
		}
	}
}

func TestNormalizeParseFormat(t *testing.T) {
	for in, want := range map[string]string{"json": "json", "xml": "xml", "html": "", "csv": "", "plain": "", "": ""} {
		if got := normalizeParseFormat(in); got != want {
			t.Fatalf("normalizeParseFormat(%q)=%q want %q", in, got, want)
		}
	}
}

func TestSupportedFormat(t *testing.T) {
	for _, ok := range []string{"json", "xml", "html", "csv", "plain"} {
		if !supportedFormat(ok) {
			t.Fatalf("%q should be supported", ok)
		}
	}
	if supportedFormat("yaml") {
		t.Fatal("yaml should be unsupported")
	}
}

func TestParseBody(t *testing.T) {
	// Blank body returns verbatim.
	if v, err := parseBody("json", "   "); err != nil || v != "   " {
		t.Fatalf("blank: %v %v", v, err)
	}
	// JSON scalar / array / object.
	if v, _ := parseBody("json", `"hi"`); v != "hi" {
		t.Fatalf("json scalar=%v", v)
	}
	if v, _ := parseBody("json", `[1,2]`); !reflect.DeepEqual(v, []any{float64(1), float64(2)}) {
		t.Fatalf("json array=%v", v)
	}
	// JSON error.
	if _, err := parseBody("json", `{bad`); err == nil {
		t.Fatal("want json error")
	}
	// XML ok.
	if v, err := parseBody("xml", `<r><a>1</a></r>`); err != nil ||
		!reflect.DeepEqual(v, map[string]any{"r": map[string]any{"a": "1"}}) {
		t.Fatalf("xml=%v err=%v", v, err)
	}
	// XML error.
	if _, err := parseBody("xml", `<a><b></c></a>`); err == nil {
		t.Fatal("want xml error")
	}
	// Unknown format -> raw.
	if v, _ := parseBody("", "raw-text"); v != "raw-text" {
		t.Fatalf("raw=%v", v)
	}
}

func TestParseXMLShapes(t *testing.T) {
	// Leaf text.
	if v, _ := parseXML(`<name>Ada</name>`); !reflect.DeepEqual(v, map[string]any{"name": "Ada"}) {
		t.Fatalf("leaf=%#v", v)
	}
	// Nested map, repeated child -> slice (3 to exercise the append branch).
	v, err := parseXML(`<list><i>1</i><i>2</i><i>3</i></list>`)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"list": map[string]any{"i": []any{"1", "2", "3"}}}
	if !reflect.DeepEqual(v, want) {
		t.Fatalf("repeat=%#v", v)
	}
	// Attributes + mixed content -> __content__.
	v2, _ := parseXML(`<a id="7">hello</a>`)
	if !reflect.DeepEqual(v2, map[string]any{"a": map[string]any{"id": "7", "__content__": "hello"}}) {
		t.Fatalf("attr=%#v", v2)
	}
	// Attributes with only whitespace text: no __content__ key.
	v3, _ := parseXML(`<a id="7">  </a>`)
	if !reflect.DeepEqual(v3, map[string]any{"a": map[string]any{"id": "7"}}) {
		t.Fatalf("attr-blank=%#v", v3)
	}
	// Child element with no stray text.
	v4, _ := parseXML(`<a><b>x</b></a>`)
	if !reflect.DeepEqual(v4, map[string]any{"a": map[string]any{"b": "x"}}) {
		t.Fatalf("child=%#v", v4)
	}
}

func TestParseXMLEmptyAndErrors(t *testing.T) {
	// Non-blank but element-free (comment only) -> empty map (EOF, no root).
	if v, err := parseXML(`<!-- just a comment -->`); err != nil ||
		!reflect.DeepEqual(v, map[string]any{}) {
		t.Fatalf("comment-only=%#v err=%v", v, err)
	}
	// Token error before any root element.
	if _, err := parseXML(`<`); err == nil {
		t.Fatal("want token error before root")
	}
	// Token error inside a (top-level) element: unclosed root.
	if _, err := parseXML(`<a>`); err == nil {
		t.Fatal("want token error inside element")
	}
}
