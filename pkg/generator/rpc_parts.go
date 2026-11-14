// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"strings"

	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/utils"
)

// bodyElementPart returns the part of msg declared with an element when msg
// is carried as that element alone, which is how findType resolves its Go
// type, or nil when msg is RPC-style or has no element part.
func (g *Generator) bodyElementPart(msg *parser.WSDLMessage) *parser.WSDLPart {
	if msg == nil || g.isRPCStyleMessage(msg) {
		return nil
	}
	for _, part := range msg.Parts {
		if part.Element != "" {
			return part
		}
	}
	return nil
}

// rpcElementParts places message parts in the operation wrappers of
// RPC-style port types. WSDL 1.1 section 3.5 puts each part inside an
// accessor element named after it, holding the part's element or, for a part
// declared with a type, the type's content.
type rpcElementParts struct {
	g                *Generator
	schemas          []*parser.XSDSchema
	decl             *declarationNamespaces
	responseElements map[string]bool
}

// path returns the xml struct tag path of an element part in an operation
// wrapper: the part accessor, then the element under the name its generated
// type declares in XMLName, which encoding/xml requires the tag to repeat.
// The namespace comes from that XMLName, so the tag names none.
func (r *rpcElementParts) path(part *parser.WSDLPart) string {
	local := utils.LocalName(part.Element)
	if el := r.element(part.Element); el != nil && r.usesTNSPrefix(el) {
		return part.Name + ">tns:" + local
	}
	return part.Name + ">" + local
}

// wrapperNamespace returns the namespace the "tns" prefix must be declared
// for when msg is sent in an operation wrapper, or "" when none of its
// element parts uses the prefix. The SOAP client declares it for a request
// with a TargetNamespace method.
func (r *rpcElementParts) wrapperNamespace(msg *parser.WSDLMessage) string {
	for _, part := range msg.Parts {
		if part.Element == "" {
			continue
		}
		if el := r.element(part.Element); el != nil && r.usesTNSPrefix(el) {
			if ct := r.namedComplexType(el); ct != nil && ct.Name == el.Name {
				return r.decl.complexType(ct)
			}
			return r.decl.element(el)
		}
	}
	return ""
}

// typePartNamespace returns the namespace of the content of a part declared
// with a type: the target namespace of the schema declaring its complex type
// when that schema is qualified, or "" for an unqualified schema or a simple
// type.
func (r *rpcElementParts) typePartNamespace(part *parser.WSDLPart) string {
	if part.Type == "" {
		return ""
	}
	name := utils.LocalName(part.Type)
	for _, schema := range r.schemas {
		for _, ct := range schema.ComplexTypes {
			if ct.Name == name && r.decl.isQualified(ct) {
				return r.decl.complexType(ct)
			}
		}
	}
	return ""
}

// elementPartNamespace returns the namespace of the global element of a part
// declared with an element.
func (r *rpcElementParts) elementPartNamespace(part *parser.WSDLPart) string {
	el := r.element(part.Element)
	if el == nil {
		return ""
	}
	if ct := r.namedComplexType(el); ct != nil && ct.Name == el.Name {
		return r.decl.complexType(ct)
	}
	return r.decl.element(el)
}

// usesTNSPrefix reports whether the type generated for the global element el
// names it "tns:<name>" in its XMLName field, mirroring the XMLNameField
// template: a struct with an XMLName, from a schema that is not qualified,
// for an element that is not a response.
func (r *rpcElementParts) usesTNSPrefix(el *parser.XSDElement) bool {
	var decl any = el
	switch {
	case el.Type != "":
		ct := r.namedComplexType(el)
		if ct == nil {
			return false
		}
		if ct.Name == el.Name {
			if !r.g.isMessageElement(ct.Name) {
				return false
			}
			decl = ct
		}
	case el.ComplexType != nil:
		if el.ComplexType.Name != "" {
			return false
		}
	default:
		return false
	}
	return !r.decl.isQualified(decl) && !r.responseElements[el.Name]
}

// namedComplexType returns the complex type el is declared with by its type
// attribute, or nil.
func (r *rpcElementParts) namedComplexType(el *parser.XSDElement) *parser.XSDComplexType {
	if el.Type == "" {
		return nil
	}
	name := utils.LocalName(el.Type)
	for _, schema := range r.schemas {
		for _, ct := range schema.ComplexTypes {
			if ct.Name == name {
				return ct
			}
		}
	}
	return nil
}

// element returns the global element ref names, matching its namespace when
// the prefix is declared on the WSDL definitions.
func (r *rpcElementParts) element(ref string) *parser.XSDElement {
	local := utils.LocalName(ref)
	ns := ""
	if prefix, _, ok := strings.Cut(ref, ":"); ok && r.g.wsdl != nil {
		ns = r.g.wsdl.Xmlns[prefix]
	}
	var fallback *parser.XSDElement
	for _, schema := range r.schemas {
		for _, el := range schema.Elements {
			if el.Name != local {
				continue
			}
			if ns == "" || schema.TargetNamespace == ns {
				return el
			}
			if fallback == nil {
				fallback = el
			}
		}
	}
	return fallback
}
