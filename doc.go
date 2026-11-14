// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package gowsdl generates Go client and server code from WSDL 1.1 and
// WSDL 2.0 documents.
//
// Create a generator with NewGoWSDL and call Start to get the generated
// source, keyed by section ("header", "types", "operations", "server").
// The types and functions re-exported from the pkg/ subpackages are kept so
// code written against older versions of gowsdl keeps compiling.
package gowsdl
