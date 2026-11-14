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
ElementFields emits one struct field per XSD element. It is called with a dict:
  Elements  the elements
  Untyped   what to emit for an element that has no ref, no type attribute and
            no inline complex or simple type. The call sites have always
            disagreed on this:
              "drop"    emit nothing
              "string"  emit a string field
              "name"    emit a field whose type is named after the element
An element with an inline simple type always gets the base of its restriction.
*/}}
{{define "ElementFields"}}
	{{$untyped := .Untyped}}
	{{range .Elements}}
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
			{{template "InlineSimpleField" (dict "Element" . "JSON" true)}}
		{{else if eq $untyped "string"}}
			{{$memberName}} string ` + "`" + `xml:"{{.Name}},omitempty" json:"{{.Name}},omitempty"` + "`" + `
		{{else if eq $untyped "name"}}
			{{$typeName := .Name}}
			{{$memberType := toGoType $typeName .Nillable}}
			{{if eq .MaxOccurs "unbounded"}}
				{{$memberName}} []{{$memberType}} ` + "`" + `xml:"{{.Name}},omitempty" json:"{{.Name}},omitempty"` + "`" + `
			{{else}}
				{{$memberName}} {{$memberType}} ` + "`" + `xml:"{{.Name}},omitempty" json:"{{.Name}},omitempty"` + "`" + `
			{{end}}
		{{end}}
	{{end}}
{{end}}

{{/*
InlineSimpleField emits the field of an element with an inline simple type: the
base of its restriction, or string when it has none (a list or a union). It is
called with a dict: Element is the element, JSON says whether to emit a json tag.
*/}}
{{define "InlineSimpleField"}}
	{{$el := .Element}}
	{{$json := .JSON}}
	{{$memberName := $el.Name | replaceReservedWords | makeFieldPublic}}
	{{$memberType := "string"}}
	{{if $el.SimpleType.Restriction.Base}}
		{{$memberType = toGoType ($el.SimpleType.Restriction.Base | removeNamespacePrefix) $el.Nillable}}
		{{if eq $el.MaxOccurs "unbounded"}}
			{{$memberType = printf "[]%s" $memberType}}
		{{end}}
	{{end}}
	{{$memberName}} {{$memberType}} ` + "`" + `xml:"{{$el.Name}},omitempty"{{if $json}} json:"{{$el.Name}},omitempty"{{end}}` + "`" + `
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
AttributeFields emits one struct field per XSD attribute. It is called with a dict:
  Attributes  the attributes
  RenameID    name the field of an attribute called "Id" or "id" IdAttr
  JSON        also emit a json tag
The call sites have always disagreed on both flags.
*/}}
{{define "AttributeFields"}}
	{{$renameID := .RenameID}}
	{{$json := .JSON}}
	{{range .Attributes}}
		{{$name := .Name}}
		{{if $renameID}}
			{{if eq .Name "Id"}}
				{{$name = "IdAttr"}}
			{{end}}
			{{if eq .Name "id"}}
				{{$name = "idAttr"}}
			{{end}}
		{{end}}
		{{replaceAttrReservedWords $name | makeFieldPublic}} {{toGoType .Type false}} ` + "`" + `xml:"{{.Name}},attr,omitempty"{{if $json}} json:"{{.Name}},omitempty"{{end}}` + "`" + `
	{{end}}
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
dict: Name is the XML name, Namespace the target namespace of its declaration.
*/}}
{{define "XMLNameField"}}
	{{if isElementFormQualified}}
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
		{{template "Elements" .Extension.Sequence}}
		{{template "AttributeFields" (dict "Attributes" .Extension.Attributes "RenameID" false "JSON" true)}}
	{{end}}

	{{template "ElementFields" (dict "Elements" $complexType.Sequence "Untyped" "string")}}
	{{template "ElementFields" (dict "Elements" $complexType.Choice "Untyped" "string")}}
	{{template "ElementFields" (dict "Elements" $complexType.SequenceChoice "Untyped" "string")}}
	{{template "ElementFields" (dict "Elements" $complexType.All "Untyped" "string")}}

	{{template "AttributeFields" (dict "Attributes" $complexType.Attributes "RenameID" true "JSON" true)}}

	{{if ne (len $complexType.Any) 0}}
		Items []string ` + "`" + `xml:",any"` + "`" + `
	{{end}}
{{end}}

