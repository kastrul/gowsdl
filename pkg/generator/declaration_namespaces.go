// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"github.com/kastrul/gowsdl/pkg/parser"
)

// resolveDeclarationNamespaces maps each global element and named complex type
// to the target namespace of the schema that declares it. Global elements are
// always qualified with their schema's target namespace, so this is the
// namespace their generated XMLName must carry. Schemas without a target
// namespace are skipped.
func resolveDeclarationNamespaces(schemas []*parser.XSDSchema) map[any]string {
	result := make(map[any]string)
	for _, schema := range schemas {
		if schema.TargetNamespace == "" {
			continue
		}
		for _, el := range schema.Elements {
			result[el] = schema.TargetNamespace
			if el.ComplexType != nil {
				result[el.ComplexType] = schema.TargetNamespace
			}
		}
		for _, ct := range schema.ComplexTypes {
			result[ct] = schema.TargetNamespace
		}
	}
	return result
}

// namespaceOf returns the target namespace of the schema declaring decl, a
// global *parser.XSDElement or *parser.XSDComplexType. When the declaring
// schema is unknown it falls back to the first schema's target namespace.
func (g *Generator) namespaceOf(decl any) string {
	if ns := g.declNamespaces[decl]; ns != "" {
		return ns
	}
	return g.getTargetNamespace()
}
