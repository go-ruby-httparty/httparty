// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

import (
	"encoding/base64"
	"strings"
)

// The query codec mirrors how HTTParty turns a query Hash into a query string:
// HTTParty's HashConversions.to_params percent-encodes each value with
// ERB::Util.url_encode and joins "key=value" pairs with "&", in the Hash's
// order (HTTParty does not sort — an ordered [Params] reproduces that order).
// [Escape] reproduces ERB::Util.url_encode byte-for-byte: the unreserved set
// [A-Za-z0-9_.\-~] is left literal and every other byte (including space)
// becomes %XX with upper-case hex.

// BuildQuery renders params as a query string: the params are emitted in
// insertion order (HTTParty preserves the Hash order rather than sorting) and
// each key and value is run through [Escape].
func BuildQuery(params *Params) string {
	var b strings.Builder
	for i, p := range params.pairs {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(Escape(p.Key))
		b.WriteByte('=')
		b.WriteString(Escape(p.Val))
	}
	return b.String()
}

// ParseQuery decodes an application/x-www-form-urlencoded query string into an
// ordered [Params]: each key and value is [Unescape]d, a bare key (no '=') maps
// to the empty string, an empty segment is skipped, and a later duplicate key
// overwrites an earlier one (keeping its position). A leading '?' is ignored.
func ParseQuery(query string) *Params {
	out := NewParams()
	query = strings.TrimPrefix(query, "?")
	if query == "" {
		return out
	}
	for _, seg := range strings.Split(query, "&") {
		if seg == "" {
			continue
		}
		k, v, _ := strings.Cut(seg, "=")
		out.Set(Unescape(k), Unescape(v))
	}
	return out
}

// Escape percent-encodes s exactly as Ruby's ERB::Util.url_encode does (the
// encoder HTTParty uses for query values): the unreserved set [A-Za-z0-9_.\-~]
// is left literal and every other byte — including a space — becomes %XX with
// upper-case hex.
func Escape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escapeUnreserved(c) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hexDigit(c >> 4))
		b.WriteByte(hexDigit(c & 0xf))
	}
	return b.String()
}

// Unescape reverses [Escape] and also decodes the '+'-for-space form some
// servers emit: '+' becomes a space and %XX becomes its byte. An invalid or
// truncated %XX is left literal, matching a tolerant decoder.
func Unescape(s string) string {
	if !strings.ContainsAny(s, "%+") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '+':
			b.WriteByte(' ')
		case s[i] == '%' && i+2 < len(s):
			hi, ok1 := fromHex(s[i+1])
			lo, ok2 := fromHex(s[i+2])
			if ok1 && ok2 {
				b.WriteByte(hi<<4 | lo)
				i += 2
				continue
			}
			b.WriteByte('%')
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// BasicHeaderFrom returns the HTTP Basic Authorization header value for a login
// and password: "Basic " followed by the newline-free base64 of
// "login:password", matching HTTParty's basic_auth handling.
func BasicHeaderFrom(login, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(login+":"+password))
}

// escapeUnreserved reports whether c is left literal by [Escape]: the
// ERB::Util.url_encode unreserved set [A-Za-z0-9_.\-~].
func escapeUnreserved(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		return true
	case c == '-', c == '_', c == '.', c == '~':
		return true
	}
	return false
}

// hexDigit maps a nibble (0..15) to its upper-case hexadecimal ASCII digit.
func hexDigit(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'A' + (n - 10)
}

// fromHex parses a single hexadecimal ASCII digit.
func fromHex(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}
