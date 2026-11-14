// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package gowsdl

import (
	"github.com/kastrul/gowsdl/pkg/core"
	"github.com/kastrul/gowsdl/pkg/http"
	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/types"
)

// Types re-exported from the pkg/ subpackages for backward compatibility.
type (
	// WSDLError is an alias of types.WSDLError.
	WSDLError = types.WSDLError
	// SchemaError is an alias of types.SchemaError.
	SchemaError = types.SchemaError
	// WSDL is an alias of parser.WSDL.
	WSDL = parser.WSDL
	// WSDL2 is an alias of parser.WSDL2.
	WSDL2 = parser.WSDL2
	// XSDSchema is an alias of parser.XSDSchema.
	XSDSchema = parser.XSDSchema
	// XSDComplexType is an alias of parser.XSDComplexType.
	XSDComplexType = parser.XSDComplexType
	// XSDElement is an alias of parser.XSDElement.
	XSDElement = parser.XSDElement
	// XSDAttribute is an alias of parser.XSDAttribute.
	XSDAttribute = parser.XSDAttribute
	// WSDLBinding is an alias of parser.WSDLBinding.
	WSDLBinding = parser.WSDLBinding
	// WSDLOperation is an alias of parser.WSDLOperation.
	WSDLOperation = parser.WSDLOperation
	// WSDLService is an alias of parser.WSDLService.
	WSDLService = parser.WSDLService
	// WSDLPort is an alias of parser.WSDLPort.
	WSDLPort = parser.WSDLPort
	// WSDLImport is an alias of parser.WSDLImport.
	WSDLImport = parser.WSDLImport
	// XSDImport is an alias of parser.XSDImport.
	XSDImport = parser.XSDImport
	// NamespaceManager is an alias of core.NamespaceManager.
	NamespaceManager = core.NamespaceManager
	// Location is an alias of http.Location.
	Location = http.Location
	// HTTPClientConfig is an alias of http.HTTPClientConfig.
	HTTPClientConfig = http.HTTPClientConfig
)

// Functions re-exported from the pkg/ subpackages for backward compatibility.
var (
	NewNamespaceManager     = core.NewNamespaceManager
	ParseLocation           = http.ParseLocation
	DefaultHTTPClientConfig = http.DefaultHTTPClientConfig
)
