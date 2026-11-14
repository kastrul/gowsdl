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
	inline := collectInlineTypes(schemas)

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

	// inlineTypeName returns the name assigned to the anonymous complex type
	// of an element. Names are resolved once, by collectInlineTypes.
	funcMap["inlineTypeName"] = inline.nameOf

	// Add function to check if this should be an optional pointer
	funcMap["isOptionalType"] = IsOptionalType

	declNamespaces := resolveDeclarationNamespaces(schemas, g.getTargetNamespace())
	funcMap["elementNamespace"] = declNamespaces.element
	funcMap["complexTypeNamespace"] = declNamespaces.complexType

	funcMap["isQualified"] = declNamespaces.isQualified

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
	data.InlineTypes = inline.byKey

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
	var data templates.ServerData

	if g.wsdlVersion == "2.0" && g.wsdl2 != nil {
		tmplText = templates.ServerWSDL2Template
		data = templates.ServerData{
			Operations:              g.serverOperationsWSDL2(g.wsdl2.Interfaces),
			AcceptSOAP12ContentType: true,
			WSDL2Layout:             true,
		}
	} else if g.wsdl != nil {
		tmplText = templates.ServerTemplate
		data = templates.ServerData{Operations: g.serverOperationsWSDL1(g.soapPortTypes())}
	} else {
		return nil, errors.New("no WSDL data available")
	}

	return renderTemplate("server", tmplText, g.createFuncMap(), data)
}

// serverTypeName returns the Go type generated server code uses for the
// message or element ref names.
func (g *Generator) serverTypeName(ref string) string {
	return utils.MakePublic(g.typeMapper.EscapeReservedWord(g.findType(ref)))
}

// serverOperationsWSDL1 resolves the request and response types of every
// operation of the given WSDL 1.1 port types from their messages.
func (g *Generator) serverOperationsWSDL1(portTypes []*parser.WSDLPortType) []templates.ServerOperation {
	var ops []templates.ServerOperation
	for _, pt := range portTypes {
		for _, op := range pt.Operations {
			ops = append(ops, templates.ServerOperation{
				Request:     g.serverTypeName(op.Input.Message),
				Response:    g.serverTypeName(op.Output.Message),
				HasRequest:  true,
				HasResponse: true,
			})
		}
	}
	return ops
}

// serverOperationsWSDL2 resolves the request and response types of every
// operation of the given WSDL 2.0 interfaces from their elements. An
// operation may lack an input or an output.
func (g *Generator) serverOperationsWSDL2(interfaces []*parser.WSDL2Interface) []templates.ServerOperation {
	var ops []templates.ServerOperation
	for _, iface := range interfaces {
		for _, op := range iface.Operations {
			var so templates.ServerOperation
			if op.Input != nil {
				so.Request, so.HasRequest = g.serverTypeName(op.Input.Element), true
			}
			if op.Output != nil {
				so.Response, so.HasResponse = g.serverTypeName(op.Output.Element), true
			}
			ops = append(ops, so)
		}
	}
	return ops
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
