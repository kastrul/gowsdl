// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"text/template"

	"github.com/kastrul/gowsdl/pkg/generator/templates"
	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/utils"
)

// renderTemplate parses text as a template named name with funcMap and
// executes it with data. Errors are wrapped with the template name.
func renderTemplate(name, text string, funcMap template.FuncMap, data any) ([]byte, error) {
	tmpl, err := template.New(name).Funcs(funcMap).Parse(text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s template: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute %s template: %w", name, err)
	}

	return buf.Bytes(), nil
}

// Template generation methods
// genHeader renders the file header, importing context and soap only when
// the given generated code refers to them.
func (g *Generator) genHeader(code ...[]byte) ([]byte, error) {
	used := referencedPackages(code...)
	data := templates.HeaderData{
		Package: g.pkg,
		Context: used["context"],
		SOAP:    used["soap"],
	}

	return renderTemplate("header", templates.HeaderTemplate, g.createFuncMap(), data)
}

func (g *Generator) genTypes() ([]byte, error) {
	funcMap := g.createFuncMap()

	// Pre-process complex types into a map for efficient lookup
	complexTypeMap := make(map[string]bool)

	// Collect all complex types first
	schemas := g.getSchemas()

	for _, schema := range schemas {
		if schema.ComplexTypes != nil {
			for _, ct := range schema.ComplexTypes {
				complexTypeMap[ct.Name] = true
			}
		}
	}

	// Collect inline complex types
	inlineTypes := CollectInlineComplexTypes(schemas)

	// Collect response element names from operations
	responseElements := make(map[string]bool)

	// Handle WSDL 1.1
	if g.wsdl != nil && g.wsdl.PortTypes != nil {
		// Create a map of messages for efficient O(1) lookup
		messageMap := make(map[string]*parser.WSDLMessage, len(g.wsdl.Messages))
		for _, msg := range g.wsdl.Messages {
			messageMap[msg.Name] = msg
		}

		for _, portType := range g.wsdl.PortTypes {
			for _, operation := range portType.Operations {
				if operation.Output.Message != "" {
					// Find the message and its parts
					msgName := utils.LocalName(operation.Output.Message)
					if msg, ok := messageMap[msgName]; ok {
						for _, part := range msg.Parts {
							if part.Element != "" {
								elementName := utils.LocalName(part.Element)
								responseElements[elementName] = true
								// Debug log (commented out for production)
								// fmt.Printf("Added response element: %s from operation %s\n", elementName, operation.Name)
							}
						}
					}
				}
			}
		}
	}

	// Handle WSDL 2.0
	if g.wsdl2 != nil && g.wsdl2.Interfaces != nil {
		for _, iface := range g.wsdl2.Interfaces {
			for _, operation := range iface.Operations {
				if operation.Output != nil && operation.Output.Element != "" {
					elementName := utils.LocalName(operation.Output.Element)
					responseElements[elementName] = true
					// Debug log (commented out for production)
					// fmt.Printf("Added WSDL2 response element: %s from operation %s\n", elementName, operation.Name)
				}
			}
		}
	}

	// Messages without parts that an operation uses as input or output are
	// declared as empty types so the operation signatures compile.
	operationMessages := make(map[string]bool)
	if g.wsdl != nil {
		for _, pt := range g.filteredPortTypes() {
			for _, op := range pt.Operations {
				operationMessages[utils.LocalName(op.Input.Message)] = true
				operationMessages[utils.LocalName(op.Output.Message)] = true
			}
		}
	}
	funcMap["isEmptyMessage"] = func(msg *parser.WSDLMessage) bool {
		return msg != nil && len(msg.Parts) == 0 && operationMessages[msg.Name]
	}

	// Add function to check if a type is complex
	funcMap["isComplexType"] = func(typeName string) bool {
		return complexTypeMap[utils.LocalName(typeName)]
	}

	// Template State Management for Inline Type Name Resolution
	// ========================================================
	// The following code implements a state management pattern within the template
	// execution to handle nested inline complex types. This is necessary because
	// templates don't have access to the parent context when processing nested structures.
	//
	// How it works:
	// 1. When the template starts processing a complex type, it calls setCurrentType
	//    to store the type name in currentTypeName
	// 2. When processing nested inline types, the template uses "$" as a special
	//    marker for parentName to indicate "use the current context"
	// 3. getInlineTypeName checks if parentName is "$" and substitutes the stored
	//    currentTypeName value
	//
	// Example template usage:
	//   {{setCurrentType .Name}}  <!-- Store "OrderType" -->
	//   {{range .Sequence}}
	//     {{$inlineTypeName := getInlineTypeName "$" .Name}}  <!-- "$" becomes "OrderType" -->
	//   {{end}}
	//
	// WARNING: This pattern is fragile as it relies on proper sequencing of template
	// calls. Always ensure setCurrentType is called before processing nested elements.

	currentTypeName := ""

	// getInlineTypeName retrieves the generated name for an inline complex type
	// Parameters:
	//   - parentName: The parent type name, or "$" to use the current context type
	//   - elementName: The element name containing the inline type
	funcMap["getInlineTypeName"] = func(parentName, elementName string) string {
		// Special case: "$" means use the type currently being processed
		// This is a magic string used by templates to reference the current context
		if parentName == "$" {
			parentName = currentTypeName
		}
		if typeName, ok := FindInlineType(inlineTypes, parentName, elementName); ok {
			return typeName
		}
		// Fallback to generating a name if not found
		return utils.MakePublic(parentName) + utils.MakePublic(elementName) + "Type"
	}

	// setCurrentType stores the name of the type currently being processed by the template
	// This enables nested template calls to access the parent context via "$"
	// Returns empty string so it can be used inline in templates without output
	funcMap["setCurrentType"] = func(typeName string) string {
		currentTypeName = typeName
		return ""
	}

	// Add function to check if this should be an optional pointer
	funcMap["isOptionalType"] = IsOptionalType

	declNamespaces := resolveDeclarationNamespaces(schemas, g.getTargetNamespace())
	funcMap["elementNamespace"] = declNamespaces.element
	funcMap["complexTypeNamespace"] = declNamespaces.complexType

	// Add function to check if elements should be qualified
	funcMap["isElementFormQualified"] = func() bool {
		// Check if any schema has elementFormDefault="qualified"
		for _, schema := range schemas {
			if schema.ElementFormDefault == "qualified" {
				return true
			}
		}
		// Default is unqualified
		return false
	}

	// Add function to check if an element is a response element
	funcMap["isResponseElement"] = func(elementName string) bool {
		result := responseElements[elementName]
		// Debug log (commented out for production)
		// fmt.Printf("isResponseElement(%s) = %v (map contains %d elements)\n", elementName, result, len(responseElements))
		return result
	}

	// Add dict function for creating template data structures
	funcMap["dict"] = func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("dict must have even number of arguments")
		}
		dict := make(map[string]interface{})
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict keys must be strings")
			}
			dict[key] = values[i+1]
		}
		return dict, nil
	}

	// Merge all schemas into a single data structure
	data := struct {
		SimpleType   []*parser.XSDSimpleType
		ComplexTypes []*parser.XSDComplexType
		Elements     []*parser.XSDElement
		Schemas      []*parser.XSDSchema
		Messages     []*parser.WSDLMessage
		PortTypes    []*parser.WSDLPortType
		InlineTypes  map[string]*InlineComplexType
	}{}

	// Add RPC-style messages for WSDL 1.1
	if g.wsdl != nil {
		data.Messages = g.wsdl.Messages
		data.PortTypes = g.wsdl.PortTypes
	}

	data.Schemas = schemas
	data.InlineTypes = inlineTypes

	// Merge all schemas
	for _, schema := range schemas {
		if schema.SimpleType != nil {
			data.SimpleType = append(data.SimpleType, schema.SimpleType...)
		}
		if schema.ComplexTypes != nil {
			data.ComplexTypes = append(data.ComplexTypes, schema.ComplexTypes...)
		}
		if schema.Elements != nil {
			data.Elements = append(data.Elements, schema.Elements...)
		}
	}

	if g.pruningActive() {
		u := g.collectTypeUsage()
		data.SimpleType = filterSlice(data.SimpleType, func(st *parser.XSDSimpleType) bool { return u.types[st.Name] })
		data.ComplexTypes = filterSlice(data.ComplexTypes, func(ct *parser.XSDComplexType) bool { return u.types[ct.Name] })
		data.Elements = filterSlice(data.Elements, func(el *parser.XSDElement) bool { return u.elements[el.Name] })
		data.Messages = filterSlice(data.Messages, func(msg *parser.WSDLMessage) bool { return u.messages[msg.Name] })
		if g.wsdl != nil {
			data.PortTypes = g.filteredPortTypes()
		}
		used := make(map[string]*InlineComplexType, len(data.InlineTypes))
		for key, it := range data.InlineTypes {
			if u.complexTypes[it.ComplexType] {
				used[key] = it
			}
		}
		data.InlineTypes = used
	}

	code, err := renderTemplate("types", templates.TypesTemplate, funcMap, data)
	if err != nil {
		return nil, err
	}

	return fixStructOmitempty(breakRecursiveStructs(dedupeStructFields(code))), nil
}

