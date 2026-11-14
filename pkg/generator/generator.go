// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/kastrul/gowsdl/pkg/generator/templates"
	"github.com/kastrul/gowsdl/pkg/http"
	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/types"
	"github.com/kastrul/gowsdl/pkg/utils"
)

// Generator handles WSDL code generation
type Generator struct {
	loc             *http.Location
	pkg             string
	wsdl            *parser.WSDL
	wsdl2           *parser.WSDL2
	wsdlVersion     string // "1.1" or "2.0"
	httpConfig      http.HTTPConfig
	useGenerics     bool
	genericsOnly    bool
	generateServer  bool
	operations      map[string]bool
	keepUnusedTypes bool
	typeMapper      *types.TypeMapper
	refNamespaces   map[*parser.XSDElement]string
}

// Option defines a function that configures Generator
type Option func(*Generator) error

// WithPackage sets the package name
func WithPackage(pkg string) Option {
	return func(g *Generator) error {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			pkg = "myservice"
		}
		g.pkg = pkg
		return nil
	}
}

// WithHTTPConfig sets the HTTP configuration. A nil config, including a nil
// *http.HTTPClientConfig, selects http.DefaultHTTPClientConfig.
func WithHTTPConfig(config http.HTTPConfig) Option {
	return func(g *Generator) error {
		if c, ok := config.(*http.HTTPClientConfig); config == nil || (ok && c == nil) {
			config = http.DefaultHTTPClientConfig()
		}
		g.httpConfig = config
		return nil
	}
}

// WithGenerics enables generic code generation
func WithGenerics() Option {
	return func(g *Generator) error {
		g.useGenerics = true
		return nil
	}
}

// WithGenericsOnly enables generic code generation and replaces the standard
// methods of WSDL 1.1 document-style operations with generic ones that keep the
// plain operation name and return soap.Result. Operations that cannot use the
// generic client keep their standard methods.
func WithGenericsOnly() Option {
	return func(g *Generator) error {
		g.useGenerics = true
		g.genericsOnly = true
		return nil
	}
}

// WithExportAllTypes is accepted for API compatibility. It currently has no
// effect on generated output.
func WithExportAllTypes(_ bool) Option {
	return func(_ *Generator) error {
		return nil
	}
}

// WithServerGeneration controls whether to generate server code
func WithServerGeneration(generate bool) Option {
	return func(g *Generator) error {
		g.generateServer = generate
		return nil
	}
}

// New creates a new Generator instance
func New(file string, opts ...Option) (*Generator, error) {
	loc, err := http.ParseLocation(file)
	if err != nil {
		return nil, err
	}

	g := &Generator{
		loc:        loc,
		pkg:        "myservice",
		httpConfig: http.DefaultHTTPClientConfig(),
		typeMapper: types.NewTypeMapper(),
	}

	for _, opt := range opts {
		if err := opt(g); err != nil {
			return nil, err
		}
	}

	return g, nil
}

// Generate initiates the code generation process
func (g *Generator) Generate(ctx context.Context) (map[string][]byte, error) {
	if err := g.unmarshal(ctx); err != nil {
		return nil, err
	}

	if err := g.validateOperationFilter(); err != nil {
		return nil, err
	}
	if g.wsdl != nil {
		httpOnly := g.httpOnlyPortTypes()
		var skipped []string
		for _, pt := range g.wsdl.PortTypes {
			if httpOnly[pt.Name] {
				skipped = append(skipped, pt.Name)
			}
		}
		if len(skipped) > 0 {
			log.Printf("[INFO] skipping port type(s) %s: bound only via HTTP GET/POST, not SOAP", strings.Join(skipped, ", "))
		}
	}
	if len(g.operations) > 0 && !g.keepUnusedTypes && g.generateServer {
		log.Printf("[WARN] type pruning is disabled because server code needs the types of every operation")
	}

	// Process WSDL nodes
	schemas := g.getSchemas()

	for _, schema := range schemas {
		parser.NewTraverser(schema, schemas).Traverse()
	}

	return g.generateCode(ctx)
}

