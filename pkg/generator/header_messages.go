// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"strings"

	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/utils"
)

// isHeaderMessage reports whether msg is bound only as a SOAP header: some
// binding operation names it in soap:header or soap:headerfault, and no port
// type operation uses it as an input, output or fault. The parts of such a
// message are sibling header blocks rather than the children of a wrapper.
func (g *Generator) isHeaderMessage(msg *parser.WSDLMessage) bool {
	if g.wsdl == nil || msg == nil {
		return false
	}

	for _, pt := range g.wsdl.PortTypes {
		for _, op := range pt.Operations {
			refs := []string{op.Input.Message, op.Output.Message}
			for _, f := range op.Faults {
				refs = append(refs, f.Message)
			}
			for _, ref := range refs {
				if utils.LocalName(ref) == msg.Name {
					return false
				}
			}
		}
	}

	names := func(headers []*parser.WSDLSOAPHeader) bool {
		for _, h := range headers {
			if utils.LocalName(h.Message) == msg.Name {
				return true
			}
			for _, hf := range h.HeadersFault {
				if utils.LocalName(hf.Message) == msg.Name {
					return true
				}
			}
		}
		return false
	}
	for _, b := range g.wsdl.Binding {
		for _, op := range b.Operations {
			if names(op.Input.SOAPHeader) || names(op.Output.SOAPHeader) {
				return true
			}
		}
	}
	return false
}

// partXMLName returns the name to use in the xml struct tag of a message part:
// the local name of its element, preceded by the element's namespace URI when
// the prefix is declared on the WSDL definitions, or the part name for a part
// declared with a type.
func (g *Generator) partXMLName(part *parser.WSDLPart) string {
	if part.Element == "" {
		return part.Name
	}
	local := utils.LocalName(part.Element)
	if g.wsdl == nil {
		return local
	}
	prefix, _, ok := strings.Cut(part.Element, ":")
	if !ok {
		prefix = ""
	}
	if ns := g.wsdl.Xmlns[prefix]; ns != "" {
		return ns + " " + local
	}
	return local
}