// filterSlice returns the elements of items for which keep returns true.
func filterSlice[T any](items []T, keep func(T) bool) []T {
	var result []T
	for _, item := range items {
		if keep(item) {
			result = append(result, item)
		}
	}
	return result
}

func (g *Generator) genOperations() ([]byte, error) {
	// Choose template based on WSDL version
	var tmplText string
	var data interface{}

	if g.wsdlVersion == "2.0" && g.wsdl2 != nil {
		tmplText = templates.OperationsWSDL2Template
		data = g.filteredInterfaces()
		if g.useGenerics {
			log.Printf("gowsdl: generic client generation is not supported for WSDL 2.0; generating standard operations only")
		}
	} else if g.wsdl != nil {
		tmplText = templates.OperationsWSDL1Template
		// Wrap data to include both PortTypes and Messages for RPC operations
		data = struct {
			PortTypes    []*parser.WSDLPortType
			Messages     []*parser.WSDLMessage
			UseGenerics  bool
			GenericsOnly bool
		}{
			PortTypes:    g.filteredPortTypes(),
			Messages:     g.wsdl.Messages,
			UseGenerics:  g.useGenerics,
			GenericsOnly: g.genericsOnly,
		}
	} else {
		return []byte{}, nil
	}

	return renderTemplate("operations", tmplText, g.createFuncMap(), data)
}

func (g *Generator) genServer() ([]byte, error) {
	// Choose template based on WSDL version
	var tmplText string
	var data interface{}

	if g.wsdlVersion == "2.0" && g.wsdl2 != nil {
		tmplText = templates.ServerWSDL2Template
		data = g.wsdl2.Interfaces
	} else if g.wsdl != nil {
		tmplText = templates.ServerTemplate
		data = g.soapPortTypes()
	} else {
		return nil, errors.New("no WSDL data available")
	}

	return renderTemplate("server", tmplText, g.createFuncMap(), data)
}

func (g *Generator) genServerHeader() ([]byte, error) {
	return renderTemplate("server_header", templates.ServerHeaderTemplate, g.createFuncMap(), g.pkg)
}

func (g *Generator) genServerWSDL(ctx context.Context) ([]byte, error) {
	// Read the original WSDL file content
	wsdlContent, err := g.fetchFile(ctx, g.loc)
	if err != nil {
		return nil, fmt.Errorf("failed to read WSDL file: %w", err)
	}

	return renderTemplate("server_wsdl", templates.ServerWSDLTemplate, g.createFuncMap(), string(wsdlContent))
}
