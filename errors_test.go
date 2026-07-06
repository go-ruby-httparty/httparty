// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorTreeMatching(t *testing.T) {
	// A RedirectionTooDeep matches itself, its parent ResponseError, and the root.
	e := newResponseError(KindRedirectionTooDeep, "too deep", nil)
	for _, sentinel := range []*Error{ErrRedirectionTooDeep, ErrResponseError, ErrError} {
		if !errors.Is(e, sentinel) {
			t.Fatalf("%s should match %s", e.Kind, sentinel.Kind)
		}
	}
	// It does not match a sibling.
	if errors.Is(e, ErrDuplicateLocationHeader) {
		t.Fatal("should not match sibling")
	}
	// UnsupportedFormat matches only itself and the root, not ResponseError.
	uf := &Error{Kind: KindUnsupportedFormat, Message: "x"}
	if errors.Is(uf, ErrResponseError) {
		t.Fatal("UnsupportedFormat is not a ResponseError")
	}
	if !errors.Is(uf, ErrError) {
		t.Fatal("everything is an Error")
	}
}

func TestErrorIsNonError(t *testing.T) {
	e := &Error{Kind: KindError, Message: "x"}
	if e.Is(fmt.Errorf("plain")) {
		t.Fatal("a plain error is not a match target")
	}
	if isKind(fmt.Errorf("plain"), ErrError) {
		t.Fatal("isKind on non-*Error must be false")
	}
}

func TestErrorMessageAndUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	e := &Error{Kind: KindError, Message: "boom", Cause: cause}
	if e.Error() != "boom" {
		t.Fatalf("message=%q", e.Error())
	}
	if !errors.Is(e, cause) {
		t.Fatal("Unwrap should expose the cause")
	}
}

func TestIsResponseError(t *testing.T) {
	if !IsResponseError(newResponseError(KindDuplicateLocationHeader, "dup", nil)) {
		t.Fatal("DuplicateLocationHeader is a ResponseError")
	}
	if IsResponseError(&Error{Kind: KindUnsupportedFormat}) {
		t.Fatal("UnsupportedFormat is not a ResponseError")
	}
}
