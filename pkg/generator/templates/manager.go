// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package templates

import (
	"text/template"

	"github.com/kastrul/gowsdl/pkg/utils"
)

// Manager manages all code generation templates
type Manager struct {
	funcMap template.FuncMap
}

// defaultFuncs are template functions the templates rely on that callers of
// NewManager may not provide. Functions in the caller's map take precedence.
var defaultFuncs = template.FuncMap{
	"makeFieldPublic": utils.MakeExportedField,
}

// NewManager creates a new template manager
func NewManager(funcMap template.FuncMap) *Manager {
	return &Manager{
		funcMap: funcMap,
	}
}

func (m *Manager) newTemplate(name string) *template.Template {
	return template.New(name).Funcs(defaultFuncs).Funcs(m.funcMap)
}

// GetOperationsTemplate returns the operations template
func (m *Manager) GetOperationsTemplate() (*template.Template, error) {
	return m.newTemplate("operations").Parse(UnifiedOperationsTemplate)
}

// GetTypesTemplate returns the types template
func (m *Manager) GetTypesTemplate() (*template.Template, error) {
	// TODO: Add TypesTemplate constant
	return m.newTemplate("types").Parse("")
}

// GetHeaderTemplate returns the header template
func (m *Manager) GetHeaderTemplate() (*template.Template, error) {
	// TODO: Add HeaderTemplate constant
	return m.newTemplate("header").Parse("")
}

// GetServerTemplate returns the server template
func (m *Manager) GetServerTemplate() (*template.Template, error) {
	// TODO: Add UnifiedServerTemplate constant
	return m.newTemplate("server").Parse("")
}

// GetGenericOperationsTemplate returns the WSDL 1.1 operations template used
// for generic client generation. Generic methods are emitted when the template
// data has UseGenerics set to true.
func (m *Manager) GetGenericOperationsTemplate() (*template.Template, error) {
	return m.newTemplate("generic_operations").Parse(OperationsWSDL1Template)
}
