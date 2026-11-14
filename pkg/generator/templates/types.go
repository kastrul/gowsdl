// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package templates

// TypesTemplate is the template for generating type definitions
const TypesTemplate = `
{{define "SimpleType"}}
	{{$type := replaceReservedWords .Name | makePublic}}
	{{if ne .List.ItemType ""}}
		type {{$type}} []{{toGoType .List.ItemType false}}
	{{else if ne .Union.MemberTypes ""}}
		type {{$type}} string
	{{else if .Restriction.Base}}
		{{if .Restriction.Enumeration}}
			type {{$type}} {{toGoType .Restriction.Base false}}

			{{$base := .Restriction.Base}}
			const (
				{{range .Restriction.Enumeration}}
					{{$enumValue := enumValue $base .Value}}
					{{if $enumValue}}
					{{$enumName := .Value | sanitizeEnumValue | replaceReservedWords | makePublic}}
					{{$type}}{{$enumName}} {{$type}} = {{$enumValue}}
					{{end}}
				{{end}}
			)
		{{else}}
			type {{$type}} {{toGoType .Restriction.Base false}}
		{{end}}
	{{end}}
{{end}}

{{/*
ElementFields emits one struct field per XSD element. An element with an inline
simple type gets the base of its restriction; an element with no type
information at all gets a string.
*/}}
{{define "ElementFields"}}
	{{range .}}
		{{$memberName := .Name | replaceReservedWords | makeFieldPublic}}
		{{if ne .Ref ""}}
			{{$memberName = .Ref | removeNamespacePrefix | replaceReservedWords | makeFieldPublic}}
			{{$typeName := .Ref | removeNamespacePrefix}}
			{{$memberType := toGoType $typeName .Nillable}}
			{{if eq .MaxOccurs "unbounded"}}
				{{$memberName}} []{{$memberType}} ` + "`" + `xml:"{{refXMLName .}},omitempty" json:"{{.Ref | removeNamespacePrefix}},omitempty"` + "`" + `
			{{else}}
				{{$memberName}} {{$memberType}} ` + "`" + `xml:"{{refXMLName .}},omitempty" json:"{{.Ref | removeNamespacePrefix}},omitempty"` + "`" + `
			{{end}}
		{{else if .ComplexType}}
			{{template "InlineComplexField" .}}
		{{else if ne .Type ""}}
			{{$typeName := .Type | removeNamespacePrefix}}
			{{$memberType := toGoType $typeName .Nillable}}
			{{if eq .MaxOccurs "unbounded"}}
				{{$memberName}} []{{$memberType}} ` + "`" + `xml:"{{.Name}},omitempty" json:"{{.Name}},omitempty"` + "`" + `
			{{else}}
				{{$memberName}} {{$memberType}} ` + "`" + `xml:"{{.Name}},omitempty" json:"{{.Name}},omitempty"` + "`" + `
			{{end}}
		{{else if .SimpleType}}
			{{template "InlineSimpleField" .}}
		{{else}}
			{{template "UntypedField" .}}
		{{end}}
	{{end}}
{{end}}

{{/*
UntypedField emits the field of an element with no type information at all: no
ref, no type attribute and no inline type.
*/}}
{{define "UntypedField"}}
	{{replaceReservedWords .Name | makeFieldPublic}} {{if eq .MaxOccurs "unbounded"}}[]{{end}}string ` + "`" + `xml:"{{.Name}},omitempty" json:"{{.Name}},omitempty"` + "`" + `
{{end}}

{{/*
InlineSimpleField emits the field of an element with an inline simple type: the
base of its restriction, or string when it has none (a list or a union).
*/}}
{{define "InlineSimpleField"}}
	{{$el := .}}
	{{$memberName := $el.Name | replaceReservedWords | makeFieldPublic}}
	{{$memberType := "string"}}
	{{if $el.SimpleType.Restriction.Base}}
		{{$memberType = toGoType ($el.SimpleType.Restriction.Base | removeNamespacePrefix) $el.Nillable}}
		{{if eq $el.MaxOccurs "unbounded"}}
			{{$memberType = printf "[]%s" $memberType}}
		{{end}}
	{{end}}
	{{$memberName}} {{$memberType}} ` + "`" + `xml:"{{$el.Name}},omitempty" json:"{{$el.Name}},omitempty"` + "`" + `
{{end}}

{{define "InlineComplexField"}}
	{{$el := .}}
	{{$memberName := $el.Name | replaceReservedWords | makeFieldPublic}}
	{{$inlineTypeName := inlineTypeName $el}}
	{{if eq $el.MaxOccurs "unbounded"}}
		{{$memberName}} []{{$inlineTypeName}} ` + "`" + `xml:"{{$el.Name}},omitempty" json:"{{$el.Name}},omitempty"` + "`" + `
	{{else if eq $el.MinOccurs "0"}}
		{{$memberName}} *{{$inlineTypeName}} ` + "`" + `xml:"{{$el.Name}},omitempty" json:"{{$el.Name}},omitempty"` + "`" + `
	{{else}}
		{{$memberName}} {{$inlineTypeName}} ` + "`" + `xml:"{{$el.Name}},omitempty" json:"{{$el.Name}},omitempty"` + "`" + `
	{{end}}
{{end}}

{{/*
AttributeFields emits one struct field per XSD attribute. The field of an
attribute called "Id" or "id" is named IdAttr so that it cannot clash with the
field of an element of that name.
*/}}
{{define "AttributeFields"}}
	{{range .}}
		{{$name := .Name}}
		{{if eq .Name "Id"}}
			{{$name = "IdAttr"}}
		{{end}}
		{{if eq .Name "id"}}
			{{$name = "idAttr"}}
		{{end}}
		{{replaceAttrReservedWords $name | makeFieldPublic}} {{toGoType .Type false}} ` + "`" + `xml:"{{attrXMLName .}},attr,omitempty" json:"{{.Name}},omitempty"` + "`" + `
	{{end}}
{{end}}

{{/*
TargetNamespaceMethod emits the TargetNamespace method of a message type. It is
called with a dict: Type is the Go type, Namespace the target namespace of its
declaration.
*/}}
{{define "TargetNamespaceMethod"}}
	// TargetNamespace returns the target namespace for SOAP envelope configuration
	func (t {{.Type}}) TargetNamespace() string {
		return "{{.Namespace}}"
	}
{{end}}

{{/*
ValueField emits the field holding the text of a simpleContent extension. It is
called with the extension's base type.
*/}}
{{define "ValueField"}}
	Value {{toGoType . false}} ` + "`" + `xml:",chardata" json:"value"` + "`" + `
{{end}}

{{/* ExtensionBase embeds the base type of a complexContent extension. */}}
{{define "ExtensionBase"}}
	{{if ne .Extension.Base ""}}
		{{if eq .Extension.Base "soap:Header"}}
			soap.Header
		{{else if eq (toGoType .Extension.Base false) "interface{}"}}
			InnerXML string ` + "`" + `xml:",innerxml"` + "`" + `
		{{else}}
			{{toGoType .Extension.Base false}}
		{{end}}
	{{end}}
{{end}}

{{/*
XMLNameField emits the XMLName field of a message element. It is called with a
dict: Name is the XML name, Namespace the target namespace of its declaration
and Qualified whether the schema declaring it has elementFormDefault="qualified".
*/}}
{{define "XMLNameField"}}
	{{if .Qualified}}
	XMLName xml.Name ` + "`" + `xml:"{{.Namespace}} {{.Name}}"` + "`" + `
	{{else if isResponseElement .Name}}
	{{/* Response elements use local name only to handle any namespace prefix */}}
	XMLName xml.Name ` + "`" + `xml:"{{.Name}}"` + "`" + `
	{{else}}
	XMLName xml.Name ` + "`" + `xml:"tns:{{.Name}}"` + "`" + `
	{{end}}
{{end}}

{{define "ComplexTypeInlineNamed"}}
	{{$complexType := .}}
	{{with $complexType.ComplexContent}}
		{{template "ExtensionBase" .}}
		{{template "ElementFields" .Extension.Sequence}}
		{{template "AttributeFields" .Extension.Attributes}}
	{{end}}

	{{template "ElementFields" $complexType.Sequence}}
	{{template "ElementFields" $complexType.Choice}}
	{{template "ElementFields" $complexType.SequenceChoice}}
	{{template "ElementFields" $complexType.All}}

	{{template "AttributeFields" $complexType.Attributes}}

	{{if ne (len $complexType.Any) 0}}
		Items []string ` + "`" + `xml:",any"` + "`" + `
	{{end}}
{{end}}

{{define "ComplexTypeInline"}}
	{{with .}}
	{{template "ExtensionBase" .ComplexContent}}

	{{template "ElementFields" .Sequence}}
	{{template "ElementFields" .Choice}}
	{{template "ElementFields" .SequenceChoice}}
	{{template "ElementFields" .All}}

	{{template "AttributeFields" .Attributes}}

	{{if ne (len .Any) 0}}
		Items []string ` + "`" + `xml:",any"` + "`" + `
	{{end}}
	{{end}}
{{end}}

{{define "ComplexType"}}
	{{$type := replaceReservedWords .Name | makePublic}}
	{{if .Abstract}}
		// {{$type}} is abstract
	{{end}}
	type {{$type}} struct {
		{{if isMessageElement .Name}}
			{{template "XMLNameField" (dict "Name" .Name "Namespace" (complexTypeNamespace .) "Qualified" (isQualified .))}}
		{{end}}
		{{template "ComplexTypeInline" .}}
		{{with .ComplexContent}}
			{{template "ElementFields" .Extension.Sequence}}
			{{template "AttributeFields" .Extension.Attributes}}
		{{end}}
		{{with .SimpleContent}}
			{{if ne .Extension.Base ""}}
				{{template "ValueField" .Extension.Base}}
			{{end}}
			{{template "AttributeFields" .Extension.Attributes}}
		{{end}}
	}
	
	{{if isMessageElement .Name}}
		{{if not (isQualified .)}}
		{{template "TargetNamespaceMethod" (dict "Type" $type "Namespace" (complexTypeNamespace .))}}
		{{end}}
	{{end}}
{{end}}

{{range .SimpleType}}
	{{template "SimpleType" .}}
{{end}}

{{range .ComplexTypes}}
	{{template "ComplexType" .}}
{{end}}

{{/* Generate named types for inline complex types */}}
{{range $key, $inline := .InlineTypes}}
	{{$type := $inline.GeneratedName}}
	type {{$type}} struct {
		{{if $inline.ComplexType.SimpleContent.Extension.Base}}
			{{/* Simple content with attributes */}}
			{{template "ValueField" $inline.ComplexType.SimpleContent.Extension.Base}}
			{{template "AttributeFields" $inline.ComplexType.SimpleContent.Extension.Attributes}}
		{{else}}
			{{template "ComplexTypeInlineNamed" $inline.ComplexType}}
		{{end}}
	}
{{end}}

{{range .Elements}}
	{{$type := replaceReservedWords .Name | makePublic}}
	{{if ne .Type ""}}
		{{$baseType := .Type | removeNamespacePrefix | replaceReservedWords}}
		{{$goType := toGoType .Type .Nillable | removePointerFromType}}
		{{$goPublicType := $baseType | makePublic}}
		{{/* Check if element name is different from the type name */}}
		{{$createAlias := true}}
		{{if eq .Name $baseType}}
			{{/* Don't create type alias if element name equals the type name */}}
			{{$createAlias = false}}
		{{end}}
		{{if $createAlias}}
			{{/* For complex types, use the public version of the type name */}}
			{{if isComplexType .Type}}
				type {{$type}} struct {
					{{template "XMLNameField" (dict "Name" .Name "Namespace" (elementNamespace .) "Qualified" (isQualified .))}}
					{{$goPublicType}}
				}
				
				{{if not (isQualified .)}}
				{{template "TargetNamespaceMethod" (dict "Type" $type "Namespace" (elementNamespace .))}}
				{{end}}
			{{else}}
				type {{$type}} {{$goType}}
			{{end}}
			{{if eq $goType "soap.XSDDateTime"}}
				func (xdt {{$type}}) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
					return soap.XSDDateTime(xdt).MarshalXML(e, start)
				}

				func (xdt *{{$type}}) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
					return (*soap.XSDDateTime)(xdt).UnmarshalXML(d, start)
				}
			{{else if eq $goType "soap.XSDDate"}}
				func (xd {{$type}}) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
					return soap.XSDDate(xd).MarshalXML(e, start)
				}

				func (xd *{{$type}}) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
					return (*soap.XSDDate)(xd).UnmarshalXML(d, start)
				}
			{{else if eq $goType "soap.XSDTime"}}
				func (xt {{$type}}) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
					return soap.XSDTime(xt).MarshalXML(e, start)
				}

				func (xt *{{$type}}) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
					return (*soap.XSDTime)(xt).UnmarshalXML(d, start)
				}
			{{end}}
		{{end}}
	{{else}}
		{{if .ComplexType}}
			{{/* ComplexTypeLocal handles global elements with nested complex types */}}
			{{$outerType := .Name | makePublic | replaceReservedWords}}
			{{if .ComplexType.Name}}
				{{$innerType := .ComplexType.Name | makePublic | replaceReservedWords}}
				type {{$outerType}} struct {
					{{$innerType}}
				}
				{{template "ComplexType" .ComplexType}}
			{{else}}
				type {{$outerType}} struct {
					{{template "XMLNameField" (dict "Name" .Name "Namespace" (elementNamespace .) "Qualified" (isQualified .))}}
					{{template "ComplexTypeInline" .ComplexType}}
					{{with .ComplexType.SimpleContent}}
						{{if ne .Extension.Base ""}}
							{{template "ValueField" .Extension.Base}}
						{{end}}
						{{template "AttributeFields" .Extension.Attributes}}
					{{end}}
				}

				{{if not (isQualified .)}}
				{{template "TargetNamespaceMethod" (dict "Type" $outerType "Namespace" (elementNamespace .))}}
				{{end}}
			{{end}}
		{{else}}
			{{if .SimpleType}}
				{{/* SimpleType local handles global elements with nested simple types */}}
				{{with .SimpleType}}
					{{if ne .Union.MemberTypes ""}}
						type {{$type}} string
					{{else if .Restriction.Base}}
						{{if .Restriction.Enumeration}}
							type {{$type}} {{toGoType .Restriction.Base false}}
							{{$base := .Restriction.Base}}
							const (
								{{range .Restriction.Enumeration}}
									{{$enumValue := enumValue $base .Value}}
									{{if $enumValue}}
									{{$enumName := .Value | sanitizeEnumValue | replaceReservedWords | makePublic}}
									{{$type}}{{$enumName}} {{$type}} = {{$enumValue}}
									{{end}}
								{{end}}
							)
						{{else}}
							type {{$type}} {{toGoType .Restriction.Base false}}
						{{end}}
					{{else}}
						type {{$type}} interface{}
					{{end}}
				{{end}}
			{{end}}
		{{end}}
	{{end}}
{{end}}

{{/* Generate wrapper types for RPC-style messages */}}
{{range .Messages}}
	{{if isEmptyMessage .}}
		{{$type := replaceReservedWords .Name | makePublic}}
		// {{$type}} is the empty {{.Name}} message. It marshals to no XML, so the SOAP body stays empty.
		type {{$type}} struct{}

		// MarshalXML implements xml.Marshaler and writes nothing.
		func ({{$type}}) MarshalXML(*xml.Encoder, xml.StartElement) error { return nil }
	{{else if and (isRPCStyleMessage .) (isHeaderMessage .)}}
		{{$type := replaceReservedWords .Name | makePublic}}
		// {{$type}} holds the parts of the {{.Name}} SOAP header message. Pass it to
		// soap.Client.AddHeader: every part set to a non-zero value is sent as its
		// own header element, and parts left zero are omitted.
		type {{$type}} struct {
			{{range .Parts}}
				{{$partName := .Name | replaceReservedWords | makeFieldPublic}}
				{{if ne .Type ""}}
					{{$partType := .Type | removeNamespacePrefix}}
					{{$partName}} {{toGoType $partType false}} ` + "`" + `xml:"{{.Name}},omitempty" json:"{{.Name}},omitempty"` + "`" + `
				{{else if ne .Element ""}}
					{{$elementType := .Element | removeNamespacePrefix}}
					{{$partName}} {{toGoType $elementType false}} ` + "`" + `xml:"{{partXMLName .}},omitempty" json:"{{.Name}},omitzero"` + "`" + `
				{{end}}
			{{end}}
		}

		// MarshalXML implements xml.Marshaler. It writes each non-zero part as a
		// sibling element, with no element for the message itself.
		func (h {{$type}}) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
			return soap.MarshalParts(e, h)
		}
	{{else if isRPCStyleMessage .}}
		{{$type := replaceReservedWords .Name | makePublic}}
		type {{$type}} struct {
			{{range .Parts}}
				{{$partName := .Name | replaceReservedWords | makeFieldPublic}}
				{{if ne .Type ""}}
					{{$partType := .Type | removeNamespacePrefix}}
					{{$partName}} {{toGoType $partType false}} ` + "`" + `xml:"{{.Name}}" json:"{{.Name}}"` + "`" + `
				{{else if ne .Element ""}}
					{{$elementType := .Element | removeNamespacePrefix}}
					{{$partName}} {{toGoType $elementType false}} ` + "`" + `xml:"{{.Name}}" json:"{{.Name}}"` + "`" + `
				{{end}}
			{{end}}
		}
	{{end}}
{{end}}

{{/* Generate RPC operation wrapper types */}}
{{range .PortTypes}}
	{{if isRPCPortType .Name}}
		{{$portType := .}}
		{{range .Operations}}
			{{$opName := .Name}}
			{{$inputMsgName := .Input.Message | removeNamespacePrefix}}
			{{$outputMsgName := .Output.Message | removeNamespacePrefix}}
			
			{{/* Find the input message and create operation wrapper */}}
			{{range $.Messages}}
				{{if eq .Name $inputMsgName}}
					// Operation wrapper for {{$opName | makePublic}}.
					type Operation{{$opName | makePublic}}In struct {
						XMLName xml.Name ` + "`" + `xml:"{{getTargetNamespace}} {{$opName}}"` + "`" + `
						
						{{range .Parts}}
							{{$partName := .Name | replaceReservedWords | makeFieldPublic}}
							{{if ne .Type ""}}
								{{$partType := .Type | removeNamespacePrefix}}
								{{$partName}} *{{toGoType $partType false | removePointerFromType}} ` + "`" + `xml:"{{.Name}},omitempty" json:"{{.Name}},omitempty"` + "`" + `
							{{else if ne .Element ""}}
								{{$elementType := .Element | removeNamespacePrefix}}
								{{$partName}} *{{toGoType $elementType false | removePointerFromType}} ` + "`" + `xml:"{{.Name}},omitempty" json:"{{.Name}},omitempty"` + "`" + `
							{{end}}
						{{end}}
					}
				{{end}}
			{{end}}
			
			{{/* Find the output message and create operation response wrapper */}}
			{{range $.Messages}}
				{{if eq .Name $outputMsgName}}
					// Operation wrapper for {{$opName | makePublic}}.
					type Operation{{$opName | makePublic}}Out struct {
						XMLName xml.Name ` + "`" + `xml:"{{getTargetNamespace}} {{$opName}}Response"` + "`" + `
						
						{{range .Parts}}
							{{$partName := .Name | replaceReservedWords | makeFieldPublic}}
							{{if ne .Type ""}}
								{{$partType := .Type | removeNamespacePrefix}}
								{{if eq .Name "return"}}
									Return *{{toGoType $partType false | removePointerFromType}} ` + "`" + `xml:"return,omitempty" json:"return,omitempty"` + "`" + `
								{{else}}
									{{$partName}} *{{toGoType $partType false | removePointerFromType}} ` + "`" + `xml:"{{.Name}},omitempty" json:"{{.Name}},omitempty"` + "`" + `
								{{end}}
							{{else if ne .Element ""}}
								{{$elementType := .Element | removeNamespacePrefix}}
								{{if eq .Name "return"}}
									Return *{{toGoType $elementType false | removePointerFromType}} ` + "`" + `xml:"return,omitempty" json:"return,omitempty"` + "`" + `
								{{else}}
									{{$partName}} *{{toGoType $elementType false | removePointerFromType}} ` + "`" + `xml:"{{.Name}},omitempty" json:"{{.Name}},omitempty"` + "`" + `
								{{end}}
							{{end}}
						{{end}}
					}
				{{end}}
			{{end}}
		{{end}}
	{{end}}
{{end}}

`