func (g *Generator) generateCode(ctx context.Context) (map[string][]byte, error) {
	gocode := make(map[string][]byte)

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Calculate number of goroutines based on generateServer flag. The
	// header is generated afterwards because its imports depend on the
	// types and operations code.
	numGoroutines := 2 // types, operations
	if g.generateServer {
		numGoroutines = 5 // add server, server_header, server_wsdl
	}
	errChan := make(chan error, numGoroutines)

	// Generate all components in parallel
	wg.Add(numGoroutines)

	go func() {
		defer wg.Done()
		if data, err := g.genTypes(); err != nil {
			errChan <- err
		} else {
			mu.Lock()
			gocode["types"] = data
			mu.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		if data, err := g.genOperations(); err != nil {
			errChan <- err
		} else {
			mu.Lock()
			gocode["operations"] = data
			mu.Unlock()
		}
	}()

	// Only generate server files if generateServer is enabled
	if g.generateServer {
		go func() {
			defer wg.Done()
			if data, err := g.genServer(); err != nil {
				errChan <- err
			} else {
				mu.Lock()
				gocode["server"] = data
				mu.Unlock()
			}
		}()

		go func() {
			defer wg.Done()
			if data, err := g.genServerHeader(); err != nil {
				errChan <- err
			} else {
				mu.Lock()
				gocode["server_header"] = data
				mu.Unlock()
			}
		}()

		go func() {
			defer wg.Done()
			if data, err := g.genServerWSDL(ctx); err != nil {
				errChan <- err
			} else {
				mu.Lock()
				gocode["server_wsdl"] = data
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		return nil, err
	}

	header, err := g.genHeader(gocode["types"], gocode["operations"])
	if err != nil {
		return nil, err
	}
	gocode["header"] = header

	return gocode, nil
}

func (g *Generator) unmarshal(ctx context.Context) error {
	data, err := g.fetchFile(ctx, g.loc)
	if err != nil {
		return &types.WSDLError{
			Op:   "fetch",
			Path: g.loc.String(),
			Err:  err,
		}
	}

	// Detect WSDL version
	version, err := parser.DetectWSDLVersion(data)
	if err != nil {
		return &types.WSDLError{
			Op:   "detect_version",
			Path: g.loc.String(),
			Err:  err,
		}
	}
	g.wsdlVersion = version

	// Parse based on version
	if version == "2.0" {
		g.wsdl2 = new(parser.WSDL2)
		err = xml.Unmarshal(data, g.wsdl2)
		if err != nil {
			return &types.WSDLError{
				Op:   "parse",
				Path: g.loc.String(),
				Err:  fmt.Errorf("failed to unmarshal WSDL 2.0: %w", err),
			}
		}
	} else {
		g.wsdl = new(parser.WSDL)
		err = xml.Unmarshal(data, g.wsdl)
		if err != nil {
			return &types.WSDLError{
				Op:   "parse",
				Path: g.loc.String(),
				Err:  fmt.Errorf("failed to unmarshal WSDL 1.1: %w", err),
			}
		}
	}

	g.resolveSchemaExternals(ctx)
	ResolveAttributeRefs(g.getSchemas())
	g.refNamespaces = resolveElementRefNamespaces(g.getSchemas())

	return nil
}

// maxSchemaImportDepth bounds how deep nested xsd:import/xsd:include chains are followed.
const maxSchemaImportDepth = 10

// resolveSchemaExternals loads schemas referenced through xsd:import and
// xsd:include and appends them to the WSDL's schema list. Schemas that cannot
// be fetched or parsed are skipped with a warning.
func (g *Generator) resolveSchemaExternals(ctx context.Context) {
	resolved := make(map[string]bool)
	for _, schema := range g.getSchemas() {
		g.resolveXSDExternals(ctx, schema, g.loc, resolved, 0)
	}
}

func (g *Generator) resolveXSDExternals(ctx context.Context, schema *parser.XSDSchema, base *http.Location, resolved map[string]bool, depth int) {
	if depth >= maxSchemaImportDepth {
		log.Printf("[WARN] schema import depth limit (%d) reached at %s", maxSchemaImportDepth, base)
		return
	}

	refs := make([]string, 0, len(schema.Imports)+len(schema.Includes))
	for _, imp := range schema.Imports {
		if imp.SchemaLocation == "" {
			continue
		}
		refs = append(refs, imp.SchemaLocation)
	}
	for _, incl := range schema.Includes {
		if incl.SchemaLocation == "" {
			continue
		}
		refs = append(refs, incl.SchemaLocation)
	}

	for _, ref := range refs {
		loc, err := base.Parse(ref)
		if err != nil {
			log.Printf("[WARN] invalid schemaLocation %q: %v", ref, err)
			continue
		}
		key := loc.String()
		if resolved[key] {
			continue
		}
		resolved[key] = true

		data, err := g.fetchFile(ctx, loc)
		if err != nil {
			log.Printf("[WARN] failed to fetch schema %s: %v", key, err)
			continue
		}

		external := new(parser.XSDSchema)
		if err := xml.Unmarshal(data, external); err != nil {
			log.Printf("[WARN] failed to parse schema %s: %v", key, err)
			continue
		}

		g.resolveXSDExternals(ctx, external, loc, resolved, depth+1)
		g.appendSchema(external)
	}
}

func (g *Generator) appendSchema(schema *parser.XSDSchema) {
	if g.wsdlVersion == "2.0" && g.wsdl2 != nil {
		g.wsdl2.Types.Schemas = append(g.wsdl2.Types.Schemas, schema)
	} else if g.wsdl != nil {
		g.wsdl.Types.Schemas = append(g.wsdl.Types.Schemas, schema)
	}
}

func (g *Generator) fetchFile(ctx context.Context, loc *http.Location) ([]byte, error) {
	if loc.IsLocal() {
		return loc.ReadFile()
	}
	return http.DownloadFile(ctx, loc.URL(), g.httpConfig)
}

// Template generation methods
// genHeader renders the file header, importing context and soap only when
// the given generated code refers to them.
func (g *Generator) genHeader(code ...[]byte) ([]byte, error) {
	funcMap := g.createFuncMap()
	tmpl, err := template.New("header").Funcs(funcMap).Parse(templates.HeaderTemplate)
	if err != nil {
		return nil, err
	}

	used := referencedPackages(code...)
	data := templates.HeaderData{
		Package: g.pkg,
		Context: used["context"],
		SOAP:    used["soap"],
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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
					msgName := removeNamespacePrefix(operation.Output.Message)
					if msg, ok := messageMap[msgName]; ok {
						for _, part := range msg.Parts {
							if part.Element != "" {
								elementName := removeNamespacePrefix(part.Element)
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
					elementName := removeNamespacePrefix(operation.Output.Element)
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
				operationMessages[removeNamespacePrefix(op.Input.Message)] = true
				operationMessages[removeNamespacePrefix(op.Output.Message)] = true
			}
		}
	}
	funcMap["isEmptyMessage"] = func(msg *parser.WSDLMessage) bool {
		return msg != nil && len(msg.Parts) == 0 && operationMessages[msg.Name]
	}

	// Add function to check if a type is complex
	funcMap["isComplexType"] = func(typeName string) bool {
		// Remove namespace prefix if present
		if idx := strings.LastIndex(typeName, ":"); idx >= 0 {
			typeName = typeName[idx+1:]
		}
		return complexTypeMap[typeName]
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

	tmpl, err := template.New("types").Funcs(funcMap).Parse(templates.TypesTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

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

	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return fixStructOmitempty(dedupeStructFields(buf.Bytes())), nil
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
	funcMap := g.createFuncMap()

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

	tmpl, err := template.New("operations").Funcs(funcMap).Parse(tmplText)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (g *Generator) genServer() ([]byte, error) {
	funcMap := g.createFuncMap()

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

	tmpl, err := template.New("server").Funcs(funcMap).Parse(tmplText)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (g *Generator) genServerHeader() ([]byte, error) {
	funcMap := g.createFuncMap()
	tmpl, err := template.New("server_header").Funcs(funcMap).Parse(templates.ServerHeaderTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, g.pkg); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (g *Generator) genServerWSDL(ctx context.Context) ([]byte, error) {
	// Read the original WSDL file content
	wsdlContent, err := g.fetchFile(ctx, g.loc)
	if err != nil {
		return nil, fmt.Errorf("failed to read WSDL file: %w", err)
	}

	funcMap := g.createFuncMap()
	tmpl, err := template.New("server_wsdl").Funcs(funcMap).Parse(templates.ServerWSDLTemplate)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, string(wsdlContent)); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// createFuncMap creates the template function map
func (g *Generator) createFuncMap() template.FuncMap {
	return template.FuncMap{
		"toGoType":                 g.toGoType,
		"enumValue":                g.enumValue,
		"replaceReservedWords":     g.typeMapper.EscapeReservedWord,
		"replaceAttrReservedWords": g.typeMapper.EscapeReservedWord,
		"makePublic":               utils.MakePublic,
		"makeFieldPublic":          utils.MakeExportedField,
		"makePrivate":              makePrivate,
		"comment":                  comment,
		"removePointerFromType":    removePointerFromType,
		"removeNamespacePrefix":    removeNamespacePrefix,
		"removeNS":                 removeNamespacePrefix,
		"findType":                 g.findType,
		"findSOAPAction":           g.findSOAPAction,
		"findWSDL2SOAPAction":      g.findWSDL2SOAPAction,
		"makeValidXmlTag":          makeValidXmlTag,
		"goString":                 goString,
		"sanitizeEnumValue":        sanitizeEnumValue,
		"getTargetNamespace":       g.getTargetNamespace,
		"isRPCStyleMessage":        g.isRPCStyleMessage,
		"isRPCPortType":            g.isRPCPortType,
		"findNameByType":           g.findNameByType,
		"isMessageElement":         g.isMessageElement,
		"refXMLName":               g.refXMLName,
	}
}

// toGoType maps an XSD type reference to the Go type name used in generated
// code. User-defined types are named exactly as their declarations are.
func (g *Generator) toGoType(xsdType string, nillable bool) string {
	if xsdType == "" || g.typeMapper.IsBuiltinType(xsdType) {
		return g.typeMapper.MapXSDTypeToGoType(xsdType, nillable)
	}
	goType := utils.MakePublic(g.typeMapper.EscapeReservedWord(removeNamespacePrefix(xsdType)))
	if nillable {
		goType = "*" + goType
	}
	return goType
}

// resolveBuiltinBase follows simpleType restriction chains until an XSD
// built-in type is reached. It returns "" when the chain cannot be resolved.
func (g *Generator) resolveBuiltinBase(xsdType string) string {
	seen := make(map[string]bool)
	for xsdType != "" && !g.typeMapper.IsBuiltinType(xsdType) {
		local := removeNamespacePrefix(xsdType)
		if seen[local] {
			return ""
		}
		seen[local] = true
		next := ""
		for _, schema := range g.getSchemas() {
			for _, st := range schema.SimpleType {
				if st.Name == local {
					next = st.Restriction.Base
					break
				}
			}
			if next != "" {
				break
			}
		}
		xsdType = next
	}
	return xsdType
}

// enumValue returns the Go literal for an enumeration value of a simple type
// restricting base. It returns "" when the value cannot be represented.
func (g *Generator) enumValue(base, value string) string {
	goType := g.typeMapper.MapXSDTypeToGoType(g.resolveBuiltinBase(base), false)
	trimmed := strings.TrimSpace(value)
	switch goType {
	case "int8", "int16", "int32", "int64":
		if _, err := strconv.ParseInt(trimmed, 10, 64); err != nil {
			return ""
		}
		return trimmed
	case "uint8", "uint16", "uint32", "uint64":
		if _, err := strconv.ParseUint(trimmed, 10, 64); err != nil {
			return ""
		}
		return trimmed
	case "float32", "float64":
		if _, err := strconv.ParseFloat(trimmed, 64); err != nil {
			return ""
		}
		return trimmed
	case "bool":
		b, err := strconv.ParseBool(trimmed)
		if err != nil {
			return ""
		}
		return strconv.FormatBool(b)
	}
	return strconv.Quote(value)
}

// isRPCPortType checks if a port type is used in an RPC-style binding
func (g *Generator) isRPCPortType(portTypeName string) bool {
	if g.wsdl == nil {
		return false
	}

	// Check all bindings to see if any RPC binding references this port type
	for _, binding := range g.wsdl.Binding {
		if binding.SOAPBinding.Style == "rpc" {
			// Extract port type name from binding type
			bindingType := binding.Type
			if idx := strings.LastIndex(bindingType, ":"); idx != -1 {
				bindingType = bindingType[idx+1:]
			}
			if bindingType == portTypeName {
				return true
			}
		}
	}

	return false
}

// Helper functions
func makePrivate(identifier string) string {
	if identifier == "" {
		return ""
	}
	return strings.ToLower(identifier[:1]) + identifier[1:]
}

func comment(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = "// " + strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

func removePointerFromType(typ string) string {
	return strings.TrimPrefix(typ, "*")
}

func removeNamespacePrefix(name string) string {
	parts := strings.Split(name, ":")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return name
}

func makeValidXmlTag(xmlName, goName string) string {
	if xmlName == goName {
		return xmlName
	}
	return xmlName + ",omitempty"
}

func goString(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}

// sanitizeEnumValue converts a string to a valid Go identifier for enum constants
func sanitizeEnumValue(s string) string {
	// For backward compatibility, convert the way the original generator did
	if s == "" {
		return "EmptyString"
	}

	// Remove spaces entirely (don't convert to underscores)
	s = strings.ReplaceAll(s, " ", "")
	// Remove hyphens
	s = strings.ReplaceAll(s, "-", "")
	// Replace dots with nothing
	s = strings.ReplaceAll(s, ".", "")
	// Remove other special characters
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, "\\", "")
	s = strings.ReplaceAll(s, "(", "")
	s = strings.ReplaceAll(s, ")", "")
	s = strings.ReplaceAll(s, "[", "")
	s = strings.ReplaceAll(s, "]", "")
	s = strings.ReplaceAll(s, "{", "")
	s = strings.ReplaceAll(s, "}", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, ";", "")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "\"", "")

	// If the result is empty after cleanup, return "Empty"
	if s == "" {
		return "Empty"
	}

	return s
}

// isRPCStyleMessage determines if a message requires a wrapper type
// This is true for messages with multiple parts or single primitive-typed parts
func (g *Generator) isRPCStyleMessage(msg *parser.WSDLMessage) bool {
	if msg == nil {
		return false
	}

	// A single element part is a document/literal body; any remaining
	// type parts are MIME attachments, not body content.
	elementParts := 0
	for _, part := range msg.Parts {
		if part.Element != "" {
			elementParts++
		}
	}
	if elementParts == 1 {
		return false
	}

	// Multiple parts always require a wrapper
	if len(msg.Parts) > 1 {
		return true
	}

	// Single part with primitive type (no element reference) requires a wrapper
	if len(msg.Parts) == 1 && msg.Parts[0].Element == "" && msg.Parts[0].Type != "" {
		return true
	}

	return false
}

// findType finds the actual type name from a message reference
func (g *Generator) findType(message string) string {
	// Remove namespace prefix from message name
	message = removeNamespacePrefix(message)

	// For WSDL 1.1, messages typically reference elements
	// Try to find the message and get its part's element/type
	if g.wsdl != nil {
		for _, msg := range g.wsdl.Messages {
			if msg.Name == message {
				// A message without parts is declared as an empty type named after the message
				if len(msg.Parts) == 0 {
					return msg.Name
				}

				// Check if this is an RPC-style operation
				if g.isRPCStyleMessage(msg) {
					// For RPC-style operations, use the message name as the type
					// This will generate a wrapper type in the types template
					return msg.Name
				}

				if len(msg.Parts) > 0 {
					// Get the first part's element or type
					part := msg.Parts[0]
					if part.Element != "" {
						return removeNamespacePrefix(part.Element)
					}
					if part.Type != "" {
						// For document-style with a complex type reference
						return removeNamespacePrefix(part.Type)
					}
				}
			}
		}
	}

	// For WSDL 2.0, check interfaces
	if g.wsdl2 != nil {
		// In WSDL 2.0, operations directly reference elements
		return message
	}

	// Fallback: return the message name without namespace
	return message
}

func (g *Generator) findSOAPAction(operation, portType string) string {
	if g.wsdl == nil {
		return ""
	}

	// Find the binding that references the port type
	for _, binding := range g.wsdl.Binding {
		// Check if this binding is for the port type we're looking for
		bindingPortType := binding.Type
		if idx := strings.LastIndex(bindingPortType, ":"); idx != -1 {
			bindingPortType = bindingPortType[idx+1:]
		}

		// Normalize names by removing common suffixes to provide a more robust comparison
		normalizedBindingPortType := bindingPortType
		normalizedBindingPortType = strings.TrimSuffix(normalizedBindingPortType, "PortType")
		normalizedBindingPortType = strings.TrimSuffix(normalizedBindingPortType, "Port")
		normalizedBindingPortType = strings.TrimSuffix(normalizedBindingPortType, "Type")

		normalizedPortType := portType
		normalizedPortType = strings.TrimSuffix(normalizedPortType, "PortType")
		normalizedPortType = strings.TrimSuffix(normalizedPortType, "Port")
		normalizedPortType = strings.TrimSuffix(normalizedPortType, "Type")

		if strings.EqualFold(normalizedBindingPortType, normalizedPortType) {
			// Find the operation in this binding
			for _, op := range binding.Operations {
				if op.Name == operation {
					return op.SOAPOperation.SOAPAction
				}
			}
		}
	}

	return ""
}

// findWSDL2SOAPAction returns the SOAP action that a WSDL 2.0 binding of the
// named interface declares for the named operation, or "" if none does.
func (g *Generator) findWSDL2SOAPAction(operation, iface string) string {
	if g.wsdl2 == nil {
		return ""
	}
	for _, binding := range g.wsdl2.Bindings {
		if removeNamespacePrefix(binding.Interface) != iface {
			continue
		}
		for _, op := range binding.Operations {
			if removeNamespacePrefix(op.Ref) != operation {
				continue
			}
			if action := op.ResolvedSOAPAction(); action != "" {
				return action
			}
		}
	}
	return ""
}

func (g *Generator) getTargetNamespace() string {
	// Get the target namespace from the first schema that has one
	schemas := g.getSchemas()

	for _, schema := range schemas {
		if schema.TargetNamespace != "" {
			return schema.TargetNamespace
		}
	}
	return ""
}

// getSchemas returns the schemas based on the WSDL version
func (g *Generator) getSchemas() []*parser.XSDSchema {
	if g.wsdlVersion == "2.0" && g.wsdl2 != nil {
		return g.wsdl2.Types.Schemas
	} else if g.wsdl != nil {
		return g.wsdl.Types.Schemas
	}
	return nil
}

// findNameByType finds an XML element by its type name and returns the element's name
// This is used to determine if XMLName should be generated for complex types
func (g *Generator) findNameByType(typeName string) string {
	// Remove namespace prefix from type name
	typeName = removeNamespacePrefix(typeName)

	// Get schemas based on WSDL version
	schemas := g.getSchemas()

	// Search for an element that has this type
	for _, schema := range schemas {
		for _, elem := range schema.Elements {
			elemType := removeNamespacePrefix(elem.Type)
			// If element type matches, return the element name
			if elemType == typeName {
				return elem.Name
			}
		}
	}

	// If no matching element found, return the original type name
	return typeName
}

// isMessageElement checks if a complex type name corresponds to an element used in WSDL messages
func (g *Generator) isMessageElement(typeName string) bool {
	// Get messages based on WSDL version
	var messages []*parser.WSDLMessage
	if g.wsdlVersion == "2.0" && g.wsdl2 != nil {
		// WSDL 2.0 doesn't use messages the same way, return false for now
		return false
	} else if g.wsdl != nil {
		messages = g.wsdl.Messages
	}

	// Check if any message part references an element with this type name
	for _, message := range messages {
		for _, part := range message.Parts {
			if part.Element != "" {
				// Extract element name from part.Element (remove namespace prefix)
				elementName := part.Element
				if idx := strings.LastIndex(elementName, ":"); idx >= 0 {
					elementName = elementName[idx+1:]
				}
				// If element name matches the type name, this type is used as a message element
				if elementName == typeName {
					return true
				}
			}
		}
	}

	return false
}
