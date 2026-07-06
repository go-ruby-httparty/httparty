// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"
)

// This file models HTTParty::Parser: mapping a response Content-Type to a
// format, and parsing a body in that format. HTTParty supports the formats
// :json, :xml, :html, :csv and :plain; a JSON body parses to a Hash/Array/scalar
// and an XML body to a nested Hash, while html/csv/plain (and any unrecognised
// type) yield the raw body string — exactly the surface [Response.Parsed]
// exposes.

// supportedFormatNames are the format symbols HTTParty's Parser recognises. A
// :format option outside this set raises HTTParty::UnsupportedFormat.
var supportedFormatNames = map[string]bool{
	"json":  true,
	"xml":   true,
	"html":  true,
	"csv":   true,
	"plain": true,
}

// supportedFormat reports whether name is a format HTTParty's Parser supports.
func supportedFormat(name string) bool { return supportedFormatNames[name] }

// normalizeParseFormat maps a supported format name to the parser this package
// applies: "json" and "xml" parse structurally; html/csv/plain (and anything
// else) yield the raw body, so they normalise to "" (no structural parse).
func normalizeParseFormat(name string) string {
	switch name {
	case "json":
		return "json"
	case "xml":
		return "xml"
	default:
		return ""
	}
}

// deriveFormat maps a response Content-Type (with optional parameters) to the
// parse format, mirroring HTTParty::Parser.format_from_mimetype over its
// SupportedFormats table: application/json, text/json and any "+json" type parse
// as JSON; text/xml, application/xml and any "+xml" type parse as XML; every
// other media type (html, csv, plain, javascript, unknown) yields the raw body,
// i.e. "".
func deriveFormat(contentType string) string {
	ct := contentType
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	ct = strings.ToLower(strings.TrimSpace(ct))
	switch {
	case ct == "application/json" || ct == "text/json" || strings.HasSuffix(ct, "+json"):
		return "json"
	case ct == "application/xml" || ct == "text/xml" || strings.HasSuffix(ct, "+xml"):
		return "xml"
	default:
		return ""
	}
}

// parseBody parses body in the given normalised format ("json", "xml", or "" for
// raw). A blank body is returned verbatim (HTTParty parses nothing from a blank
// body). A malformed JSON or XML body returns an HTTParty::Error(ParsingError-
// like) via the returned error, matching HTTParty's parser raising on bad input.
func parseBody(format, body string) (any, error) {
	if strings.TrimSpace(body) == "" {
		return body, nil
	}
	switch format {
	case "json":
		var v any
		if err := json.Unmarshal([]byte(body), &v); err != nil {
			return nil, &Error{Kind: KindError, Message: err.Error(), Cause: err}
		}
		return v, nil
	case "xml":
		v, err := parseXML(body)
		if err != nil {
			return nil, &Error{Kind: KindError, Message: err.Error(), Cause: err}
		}
		return v, nil
	default:
		return body, nil
	}
}

// parseXML parses an XML document into a nested map[string]any keyed by the root
// element name, modelling the Hash HTTParty produces for an XML body: a
// leaf element becomes its (trimmed) text, an element with child elements
// becomes a map of child-name→value, a repeated child name becomes a slice, and
// attributes are folded in as string-valued keys (with mixed text kept under
// "__content__").
func parseXML(s string) (any, error) {
	dec := xml.NewDecoder(strings.NewReader(s))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return map[string]any{}, nil
		}
		if err != nil {
			return nil, err
		}
		if se, ok := tok.(xml.StartElement); ok {
			v, err := parseElement(dec, se)
			if err != nil {
				return nil, err
			}
			return map[string]any{se.Name.Local: v}, nil
		}
	}
}

// parseElement consumes the children of se up to its end tag and returns the
// value for it: a map when it has attributes or child elements, otherwise its
// trimmed text.
func parseElement(dec *xml.Decoder, se xml.StartElement) (any, error) {
	children := map[string]any{}
	var text strings.Builder
	hasMap := false
	for _, a := range se.Attr {
		children[a.Name.Local] = a.Value
		hasMap = true
	}
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			hasMap = true
			cv, err := parseElement(dec, t)
			if err != nil {
				return nil, err
			}
			addChild(children, t.Name.Local, cv)
		case xml.CharData:
			text.Write(t)
		case xml.EndElement:
			if !hasMap {
				return strings.TrimSpace(text.String()), nil
			}
			if txt := strings.TrimSpace(text.String()); txt != "" {
				children["__content__"] = txt
			}
			return children, nil
		}
	}
}

// addChild inserts value v under key k, promoting to a slice when k repeats
// (mirroring how repeated XML elements become an array).
func addChild(m map[string]any, k string, v any) {
	ex, ok := m[k]
	if !ok {
		m[k] = v
		return
	}
	if arr, ok := ex.([]any); ok {
		m[k] = append(arr, v)
		return
	}
	m[k] = []any{ex, v}
}
