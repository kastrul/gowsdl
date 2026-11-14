// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"strings"

	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/utils"
)

// WithTypePruning controls whether, when an operation filter is set via
// WithOperations, only the schema types reachable from the selected
// operations are generated. It is enabled by default and has no effect
// without an operation filter or when server code is generated.
func WithTypePruning(enabled bool) Option {
	return func(g *Generator) error {
		g.keepUnusedTypes = !enabled
		return nil
	}
}

// pruningActive reports whether generated types should be restricted to
// those reachable from the selected operations. Server code covers every
// operation, so it needs every type.
func (g *Generator) pruningActive() bool {
	return len(g.operations) > 0 && !g.keepUnusedTypes && !g.generateServer
}

// typeUsage is the set of schema declarations reachable from the selected
// operations. Names are local (prefix-less), matching how generated code
// refers to declarations.
type typeUsage struct {
	types        map[string]bool
	elements     map[string]bool
	attributes   map[string]bool
	messages     map[string]bool
	complexTypes map[*parser.XSDComplexType]bool

	schemaComplex    map[string][]*parser.XSDComplexType
	schemaSimple     map[string][]*parser.XSDSimpleType
	schemaElements   map[string][]*parser.XSDElement
	schemaAttributes map[string][]*parser.XSDAttribute
}

func newTypeUsage(schemas []*parser.XSDSchema) *typeUsage {
	u := &typeUsage{
		types:            make(map[string]bool),
		elements:         make(map[string]bool),
		attributes:       make(map[string]bool),
		messages:         make(map[string]bool),
		complexTypes:     make(map[*parser.XSDComplexType]bool),
		schemaComplex:    make(map[string][]*parser.XSDComplexType),
		schemaSimple:     make(map[string][]*parser.XSDSimpleType),
		schemaElements:   make(map[string][]*parser.XSDElement),
		schemaAttributes: make(map[string][]*parser.XSDAttribute),
	}
	for _, schema := range schemas {
		for _, ct := range schema.ComplexTypes {
			u.schemaComplex[ct.Name] = append(u.schemaComplex[ct.Name], ct)
		}
		for _, st := range schema.SimpleType {
			u.schemaSimple[st.Name] = append(u.schemaSimple[st.Name], st)
		}
		for _, el := range schema.Elements {
			u.schemaElements[el.Name] = append(u.schemaElements[el.Name], el)
		}
		for _, attr := range schema.Attributes {
			u.schemaAttributes[attr.Name] = append(u.schemaAttributes[attr.Name], attr)
		}
	}
	return u
}

// collectTypeUsage walks the selected operations and returns every schema
// declaration their client code depends on.
func (g *Generator) collectTypeUsage() *typeUsage {
	u := newTypeUsage(g.getSchemas())

	if g.wsdl != nil {
		messages := make(map[string]*parser.WSDLMessage, len(g.wsdl.Messages))
		for _, msg := range g.wsdl.Messages {
			messages[msg.Name] = msg
		}
		useMessage := func(ref string) {
			name := utils.LocalName(ref)
			if name == "" || u.messages[name] {
				return
			}
			u.messages[name] = true
			if msg, ok := messages[name]; ok {
				for _, part := range msg.Parts {
					u.useElement(part.Element)
					u.useType(part.Type)
				}
			}
		}
		useHeaders := func(headers []*parser.WSDLSOAPHeader) {
			for _, h := range headers {
				useMessage(h.Message)
				for _, hf := range h.HeadersFault {
					useMessage(hf.Message)
				}
			}
		}

		for _, pt := range g.filteredPortTypes() {
			selected := make(map[string]bool, len(pt.Operations))
			for _, op := range pt.Operations {
				selected[op.Name] = true
				useMessage(op.Input.Message)
				useMessage(op.Output.Message)
				for _, f := range op.Faults {
					useMessage(f.Message)
				}
			}
			for _, b := range g.bindingsForPortType(pt.Name) {
				for _, op := range b.Operations {
					if selected[op.Name] {
						useHeaders(op.Input.SOAPHeader)
						useHeaders(op.Output.SOAPHeader)
					}
				}
			}
		}
	}

	if g.wsdl2 != nil {
		for _, iface := range g.filteredInterfaces() {
			faults := make(map[string]string, len(iface.Faults))
			for _, f := range iface.Faults {
				faults[f.Name] = f.Element
			}
			for _, op := range iface.Operations {
				if op.Input != nil {
					u.useElement(op.Input.Element)
				}
				if op.Output != nil {
					u.useElement(op.Output.Element)
				}
				for _, f := range append(append([]*parser.WSDL2OperationFaultRef{}, op.InFault...), op.OutFault...) {
					u.useElement(faults[utils.LocalName(f.Ref)])
				}
			}
		}
	}

	return u
}

