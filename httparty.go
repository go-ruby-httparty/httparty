// Copyright (c) the go-ruby-httparty/httparty authors
//
// SPDX-License-Identifier: BSD-3-Clause

package httparty

// defaultClient backs the package-level verb functions, modelling the HTTParty
// module methods (HTTParty.get, …) which issue requests with no class-level
// configuration.
var defaultClient = &Client{}

// Get issues a GET request with no base configuration, mirroring HTTParty.get.
func Get(url string, opts ...RequestOptions) (*Response, error) {
	return defaultClient.Get(url, opts...)
}

// Head issues a HEAD request, mirroring HTTParty.head.
func Head(url string, opts ...RequestOptions) (*Response, error) {
	return defaultClient.Head(url, opts...)
}

// Options issues an OPTIONS request, mirroring HTTParty.options.
func Options(url string, opts ...RequestOptions) (*Response, error) {
	return defaultClient.Options(url, opts...)
}

// Post issues a POST request, mirroring HTTParty.post.
func Post(url string, opts ...RequestOptions) (*Response, error) {
	return defaultClient.Post(url, opts...)
}

// Put issues a PUT request, mirroring HTTParty.put.
func Put(url string, opts ...RequestOptions) (*Response, error) {
	return defaultClient.Put(url, opts...)
}

// Patch issues a PATCH request, mirroring HTTParty.patch.
func Patch(url string, opts ...RequestOptions) (*Response, error) {
	return defaultClient.Patch(url, opts...)
}

// Delete issues a DELETE request, mirroring HTTParty.delete.
func Delete(url string, opts ...RequestOptions) (*Response, error) {
	return defaultClient.Delete(url, opts...)
}