{{define "ComplexTypeInline"}}
	{{with .}}
	{{template "ExtensionBase" .ComplexContent}}

	{{template "ElementFields" (dict "Elements" .Sequence "Untyped" "name")}}
	{{template "ElementFields" (dict "Elements" .Choice "Untyped" "drop")}}
	{{template "ElementFields" (dict "Elements" .SequenceChoice "Untyped" "drop")}}
	{{template "ElementFields" (dict "Elements" .All "Untyped" "drop")}}

	{{template "AttributeFields" (dict "Attributes" .Attributes "RenameID" true "JSON" true)}}

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
			{{template "XMLNameField" (dict "Name" .Name "Namespace" (complexTypeNamespace .))}}
		{{end}}
		{{template "ComplexTypeInline" .}}
		{{with .ComplexContent}}
			{{template "Elements" .Extension.Sequence}}
			{{template "AttributeFields" (dict "Attributes" .Extension.Attributes "RenameID" false "JSON" false)}}
		{{end}}
		{{with .SimpleContent}}
			{{if ne .Extension.Base ""}}
				Value {{toGoType .Extension.Base false}} ` + "`" + `xml:",chardata"` + "`" + `
			{{end}}
			{{template "AttributeFields" (dict "Attributes" .Extension.Attributes "RenameID" false "JSON" false)}}
		{{end}}
	}
	
	{{if isMessageElement .Name}}
		{{if not isElementFormQualified}}
		// TargetNamespace returns the target namespace for SOAP envelope configuration
		func (t {{$type}}) TargetNamespace() string {
			return "{{complexTypeNamespace .}}"
		}
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
			Value {{toGoType $inline.ComplexType.SimpleContent.Extension.Base false}} ` + "`" + `xml:",chardata" json:"value"` + "`" + `
			{{template "AttributeFields" (dict "Attributes" $inline.ComplexType.SimpleContent.Extension.Attributes "RenameID" false "JSON" true)}}
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
					{{template "XMLNameField" (dict "Name" .Name "Namespace" (elementNamespace .))}}
					{{$goPublicType}}
				}
				
				{{if not isElementFormQualified}}
				func (t {{$type}}) TargetNamespace() string {
					return "{{elementNamespace .}}"
				}
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
					{{template "XMLNameField" (dict "Name" .Name "Namespace" (elementNamespace .))}}
					{{template "ComplexTypeInline" .ComplexType}}
					{{with .ComplexType.SimpleContent}}
						{{if ne .Extension.Base ""}}
							Value {{toGoType .Extension.Base false}} ` + "`" + `xml:",chardata" json:"value"` + "`" + `
						{{end}}
						{{template "AttributeFields" (dict "Attributes" .Extension.Attributes "RenameID" false "JSON" true)}}
					{{end}}
				}

				{{if not isElementFormQualified}}
				func (t {{$outerType}}) TargetNamespace() string {
					return "{{elementNamespace .}}"
				}
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

{{define "Elements"}}
	{{range .}}
		{{if ne .Ref ""}}
			{{removeNamespacePrefix .Ref | replaceReservedWords | makeFieldPublic}} {{if eq .MaxOccurs "unbounded"}}[]{{end}}{{toGoType .Ref .Nillable}} ` + "`" + `xml:"{{refXMLName .}},omitempty"` + "`" + `
		{{else}}
			{{if not .Type}}
				{{if .ComplexType}}
					{{$inlineTypeName := inlineTypeName .}}
					{{replaceReservedWords .Name | makeFieldPublic}} {{if eq .MaxOccurs "unbounded"}}[]{{else if eq .MinOccurs "0"}}*{{end}}{{$inlineTypeName}} ` + "`" + `xml:"{{.Name}},omitempty"` + "`" + `
				{{else if .SimpleType}}
					{{template "InlineSimpleField" (dict "Element" . "JSON" false)}}
				{{end}}
			{{else}}
				{{replaceReservedWords .Name | makeFieldPublic}} {{if eq .MaxOccurs "unbounded"}}[]{{end}}{{toGoType .Type .Nillable}} ` + "`" + `xml:"{{.Name}},omitempty"` + "`" + `
			{{end}}
		{{end}}
	{{end}}
{{end}}

`
