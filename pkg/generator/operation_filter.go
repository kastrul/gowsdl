// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/utils"
)

// WithOperations restricts client code generation to the named WSDL operations.
// Names are matched exactly against the operation name attribute in the WSDL.
// Port types (WSDL 1.1) or interfaces (WSDL 2.0) left without any selected
// operation are omitted from the client code. An empty list generates all
// operations. Types and server code are not affected.
func WithOperations(names ...string) Option {
	return func(g *Generator) error {
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if g.operations == nil {
				g.operations = make(map[string]bool)
			}
			g.operations[name] = true
		}
		return nil
	}
}

// ParseOperationList splits a comma-separated list of operation names,
// trimming whitespace and dropping empty entries.
func ParseOperationList(list string) []string {
	var names []string
	for _, name := range strings.Split(list, ",") {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// validateOperationFilter reports an error when the filter names operations
// that do not exist in the parsed WSDL.
func (g *Generator) validateOperationFilter() error {
	if len(g.operations) == 0 {
		return nil
	}

	known := make(map[string]bool)
	if g.wsdl != nil {
		for _, pt := range g.soapPortTypes() {
			for _, op := range pt.Operations {
				known[op.Name] = true
			}
		}
	}
	if g.wsdl2 != nil {
		for _, iface := range g.wsdl2.Interfaces {
			for _, op := range iface.Operations {
				known[op.Name] = true
			}
		}
	}

	var unknown []string
	for name := range g.operations {
		if !known[name] {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)

	available := make([]string, 0, len(known))
	for name := range known {
		available = append(available, name)
	}
	sort.Strings(available)

	return fmt.Errorf("unknown operation(s) %s; available operations: %s",
		strings.Join(unknown, ", "), strings.Join(available, ", "))
}

// soapPortTypes returns the WSDL 1.1 port types a SOAP client can reach.
// A port type is left out when it is referenced by at least one binding and
// every binding referencing it is HTTP-only (http:binding without a SOAP
// binding). Port types without any binding are kept.
func (g *Generator) soapPortTypes() []*parser.WSDLPortType {
	if g.wsdl == nil {
		return nil
	}
	skipped := g.httpOnlyPortTypes()
	if len(skipped) == 0 {
		return g.wsdl.PortTypes
	}
	return filterSlice(g.wsdl.PortTypes, func(pt *parser.WSDLPortType) bool { return !skipped[pt.Name] })
}

// httpOnlyPortTypes returns the names of the WSDL 1.1 port types that are
// bound only through HTTP-only bindings.
func (g *Generator) httpOnlyPortTypes() map[string]bool {
	httpOnly := make(map[string]bool)
	soap := make(map[string]bool)
	for _, b := range g.wsdl.Binding {
		name := utils.LocalName(b.Type)
		if b.IsHTTPOnly() {
			httpOnly[name] = true
		} else {
			soap[name] = true
		}
	}
	for name := range soap {
		delete(httpOnly, name)
	}
	return httpOnly
}

// filteredPortTypes returns the SOAP-reachable WSDL 1.1 port types restricted
// to the selected operations. The parsed WSDL is not modified.
func (g *Generator) filteredPortTypes() []*parser.WSDLPortType {
	return filterOperations(g.soapPortTypes(), g.operations,
		func(pt *parser.WSDLPortType) *[]*parser.WSDLOperation { return &pt.Operations },
		func(op *parser.WSDLOperation) string { return op.Name })
}

// filteredInterfaces returns the WSDL 2.0 interfaces restricted to the
// selected operations. The parsed WSDL is not modified.
func (g *Generator) filteredInterfaces() []*parser.WSDL2Interface {
	return filterOperations(g.wsdl2.Interfaces, g.operations,
		func(iface *parser.WSDL2Interface) *[]*parser.WSDL2Operation { return &iface.Operations },
		func(op *parser.WSDL2Operation) string { return op.Name })
}

// filterOperations returns shallow copies of containers restricted to the
// operations named in selected, dropping containers left without any. An
// empty selection returns containers unchanged. The originals are not
// modified.
func filterOperations[C, O any](containers []*C, selected map[string]bool, operations func(*C) *[]O, name func(O) string) []*C {
	if len(selected) == 0 {
		return containers
	}

	result := make([]*C, 0, len(containers))
	for _, c := range containers {
		var ops []O
		for _, op := range *operations(c) {
			if selected[name(op)] {
				ops = append(ops, op)
			}
		}
		if len(ops) == 0 {
			continue
		}
		filtered := *c
		*operations(&filtered) = ops
		result = append(result, &filtered)
	}
	return result
}
