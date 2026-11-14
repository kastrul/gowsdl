// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"strings"

	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/utils"
)

// toGoType maps an XSD type reference to the Go type name used in generated
// code. User-defined types are named exactly as their declarations are.
func (g *Generator) toGoType(xsdType string, nillable bool) string {
	if xsdType == "" || g.typeMapper.IsBuiltinType(xsdType) {
		return g.typeMapper.MapXSDTypeToGoType(xsdType, nillable)
	}
	goType := utils.MakePublic(g.typeMapper.EscapeReservedWord(utils.LocalName(xsdType)))
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
		local := utils.LocalName(xsdType)
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

// isRPCPortType checks if a port type is used in an RPC-style binding
func (g *Generator) isRPCPortType(portTypeName string) bool {
	if g.wsdl == nil {
		return false
	}

	// Check all bindings to see if any RPC binding references this port type
	for _, binding := range g.wsdl.Binding {
		if binding.SOAPBinding.Style == "rpc" {
			// Extract port type name from binding type
			bindingType := utils.LocalName(binding.Type)
			if bindingType == portTypeName {
				return true
			}
		}
	}

	return false
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
	message = utils.LocalName(message)

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
						return utils.LocalName(part.Element)
					}
					if part.Type != "" {
						// For document-style with a complex type reference
						return utils.LocalName(part.Type)
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
		bindingPortType := utils.LocalName(binding.Type)

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
		if utils.LocalName(binding.Interface) != iface {
			continue
		}
		for _, op := range binding.Operations {
			if utils.LocalName(op.Ref) != operation {
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
	typeName = utils.LocalName(typeName)

	// Get schemas based on WSDL version
	schemas := g.getSchemas()

	// Search for an element that has this type
	for _, schema := range schemas {
		for _, elem := range schema.Elements {
			elemType := utils.LocalName(elem.Type)
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
				elementName := utils.LocalName(part.Element)
				// If element name matches the type name, this type is used as a message element
				if elementName == typeName {
					return true
				}
			}
		}
	}

	return false
}
