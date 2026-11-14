// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package http fetches WSDL and XSD documents for gowsdl.
//
// It resolves locations relative to the referring document, downloads remote
// documents with a configurable client (timeouts, retries, TLS, proxy) and
// can rate-limit requests.
package http
