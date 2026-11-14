// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"fmt"

	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/types"
	"github.com/kastrul/gowsdl/pkg/utils"
)

// InlineComplexType represents an inline complex type that needs to be extracted
type InlineComplexType struct {
	Name          string
	GeneratedName string
	// Scope is the parent name under which the inline types nested in this one were collected.
	Scope       string
	ComplexType *parser.XSDComplexType
	MinOccurs   string
	MaxOccurs   string
}

// CollectInlineComplexTypes finds all inline complex types in the schema and generates names for them.
// Generated names never collide with top-level schema types or with each other.
func CollectInlineComplexTypes(schemas []*parser.XSDSchema) map[string]*InlineComplexType {
	inlineTypes := make(map[string]*InlineComplexType)
	used := declaredTypeNames(schemas)

	for _, schema := range schemas {
		for _, ct := range schema.ComplexTypes {
			collectFromComplexType(ct.Name, ct, inlineTypes, used)
		}

		for _, el := range schema.Elements {
			if el.ComplexType == nil {
				continue
			}
			typeName := utils.MakePublic(el.Name) + "Type"
			// The element itself is generated as a struct, so a colliding
			// alias would only duplicate an existing declaration.
			if !used[typeName] {
				used[typeName] = true
				key := fmt.Sprintf("element_%s", el.Name)
				inlineTypes[key] = &InlineComplexType{
					Name:          el.Name,
					GeneratedName: typeName,
					Scope:         el.Name,
					ComplexType:   el.ComplexType,
					MinOccurs:     el.MinOccurs,
					MaxOccurs:     el.MaxOccurs,
				}
			}
			collectFromComplexType(el.Name, el.ComplexType, inlineTypes, used)
		}
	}

	return inlineTypes
}

// declaredTypeNames returns the Go names of all top-level types declared by the schemas.
func declaredTypeNames(schemas []*parser.XSDSchema) map[string]bool {
	mapper := types.NewTypeMapper()
	goName := func(name string) string {
		return utils.MakePublic(mapper.EscapeReservedWord(name))
	}

	used := make(map[string]bool)
	for _, schema := range schemas {
		for _, st := range schema.SimpleType {
			used[goName(st.Name)] = true
		}
		for _, ct := range schema.ComplexTypes {
			used[goName(ct.Name)] = true
		}
		for _, el := range schema.Elements {
			used[goName(el.Name)] = true
		}
	}
	return used
}

// uniqueTypeName returns name, or name with a numeric suffix if it is already used,
// and marks the result as used.
func uniqueTypeName(name string, used map[string]bool) string {
	candidate := name
	for i := 2; used[candidate]; i++ {
		candidate = fmt.Sprintf("%s%d", name, i)
	}
	used[candidate] = true
	return candidate
}

// collectFromComplexType recursively collects inline complex types from a complex type
func collectFromComplexType(parentTypeName string, ct *parser.XSDComplexType, inlineTypes map[string]*InlineComplexType, used map[string]bool) {
	collect := func(elements []*parser.XSDElement) {
		for _, el := range elements {
			if el.ComplexType == nil {
				continue
			}
			typeName := uniqueTypeName(utils.MakePublic(parentTypeName)+utils.MakePublic(el.Name)+"Type", used)
			key := fmt.Sprintf("%s_%s", parentTypeName, el.Name)

			inlineTypes[key] = &InlineComplexType{
				Name:          el.Name,
				GeneratedName: typeName,
				Scope:         typeName,
				ComplexType:   el.ComplexType,
				MinOccurs:     el.MinOccurs,
				MaxOccurs:     el.MaxOccurs,
			}

			collectFromComplexType(typeName, el.ComplexType, inlineTypes, used)
		}
	}

	collect(ct.Sequence)
	collect(ct.Choice)
	collect(ct.All)
	if ct.ComplexContent.Extension.Base != "" {
		collect(ct.ComplexContent.Extension.Sequence)
	}
}

// GetInlineTypeKey generates a unique key for an inline type based on its context
func GetInlineTypeKey(parentName, elementName string) string {
	if parentName == "" {
		return fmt.Sprintf("element_%s", elementName)
	}
	return fmt.Sprintf("%s_%s", parentName, elementName)
}

// FindInlineType finds the generated type name for an inline complex type
func FindInlineType(inlineTypes map[string]*InlineComplexType, parentName, elementName string) (string, bool) {
	key := GetInlineTypeKey(parentName, elementName)
	if it, ok := inlineTypes[key]; ok {
		return it.GeneratedName, true
	}
	return "", false
}

// IsOptionalType checks if the type should use a pointer
func IsOptionalType(minOccurs string, maxOccurs string) bool {
	return minOccurs == "0" && maxOccurs != "unbounded"
}
