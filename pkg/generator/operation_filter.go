// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kastrul/gowsdl/pkg/parser"
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
		for _, pt := range g.wsdl.PortTypes {
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

// filteredPortTypes returns the WSDL 1.1 port types restricted to the
// selected operations. The parsed WSDL is not modified.
func (g *Generator) filteredPortTypes() []*parser.WSDLPortType {
	if len(g.operations) == 0 {
		return g.wsdl.PortTypes
	}

	result := make([]*parser.WSDLPortType, 0, len(g.wsdl.PortTypes))
	for _, pt := range g.wsdl.PortTypes {
		var ops []*parser.WSDLOperation
		for _, op := range pt.Operations {
			if g.operations[op.Name] {
				ops = append(ops, op)
			}
		}
		if len(ops) == 0 {
			continue
		}
		filtered := *pt
		filtered.Operations = ops
		result = append(result, &filtered)
	}
	return result
}

// filteredInterfaces returns the WSDL 2.0 interfaces restricted to the
// selected operations. The parsed WSDL is not modified.
func (g *Generator) filteredInterfaces() []*parser.WSDL2Interface {
	if len(g.operations) == 0 {
		return g.wsdl2.Interfaces
	}

	result := make([]*parser.WSDL2Interface, 0, len(g.wsdl2.Interfaces))
	for _, iface := range g.wsdl2.Interfaces {
		var ops []*parser.WSDL2Operation
		for _, op := range iface.Operations {
			if g.operations[op.Name] {
				ops = append(ops, op)
			}
		}
		if len(ops) == 0 {
			continue
		}
		filtered := *iface
		filtered.Operations = ops
		result = append(result, &filtered)
	}
	return result
}
