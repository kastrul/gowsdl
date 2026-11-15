// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package templates

// OperationsWSDL1Template is the template for WSDL 1.1 operations
const OperationsWSDL1Template = `
{{range .PortTypes}}
	{{$portType := .Name | makePublic}}{{$portTypeName := .Name}}

	type {{$portType}} interface {
		{{range .Operations}}
			{{$requestType := findType .Input.Message | replaceReservedWords | makePublic}}
			{{$responseType := findType .Output.Message | replaceReservedWords | makePublic}}{{$generic := and $.UseGenerics (not (isRPCPortType $portTypeName)) (ne $requestType "") (ne $responseType "")}}

			{{if .Faults}}
			// Error can be either of the following types:
			{{range .Faults}}
			//   - {{.Name}}{{if .Doc}} {{.Doc}}{{end}}{{end}}
			{{end}}
			{{if .Doc}}{{comment .Doc}}{{end}}
			{{if not .Output.Message -}}
			{{.Name | makePublic}}(ctx context.Context, request *{{$requestType}}) error
			{{- else if and $generic $.GenericsOnly -}}
			{{.Name | makePublic}}(ctx context.Context, request *{{$requestType}}) (soap.Result[{{$responseType}}], error)
			{{- else -}}
			{{.Name | makePublic}}(ctx context.Context, request *{{$requestType}}) (*{{$responseType}}, error)
			{{- end}}
			{{if and $generic (not $.GenericsOnly)}}
			{{.Name | makePublic}}Generic(ctx context.Context, request *{{$requestType}}) (soap.Result[{{$responseType}}], error)
			{{end}}
		{{end}}
	}

	type {{$portType | makePrivate}} struct {
		client *soap.Client
	}

	func New{{$portType}}(client *soap.Client) {{$portType}} {
		return &{{$portType | makePrivate}}{
			client: client,
		}
	}

	{{range .Operations}}
		{{$requestType := findType .Input.Message | replaceReservedWords | makePublic}}
		{{$responseType := findType .Output.Message | replaceReservedWords | makePublic}}
		{{$soapAction := findSOAPAction .Name $portTypeName}}{{$generic := and $.UseGenerics (not (isRPCPortType $portTypeName)) (ne $requestType "") (ne $responseType "")}}

		{{if not (and $generic $.GenericsOnly) -}}
		{{$oneWay := not .Output.Message}}
		func (service *{{$portType | makePrivate}}) {{.Name | makePublic}}(ctx context.Context, request *{{$requestType}}) {{if $oneWay}}error{{else}}(*{{$responseType}}, error){{end}} {
			{{if isRPCPortType $portTypeName}}
				// RPC-style operation: wrap request in operation element
				{{$opName := .Name}}
				{{$inputMsgName := .Input.Message | removeNamespacePrefix}}
				// Create operation wrapper directly
				α := Operation{{$opName | makePublic}}In{
					{{/* Convert request fields to operation wrapper */}}
					{{range $.Messages}}
						{{if eq .Name $inputMsgName}}
							{{with bodyElementPart .}}
								{{/* The request is the element of the message's only body part */}}
								{{.Name | replaceReservedWords | makeFieldPublic}}: request,
							{{else}}
								{{range .Parts}}
									{{.Name | replaceReservedWords | makeFieldPublic}}: &request.{{.Name | replaceReservedWords | makeFieldPublic}},
								{{end}}
							{{end}}
						{{end}}
					{{end}}
				}
				{{if $oneWay}}
				return service.client.CallContext(ctx, "{{$soapAction}}", α, nil)
				{{else}}
				var γ Operation{{.Name | makePublic}}Out
				
				err := service.client.CallContext(ctx, "{{$soapAction}}", α, &γ)
				if err != nil {
					return nil, err
				}
				
				// Convert response wrapper to output type
				{{$outputMsgName := .Output.Message | removeNamespacePrefix}}
				{{range $.Messages}}
					{{if eq .Name $outputMsgName}}
						{{with bodyElementPart .}}
							{{/* The response is the element of the message's only body part */}}
							{{$field := .Name | replaceReservedWords | makeFieldPublic}}
							{{if eq .Name "return"}}{{$field = "Return"}}{{end}}
							if γ.{{$field}} == nil {
								return nil, nil
							}
							return γ.{{$field}}, nil
						{{else}}{{if eq (len .Parts) 1}}
							{{range .Parts}}
								{{$partName := .Name | replaceReservedWords | makeFieldPublic}}
								{{if eq .Name "return"}}
									if γ.Return == nil {
										return nil, nil
									}
									result := &{{$responseType}}{
										Result: *γ.Return,
									}
									return result, nil
								{{else}}
									if γ.{{$partName}} == nil {
										return nil, nil
									}
									result := &{{$responseType}}{
										{{$partName}}: *γ.{{$partName}},
									}
									return result, nil
								{{end}}
							{{end}}
						{{else}}
							// Multiple parts - need to handle each field
							result := &{{$responseType}}{
								{{range .Parts}}
									{{$partName := .Name | replaceReservedWords | makeFieldPublic}}
									{{$partName}}: γ.{{$partName}},
								{{end}}
							}
							return result, nil
						{{end}}{{end}}
					{{end}}
				{{end}}
				{{end}}
			{{else if $oneWay}}
				// One-way operation: no response is expected
				return service.client.CallContext(ctx, "{{$soapAction}}", request, nil)
			{{else}}
				// Document-style operation
				response := new({{$responseType}})
				err := service.client.CallContext(ctx, "{{$soapAction}}", request, response)
				if err != nil {
					return nil, err
				}
				return response, nil
			{{end}}
		}
		{{- end}}

		{{if $generic}}
		func (service *{{$portType | makePrivate}}) {{.Name | makePublic}}{{if not $.GenericsOnly}}Generic{{end}}(ctx context.Context, request *{{$requestType}}) (soap.Result[{{$responseType}}], error) {
			if request == nil {
				return soap.Result[{{$responseType}}]{}, soap.ErrNilRequest
			}
			return service.client.CallResult[{{$responseType}}](ctx, "{{$soapAction}}", request), nil
		}
		{{end}}
	{{end}}
{{end}}
`
