// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"strconv"
	"strings"
	"text/template"

	"github.com/kastrul/gowsdl/pkg/utils"
)

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
		"removeNamespacePrefix":    utils.LocalName,
		"removeNS":                 utils.LocalName,
		"findType":                 g.findType,
		"findSOAPAction":           g.findSOAPAction,
		"findWSDL2SOAPAction":      g.findWSDL2SOAPAction,
		"makeValidXmlTag":          makeValidXMLTag,
		"goString":                 goString,
		"sanitizeEnumValue":        sanitizeEnumValue,
		"getTargetNamespace":       g.getTargetNamespace,
		"isRPCStyleMessage":        g.isRPCStyleMessage,
		"isRPCPortType":            g.isRPCPortType,
		"findNameByType":           g.findNameByType,
		"isMessageElement":         g.isMessageElement,
		"refXMLName":               g.refXMLName,
		"attrXMLName":              g.attrXMLName,
		"isHeaderMessage":          g.isHeaderMessage,
		"partXMLName":              g.partXMLName,
	}
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

func makeValidXMLTag(xmlName, goName string) string {
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
