// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"strings"

	"github.com/kastrul/gowsdl/pkg/parser"
)

// resolveElementRefNamespaces maps each element declared with ref to the
// namespace URI of the referenced element, when that namespace differs from the
// target namespace of the schema containing the reference. Prefixed refs are
// resolved through the schema's xmlns declarations and unprefixed refs through
// its default namespace. Refs that cannot be resolved are left out, so their
// tags fall back to the local name and match elements in any namespace when
// unmarshaling.
func resolveElementRefNamespaces(schemas []*parser.XSDSchema) map[*parser.XSDElement]string {
	result := make(map[*parser.XSDElement]string)

	for _, schema := range schemas {
		visited := make(map[*parser.XSDComplexType]bool)

		var walkComplexType func(ct *parser.XSDComplexType)
		walkElements := func(elements []*parser.XSDElement) {
			for _, el := range elements {
				if el.Ref != "" {
					if ns := refNamespaceURI(schema, el.Ref); ns != "" && ns != schema.TargetNamespace {
						result[el] = ns
					}
				}
				if el.ComplexType != nil {
					walkComplexType(el.ComplexType)
				}
			}
		}

		walkComplexType = func(ct *parser.XSDComplexType) {
			if ct == nil || visited[ct] {
				return
			}
			visited[ct] = true
			walkElements(ct.Sequence)
			walkElements(ct.Choice)
			walkElements(ct.SequenceChoice)
			walkElements(ct.All)
			walkElements(ct.ComplexContent.Extension.Sequence)
			walkElements(ct.ComplexContent.Extension.Choice)
			walkElements(ct.ComplexContent.Extension.SequenceChoice)
		}

		for _, ct := range schema.ComplexTypes {
			walkComplexType(ct)
		}
		walkElements(schema.Elements)
	}

	return result
}

func refNamespaceURI(schema *parser.XSDSchema, ref string) string {
	prefix, _, ok := strings.Cut(ref, ":")
	if !ok {
		prefix = ""
	}
	return schema.Xmlns[prefix]
}

// refXMLName returns the name to use in the xml struct tag of an element
// declared with ref: the local part of the ref, preceded by the referenced
// element's namespace URI when it lives in a different namespace than the
// schema that references it.
func (g *Generator) refXMLName(el *parser.XSDElement) string {
	local := removeNamespacePrefix(el.Ref)
	if ns := g.refNamespaces[el]; ns != "" {
		return ns + " " + local
	}
	return local
}
