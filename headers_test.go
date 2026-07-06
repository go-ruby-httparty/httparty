// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

import (
	"reflect"
	"testing"
)

func TestHeadersSetGetCaseInsensitive(t *testing.T) {
	h := HeadersOf([2]string{"Content-Type", "text/plain"})
	if v, ok := h.Get("content-type"); !ok || v != "text/plain" {
		t.Fatalf("get=%q ok=%v", v, ok)
	}
	// Set overwrites in place, preserving the original key casing.
	h.Set("CONTENT-TYPE", "application/json")
	if h.Len() != 1 {
		t.Fatalf("len=%d", h.Len())
	}
	if h.Pairs()[0].Key != "Content-Type" || h.Pairs()[0].Val != "application/json" {
		t.Fatalf("pairs=%+v", h.Pairs())
	}
	if _, ok := h.Get("missing"); ok {
		t.Fatal("missing should be absent")
	}
}

func TestHeadersMultiValue(t *testing.T) {
	h := NewHeaders()
	h.Add("Set-Cookie", "a=1")
	h.Add("set-cookie", "b=2")
	if vs := h.Values("Set-Cookie"); !reflect.DeepEqual(vs, []string{"a=1", "b=2"}) {
		t.Fatalf("values=%v", vs)
	}
	if v, _ := h.Get("Set-Cookie"); v != "a=1, b=2" {
		t.Fatalf("joined=%q", v)
	}
	// Set collapses the repeats to a single value.
	h.Set("set-cookie", "c=3")
	if vs := h.Values("Set-Cookie"); !reflect.DeepEqual(vs, []string{"c=3"}) {
		t.Fatalf("after set=%v", vs)
	}
}

func TestHeadersSetDefaultAndHasAndDelete(t *testing.T) {
	h := HeadersOf([2]string{"A", "1"})
	h.SetDefault("A", "override") // present -> unchanged
	h.SetDefault("B", "2")        // absent -> added
	if v, _ := h.Get("A"); v != "1" {
		t.Fatalf("A=%q", v)
	}
	if !h.Has("b") {
		t.Fatal("B should be present")
	}
	if h.Has("zzz") {
		t.Fatal("zzz absent")
	}
	h.Delete("a")
	if h.Has("A") {
		t.Fatal("A should be deleted")
	}
	if vs := h.Values("A"); vs != nil {
		t.Fatalf("deleted values=%v", vs)
	}
}

func TestHeadersCloneAndMerge(t *testing.T) {
	h := HeadersOf([2]string{"A", "1"}, [2]string{"B", "2"})
	c := h.Clone()
	c.Set("A", "changed")
	if v, _ := h.Get("A"); v != "1" {
		t.Fatalf("clone leaked: %q", v)
	}
	m := h.Merge(HeadersOf([2]string{"B", "20"}, [2]string{"C", "3"}))
	if v, _ := m.Get("B"); v != "20" {
		t.Fatalf("merge B=%q", v)
	}
	if v, _ := m.Get("C"); v != "3" {
		t.Fatalf("merge C=%q", v)
	}
	// Merge(nil) is a no-op clone.
	if h.Merge(nil).Len() != 2 {
		t.Fatal("merge nil")
	}
}