func (u *typeUsage) useType(ref string) {
	name := utils.LocalName(ref)
	if name == "" || u.types[name] {
		return
	}
	u.types[name] = true
	for _, ct := range u.schemaComplex[name] {
		u.walkComplexType(ct)
	}
	for _, st := range u.schemaSimple[name] {
		u.walkSimpleType(st)
	}
}

func (u *typeUsage) useElement(ref string) {
	name := utils.LocalName(ref)
	if name == "" || u.elements[name] {
		return
	}
	u.elements[name] = true
	for _, el := range u.schemaElements[name] {
		u.walkElement(el)
	}
}

func (u *typeUsage) useAttribute(ref string) {
	name := utils.LocalName(ref)
	if name == "" || u.attributes[name] {
		return
	}
	u.attributes[name] = true
	for _, attr := range u.schemaAttributes[name] {
		u.walkAttribute(attr)
	}
}

func (u *typeUsage) walkElement(el *parser.XSDElement) {
	if el == nil {
		return
	}
	u.useElement(el.Ref)
	u.useType(el.Type)
	u.walkComplexType(el.ComplexType)
	u.walkSimpleType(el.SimpleType)
	for _, grp := range el.Groups {
		for _, list := range [][]parser.XSDElement{grp.Sequence, grp.Choice, grp.All} {
			for i := range list {
				u.walkElement(&list[i])
			}
		}
	}
}

func (u *typeUsage) walkElements(elements []*parser.XSDElement) {
	for _, el := range elements {
		u.walkElement(el)
	}
}

func (u *typeUsage) walkComplexType(ct *parser.XSDComplexType) {
	if ct == nil || u.complexTypes[ct] {
		return
	}
	u.complexTypes[ct] = true

	u.walkElements(ct.Sequence)
	u.walkElements(ct.Choice)
	u.walkElements(ct.SequenceChoice)
	u.walkElements(ct.All)
	u.walkAttributes(ct.Attributes)
	u.walkExtension(&ct.ComplexContent.Extension)
	u.walkExtension(&ct.SimpleContent.Extension)
}

func (u *typeUsage) walkExtension(ext *parser.XSDExtension) {
	u.useType(ext.Base)
	u.walkAttributes(ext.Attributes)
	u.walkElements(ext.Sequence)
	u.walkElements(ext.Choice)
	u.walkElements(ext.SequenceChoice)
}

func (u *typeUsage) walkAttributes(attrs []*parser.XSDAttribute) {
	for _, attr := range attrs {
		u.walkAttribute(attr)
	}
}

func (u *typeUsage) walkAttribute(attr *parser.XSDAttribute) {
	if attr == nil {
		return
	}
	u.useAttribute(attr.Ref)
	u.useType(attr.Type)
	u.walkSimpleType(attr.SimpleType)
}

func (u *typeUsage) walkSimpleType(st *parser.XSDSimpleType) {
	if st == nil {
		return
	}
	u.useType(st.Restriction.Base)
	u.useType(st.List.ItemType)
	u.walkSimpleType(st.List.SimpleType)
	for _, member := range strings.Fields(st.Union.MemberTypes) {
		u.useType(member)
	}
	for _, member := range st.Union.SimpleType {
		u.walkSimpleType(member)
	}
}
