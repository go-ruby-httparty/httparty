// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

import "testing"

func TestParamsOrderSetOverwriteGet(t *testing.T) {
	p := ParamsOf([2]string{"b", "2"}, [2]string{"a", "1"})
	p.Set("b", "20") // overwrite keeps position
	if p.Len() != 2 {
		t.Fatalf("len=%d", p.Len())
	}
	if p.Pairs()[0].Key != "b" || p.Pairs()[0].Val != "20" {
		t.Fatalf("pairs=%+v", p.Pairs())
	}
	if v, ok := p.Get("a"); !ok || v != "1" {
		t.Fatalf("get a=%q ok=%v", v, ok)
	}
	if _, ok := p.Get("zzz"); ok {
		t.Fatal("zzz absent")
	}
	if !p.Has("a") || p.Has("zzz") {
		t.Fatal("has")
	}
}

func TestParamsZeroValueSet(t *testing.T) {
	// A zero-value Params (no NewParams) lazily initialises its index on Set.
	var p Params
	p.Set("a", "1")
	if v, ok := p.Get("a"); !ok || v != "1" {
		t.Fatalf("zero-value set: %q %v", v, ok)
	}
}

func TestParamsDeleteReindex(t *testing.T) {
	p := ParamsOf([2]string{"a", "1"}, [2]string{"b", "2"}, [2]string{"c", "3"})
	p.Delete("a")
	p.Delete("missing") // no-op
	if p.Len() != 2 {
		t.Fatalf("len=%d", p.Len())
	}
	// After reindex, b and c are still addressable.
	if v, _ := p.Get("c"); v != "3" {
		t.Fatalf("c=%q", v)
	}
	p.Set("c", "30")
	if v, _ := p.Get("c"); v != "30" {
		t.Fatalf("reindexed c=%q", v)
	}
}

func TestParamsCloneMergeEncode(t *testing.T) {
	p := ParamsOf([2]string{"a", "1"})
	c := p.Clone()
	c.Set("a", "9")
	if v, _ := p.Get("a"); v != "1" {
		t.Fatalf("clone leaked: %q", v)
	}
	m := p.Merge(ParamsOf([2]string{"a", "2"}, [2]string{"b", "hi there"}))
	if v, _ := m.Get("a"); v != "2" {
		t.Fatalf("merge a=%q", v)
	}
	if p.Merge(nil).Len() != 1 {
		t.Fatal("merge nil")
	}
	if got := m.Encode(); got != "a=2&b=hi%20there" {
		t.Fatalf("encode=%q", got)
	}
}
