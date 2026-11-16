// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package parser

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// XSDCompositor is one xs:sequence, xs:choice or xs:all enclosing an element
// declaration or group reference within its complex type or group
// definition.
type XSDCompositor struct {
	// Kind is "sequence", "choice" or "all".
	Kind string
	// ID tells the compositor apart from the others of the same complex
	// type or group definition: it is its 1-based document order among
	// them.
	ID int
	// MinOccurs and MaxOccurs are the compositor's own attributes, "" when
	// absent.
	MinOccurs string
	MaxOccurs string
	// Group is the name of the group definition whose top-level model group
	// the compositor is, and "" for any other.
	Group string
}

// XSDCompositors lists the compositors enclosing an element declaration or
// group reference, outermost first. It is read from and written to an
// attribute the parser adds to the schema tokens.
type XSDCompositors []XSDCompositor

// compositorsAttr is the attribute markOptionalGroupMembers adds to element
// declarations and group references, read into their Compositors.
var compositorsAttr = xml.Name{Space: "urn:gowsdl:parser", Local: "compositors"}

// String returns the attribute form of c: one kind:id:minOccurs:maxOccurs
// entry per compositor, separated by spaces.
func (c XSDCompositors) String() string {
	parts := make([]string, len(c))
	for i, comp := range c {
		parts[i] = comp.Kind + ":" + strconv.Itoa(comp.ID) + ":" + comp.MinOccurs + ":" + comp.MaxOccurs
	}
	return strings.Join(parts, " ")
}

// UnmarshalXMLAttr implements xml.UnmarshalerAttr, reading the form String
// writes.
func (c *XSDCompositors) UnmarshalXMLAttr(attr xml.Attr) error {
	fields := strings.Fields(attr.Value)
	out := make(XSDCompositors, 0, len(fields))
	for _, f := range fields {
		parts := strings.Split(f, ":")
		if len(parts) != 4 {
			return fmt.Errorf("parser: malformed compositor %q", f)
		}
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			return fmt.Errorf("parser: malformed compositor %q: %w", f, err)
		}
		out = append(out, XSDCompositor{Kind: parts[0], ID: id, MinOccurs: parts[2], MaxOccurs: parts[3]})
	}
	*c = out
	return nil
}

// MarshalXMLAttr implements xml.MarshalerAttr, writing the form String
// returns, and no attribute when c is empty.
func (c XSDCompositors) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	if len(c) == 0 {
		return xml.Attr{}, nil
	}
	return xml.Attr{Name: name, Value: c.String()}, nil
}

// compositorPath returns the compositors among the open elements of stack,
// whose IDs are in ids, from the nearest complexType inwards.
func compositorPath(stack []xml.StartElement, ids []int) XSDCompositors {
	from := 0
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].Name.Local == "complexType" {
			from = i + 1
			break
		}
	}
	var path XSDCompositors
	for i := from; i < len(stack); i++ {
		switch stack[i].Name.Local {
		case "sequence", "choice", "all":
			c := XSDCompositor{Kind: stack[i].Name.Local, ID: ids[i]}
			for _, a := range stack[i].Attr {
				if a.Name.Space != "" {
					continue
				}
				switch a.Name.Local {
				case "minOccurs":
					c.MinOccurs = strings.TrimSpace(a.Value)
				case "maxOccurs":
					c.MaxOccurs = strings.TrimSpace(a.Value)
				}
			}
			path = append(path, c)
		}
	}
	return path
}
