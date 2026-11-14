package generator

import (
	"context"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kastrul/gowsdl/pkg/parser"
)

func TestRenderTemplate_RendersWithFuncMap(t *testing.T) {
	funcMap := template.FuncMap{"shout": func(s string) string { return s + "!" }}

	out, err := renderTemplate("greeting", `hello {{shout .}}`, funcMap, "world")

	require.NoError(t, err)
	assert.Equal(t, "hello world!", string(out))
}

func TestRenderTemplate_WrapsParseErrorWithTemplateName(t *testing.T) {
	_, err := renderTemplate("broken", `{{`, nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse broken template")
}

func TestRenderTemplate_WrapsExecuteErrorWithTemplateName(t *testing.T) {
	_, err := renderTemplate("failing", `{{.Missing}}`, nil, struct{}{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute failing template")
}

func TestGenerateCode_PartsDependOnServerGeneration(t *testing.T) {
	for _, tt := range []struct {
		server bool
		want   []string
	}{
		{false, []string{"header", "operations", "types"}},
		{true, []string{"header", "operations", "server", "server_header", "server_wsdl", "types"}},
	} {
		g, err := New("../../fixtures/stock.wsdl", WithPackage("stock"), WithServerGeneration(tt.server))
		require.NoError(t, err)

		out, err := g.Generate(context.Background())
		require.NoError(t, err)

		keys := make([]string, 0, len(out))
		for k := range out {
			keys = append(keys, k)
		}
		assert.ElementsMatch(t, tt.want, keys)
	}
}

func TestFilterOperations(t *testing.T) {
	op := func(name string) *parser.WSDLOperation { return &parser.WSDLOperation{Name: name} }
	a := &parser.WSDLPortType{Name: "A", Operations: []*parser.WSDLOperation{op("One"), op("Two")}}
	b := &parser.WSDLPortType{Name: "B", Operations: []*parser.WSDLOperation{op("Three")}}
	all := []*parser.WSDLPortType{a, b}
	ops := func(pt *parser.WSDLPortType) *[]*parser.WSDLOperation { return &pt.Operations }
	name := func(o *parser.WSDLOperation) string { return o.Name }

	assert.Equal(t, all, filterOperations(all, nil, ops, name))

	got := filterOperations(all, map[string]bool{"Two": true}, ops, name)
	require.Len(t, got, 1)
	assert.Equal(t, "A", got[0].Name)
	require.Len(t, got[0].Operations, 1)
	assert.Equal(t, "Two", got[0].Operations[0].Name)
	assert.Len(t, a.Operations, 2, "the original port type must not be modified")
}
