package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kastrul/gowsdl/pkg/generator/templates"
	"github.com/kastrul/gowsdl/pkg/parser"
	"github.com/kastrul/gowsdl/pkg/types"
)

func TestServerOperationsWSDL1_ResolvesTypesFromMessages(t *testing.T) {
	g := &Generator{
		typeMapper: types.NewTypeMapper(),
		wsdl: &parser.WSDL{Messages: []*parser.WSDLMessage{
			{Name: "GetIn", Parts: []*parser.WSDLPart{{Name: "parameters", Element: "tns:getOrder"}}},
			{Name: "GetOut", Parts: []*parser.WSDLPart{{Name: "parameters", Element: "tns:getOrderResponse"}}},
		}},
	}
	portTypes := []*parser.WSDLPortType{{Name: "Orders", Operations: []*parser.WSDLOperation{{
		Name:   "Get",
		Input:  parser.WSDLInput{Message: "tns:GetIn"},
		Output: parser.WSDLOutput{Message: "tns:GetOut"},
	}}}}

	assert.Equal(t, []templates.ServerOperation{
		{Request: "GetOrder", Response: "GetOrderResponse", HasRequest: true, HasResponse: true},
	}, g.serverOperationsWSDL1(portTypes))
}

func TestServerOperationsWSDL2_ResolvesTypesFromElements(t *testing.T) {
	g := &Generator{typeMapper: types.NewTypeMapper(), wsdl2: &parser.WSDL2{}}
	interfaces := []*parser.WSDL2Interface{{Name: "Orders", Operations: []*parser.WSDL2Operation{
		{Name: "Get", Input: &parser.WSDL2OperationMsg{Element: "tns:getOrder"}, Output: &parser.WSDL2OperationMsg{Element: "tns:getOrderResponse"}},
		{Name: "Notify", Input: &parser.WSDL2OperationMsg{Element: "tns:notify"}},
		{Name: "Beat", Output: &parser.WSDL2OperationMsg{Element: "tns:heartbeat"}},
	}}}

	assert.Equal(t, []templates.ServerOperation{
		{Request: "GetOrder", Response: "GetOrderResponse", HasRequest: true, HasResponse: true},
		{Request: "Notify", HasRequest: true},
		{Response: "Heartbeat", HasResponse: true},
	}, g.serverOperationsWSDL2(interfaces))
}

func TestServerTemplate_OneTextServesBothVersions(t *testing.T) {
	assert.Equal(t, templates.ServerTemplate, templates.ServerWSDL2Template)

	ops := []templates.ServerOperation{
		{Request: "Get", Response: "GetResponse", HasRequest: true, HasResponse: true},
		{Request: "Notify", HasRequest: true},
	}
	g := &Generator{typeMapper: types.NewTypeMapper()}

	v1, err := renderTemplate("server", templates.ServerTemplate, g.createFuncMap(), templates.ServerData{Operations: ops})
	require.NoError(t, err)
	v2, err := renderTemplate("server", templates.ServerWSDL2Template, g.createFuncMap(), templates.ServerData{
		Operations:              ops,
		AcceptSOAP12ContentType: true,
	})
	require.NoError(t, err)

	for _, code := range []string{string(v1), string(v2)} {
		assert.Contains(t, code, "Get *Get `xml:\",omitempty\"`")
		assert.Contains(t, code, "Notify *Notify `xml:\",omitempty\"`")
		assert.Contains(t, code, "Get *GetResponse `xml:\",omitempty\"`")
		assert.Contains(t, code, "func (service *SOAPBodyRequest) GetFunc(request *Get) (*GetResponse, error)")
		assert.NotContains(t, code, "NotifyFunc")
	}
	assert.Contains(t, string(v1), "Invalid content type for SOAP 1.1; expected 'text/xml'")
	assert.NotContains(t, string(v1), "application/soap+xml")
	assert.Contains(t, string(v2), `!strings.Contains(header, "application/soap+xml")`)
	assert.NotContains(t, string(v2), "Invalid content type for SOAP 1.1")
}
