// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

import "strings"

// Headers is a case-insensitive, insertion-ordered, multi-valued header map. A
// lookup matches keys case-insensitively while the original casing of the key
// is preserved for iteration and display. Multiple values for the same field
// are retained (like Net::HTTP's get_fields): [Headers.Get] joins them with
// ", " (the value HTTParty exposes via response.headers['field']) while
// [Headers.Values] returns them individually — which is how a duplicate
// Location header on a redirect is detected.
type Headers struct {
	pairs []Pair // ordered; may hold several entries with the same (folded) key
}

// NewHeaders returns an empty [Headers].
func NewHeaders() *Headers { return &Headers{} }

// HeadersOf builds a [Headers] from ordered key/value pairs (each via [Headers.Set]).
func HeadersOf(kv ...[2]string) *Headers {
	h := NewHeaders()
	for _, e := range kv {
		h.Set(e[0], e[1])
	}
	return h
}

// Len reports the number of stored header entries (counting repeats).
func (h *Headers) Len() int { return len(h.pairs) }

// Pairs returns the header entries in insertion order. The slice must not be
// mutated.
func (h *Headers) Pairs() []Pair { return h.pairs }

// Set assigns a single value to key: the first existing entry for key
// (case-insensitive) is updated in place and any further entries for key are
// dropped, so key ends up with exactly one value. A previously absent key is
// appended.
func (h *Headers) Set(key, val string) {
	lk := strings.ToLower(key)
	first := -1
	kept := h.pairs[:0]
	for _, p := range h.pairs {
		if strings.ToLower(p.Key) != lk {
			kept = append(kept, p)
			continue
		}
		if first == -1 {
			first = len(kept)
			p.Val = val
			kept = append(kept, p)
		}
	}
	h.pairs = kept
	if first == -1 {
		h.pairs = append(h.pairs, Pair{Key: key, Val: val})
	}
}

// Add appends another value for key without removing existing ones, mirroring a
// server sending the same header field twice.
func (h *Headers) Add(key, val string) {
	h.pairs = append(h.pairs, Pair{Key: key, Val: val})
}

// SetDefault assigns key→val only when key is absent (case-insensitive), so a
// caller-supplied value is never clobbered.
func (h *Headers) SetDefault(key, val string) {
	if !h.Has(key) {
		h.Set(key, val)
	}
}

// Get returns the ", "-joined values for key (case-insensitive) and whether the
// key was present, matching how HTTParty reads response.headers['field'].
func (h *Headers) Get(key string) (string, bool) {
	vals := h.Values(key)
	if len(vals) == 0 {
		return "", false
	}
	return strings.Join(vals, ", "), true
}

// Values returns every value stored for key (case-insensitive) in order.
func (h *Headers) Values(key string) []string {
	lk := strings.ToLower(key)
	var out []string
	for _, p := range h.pairs {
		if strings.ToLower(p.Key) == lk {
			out = append(out, p.Val)
		}
	}
	return out
}

// Has reports whether key is present (case-insensitive).
func (h *Headers) Has(key string) bool {
	lk := strings.ToLower(key)
	for _, p := range h.pairs {
		if strings.ToLower(p.Key) == lk {
			return true
		}
	}
	return false
}

// Delete removes every entry for key (case-insensitive), keeping the order of
// the remaining headers.
func (h *Headers) Delete(key string) {
	lk := strings.ToLower(key)
	kept := h.pairs[:0]
	for _, p := range h.pairs {
		if strings.ToLower(p.Key) != lk {
			kept = append(kept, p)
		}
	}
	h.pairs = kept
}

// Clone returns a copy of h preserving order and repeats.
func (h *Headers) Clone() *Headers {
	c := NewHeaders()
	c.pairs = append(c.pairs, h.pairs...)
	return c
}

// Merge overlays other's headers onto a copy of h with [Headers.Set] semantics
// (each key in other replaces h's value for that key) and returns the result.
func (h *Headers) Merge(other *Headers) *Headers {
	out := h.Clone()
	if other != nil {
		for _, e := range other.pairs {
			out.Set(e.Key, e.Val)
		}
	}
	return out
}
