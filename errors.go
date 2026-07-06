// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

// Error is the root of HTTParty's error tree (HTTParty::Error < StandardError).
// The concrete kinds are distinguished by [Error.Kind]; the response-carrying
// errors (the HTTParty::ResponseError subtree) keep the [Response] that caused
// them. Match with errors.Is against the Err* sentinels, where a superclass
// matches its subclasses — mirroring Ruby's rescue of an HTTParty::Error
// subclass.
type Error struct {
	// Kind names the specific HTTParty error subclass (see the Err* sentinels).
	Kind ErrorKind
	// Message is the error text (HTTParty::Error#message).
	Message string
	// Response is the response context for the ResponseError subtree
	// (RedirectionTooDeep / DuplicateLocationHeader); nil otherwise.
	Response *Response
	// Cause is an underlying error, if any (e.g. a transport or JSON failure).
	Cause error
}

// ErrorKind identifies an HTTParty error subclass.
type ErrorKind string

// The HTTParty error subclasses, named as in the gem.
const (
	KindError                   ErrorKind = "HTTParty::Error"
	KindUnsupportedFormat       ErrorKind = "HTTParty::UnsupportedFormat"
	KindUnsupportedURIScheme    ErrorKind = "HTTParty::UnsupportedURIScheme"
	KindResponseError           ErrorKind = "HTTParty::ResponseError"
	KindRedirectionTooDeep      ErrorKind = "HTTParty::RedirectionTooDeep"
	KindDuplicateLocationHeader ErrorKind = "HTTParty::DuplicateLocationHeader"
)

// Sentinel errors for errors.Is matching. Each names an HTTParty error kind; a
// concrete [Error] with that Kind (or a subtree of it) matches via [Error.Is].
var (
	ErrError                   = &Error{Kind: KindError, Message: string(KindError)}
	ErrUnsupportedFormat       = &Error{Kind: KindUnsupportedFormat, Message: string(KindUnsupportedFormat)}
	ErrUnsupportedURIScheme    = &Error{Kind: KindUnsupportedURIScheme, Message: string(KindUnsupportedURIScheme)}
	ErrResponseError           = &Error{Kind: KindResponseError, Message: string(KindResponseError)}
	ErrRedirectionTooDeep      = &Error{Kind: KindRedirectionTooDeep, Message: string(KindRedirectionTooDeep)}
	ErrDuplicateLocationHeader = &Error{Kind: KindDuplicateLocationHeader, Message: string(KindDuplicateLocationHeader)}
)

// errorParents maps each kind to its parent kind in the HTTParty hierarchy.
// UnsupportedFormat, UnsupportedURIScheme and ResponseError descend directly
// from Error; RedirectionTooDeep and DuplicateLocationHeader descend from
// ResponseError. The root, KindError, has no parent.
var errorParents = map[ErrorKind]ErrorKind{
	KindUnsupportedFormat:       KindError,
	KindUnsupportedURIScheme:    KindError,
	KindResponseError:           KindError,
	KindRedirectionTooDeep:      KindResponseError,
	KindDuplicateLocationHeader: KindResponseError,
}

// Error implements the error interface (HTTParty::Error#message).
func (e *Error) Error() string { return e.Message }

// Unwrap exposes the underlying cause for errors.Is/As.
func (e *Error) Unwrap() error { return e.Cause }

// Is reports whether e matches target: true when target is a [*Error] whose
// Kind is e's Kind or an ancestor of it, so errors.Is(err, ErrResponseError)
// matches a RedirectionTooDeep, and errors.Is(err, ErrError) matches every
// HTTParty error — mirroring Ruby's rescue of a superclass.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	for k := e.Kind; ; {
		if k == t.Kind {
			return true
		}
		parent, ok := errorParents[k]
		if !ok {
			return false
		}
		k = parent
	}
}

// newResponseError builds a ResponseError-subtree [Error] carrying the response
// context, mirroring how HTTParty raises RedirectionTooDeep /
// DuplicateLocationHeader with the offending response.
func newResponseError(kind ErrorKind, msg string, resp *Response) *Error {
	return &Error{Kind: kind, Message: msg, Response: resp}
}

// IsResponseError reports whether err is an HTTParty::ResponseError (or subclass).
func IsResponseError(err error) bool { return isKind(err, ErrResponseError) }

// isKind is the errors.Is shim used by the predicate helpers.
func isKind(err error, sentinel *Error) bool {
	e, ok := err.(*Error)
	return ok && e.Is(sentinel)
}
