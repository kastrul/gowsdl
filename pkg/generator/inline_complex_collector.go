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
// The result is keyed by GetInlineTypeKey.
func CollectInlineComplexTypes(schemas []*parser.XSDSchema) map[string]*InlineComplexType {
	return collectInlineTypes(schemas).byKey
}

// inlineTypes is the single source of truth for the names of anonymous
// complex types: every name is assigned once, by collectInlineTypes, and
// looked up by the complex type itself.
type inlineTypes struct {
	// byKey holds the collected types under their GetInlineTypeKey key. It
	// is what CollectInlineComplexTypes returns and what the types template
	// declares.
	byKey map[string]*InlineComplexType
	// keyOf maps each anonymous complex type to its key in byKey. When two
	// types are filed under the same key the later one replaces the earlier
	// in byKey, so it is the only one declared and names both.
	keyOf map[*parser.XSDComplexType]string
	used  map[string]bool
}

// collectInlineTypes names every anonymous complex type of the schemas.
func collectInlineTypes(schemas []*parser.XSDSchema) *inlineTypes {
	c := &inlineTypes{
		byKey: make(map[string]*InlineComplexType),
		keyOf: make(map[*parser.XSDComplexType]string),
		used:  declaredTypeNames(schemas),
	}

	for _, schema := range schemas {
		for _, ct := range schema.ComplexTypes {
			c.collectNested(ct.Name, ct)
		}

		for _, el := range schema.Elements {
			if el.ComplexType == nil {
				continue
			}
			typeName := utils.MakePublic(el.Name) + "Type"
			// The element itself is generated as a struct, so a colliding
			// alias would only duplicate an existing declaration.
			if !c.used[typeName] {
				c.used[typeName] = true
				c.add("", el, typeName, el.Name)
			}
			c.collectNested(el.Name, el.ComplexType)
		}
	}

	return c
}

// add records the anonymous complex type of el under parentName.
func (c *inlineTypes) add(parentName string, el *parser.XSDElement, generatedName, scope string) {
	key := GetInlineTypeKey(parentName, el.Name)
	c.byKey[key] = &InlineComplexType{
		Name:          el.Name,
		GeneratedName: generatedName,
		Scope:         scope,
		ComplexType:   el.ComplexType,
		MinOccurs:     el.MinOccurs,
		MaxOccurs:     el.MaxOccurs,
	}
	c.keyOf[el.ComplexType] = key
}

// collectNested recursively collects the inline complex types nested in ct,
// whose own Go name or scope is parentTypeName.
func (c *inlineTypes) collectNested(parentTypeName string, ct *parser.XSDComplexType) {
	collect := func(elements []*parser.XSDElement) {
		for _, el := range elements {
			if el.ComplexType == nil {
				continue
			}
			typeName := uniqueTypeName(utils.MakePublic(parentTypeName)+utils.MakePublic(el.Name)+"Type", c.used)
			c.add(parentTypeName, el, typeName, typeName)
			c.collectNested(typeName, el.ComplexType)
		}
	}

	collect(ct.Sequence)
	collect(ct.Choice)
	collect(ct.SequenceChoice)
	collect(ct.All)
	collect(ct.ComplexContent.Extension.Sequence)
}

// nameOf returns the Go type name assigned to the anonymous complex type of
// el. It reports an error for an element whose type was not collected,
// instead of inventing a name for a type that is never declared.
func (c *inlineTypes) nameOf(el *parser.XSDElement) (string, error) {
	if el == nil || el.ComplexType == nil {
		return "", fmt.Errorf("element has no inline complex type")
	}
	key, ok := c.keyOf[el.ComplexType]
	if !ok {
		return "", fmt.Errorf("no name was assigned to the inline complex type of element %q", el.Name)
	}
	return c.byKey[key].GeneratedName, nil
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

// GetInlineTypeKey returns the key CollectInlineComplexTypes files an inline
// type under: parentName is the name of the enclosing type, or "" for the
// anonymous type of a global element.
func GetInlineTypeKey(parentName, elementName string) string {
	if parentName == "" {
		return fmt.Sprintf("element_%s", elementName)
	}
	return fmt.Sprintf("%s_%s", parentName, elementName)
}

// FindInlineType finds the generated type name for an inline complex type by
// the names of its enclosing type and its element.
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
