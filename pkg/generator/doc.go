// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package generator turns a parsed WSDL document into Go source code.
//
// A Generator is created with New and configured with Option values such as
// WithPackage, WithOperations and WithGenerics. Generate returns the
// formatted source, keyed by section.
package generator
