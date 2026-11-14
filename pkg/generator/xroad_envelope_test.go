package generator

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// xroadMixedQualificationWSDL has the shape of X-Road producer WSDLs such as
// KIRST: the operation schema leaves elementFormDefault unqualified, while
// small inline schemas for the X-Road header and identifiers are qualified, and
// the client identifier type is a complexContent restriction.
const xroadMixedQualificationWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<wsdl:definitions xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"
	xmlns:xsd="http://www.w3.org/2001/XMLSchema"
	xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
	xmlns:svc="http://example.com/svc"
	xmlns:xroad="http://example.com/xroad"
	xmlns:id="http://example.com/identifiers"
	targetNamespace="http://example.com/svc">
	<wsdl:types>
		<xsd:schema targetNamespace="http://example.com/svc">
			<xsd:complexType name="queryRequestType">
				<xsd:sequence><xsd:element name="code" type="xsd:string"/></xsd:sequence>
			</xsd:complexType>
			<xsd:element name="query">
				<xsd:complexType>
					<xsd:sequence><xsd:element name="request" type="svc:queryRequestType"/></xsd:sequence>
				</xsd:complexType>
			</xsd:element>
			<xsd:element name="queryResponse">
				<xsd:complexType>
					<xsd:sequence><xsd:element name="result" type="xsd:string"/></xsd:sequence>
				</xsd:complexType>
			</xsd:element>
		</xsd:schema>
		<xsd:schema targetNamespace="http://example.com/xroad" elementFormDefault="qualified">
			<xsd:import namespace="http://example.com/identifiers"/>
			<xsd:element name="client" type="id:ClientIdentifierType"/>
			<xsd:element name="id" type="xsd:string"/>
			<xsd:element name="userId" type="xsd:string"/>
		</xsd:schema>
		<xsd:schema targetNamespace="http://example.com/identifiers" elementFormDefault="qualified">
			<xsd:element name="xRoadInstance" type="xsd:string"/>
			<xsd:element name="memberCode" type="xsd:string"/>
			<xsd:element name="serviceCode" type="xsd:string"/>
			<xsd:attribute name="objectType" type="xsd:string"/>
			<xsd:complexType name="IdentifierType">
				<xsd:sequence>
					<xsd:element minOccurs="0" ref="id:xRoadInstance"/>
					<xsd:element minOccurs="0" ref="id:memberCode"/>
					<xsd:element minOccurs="0" ref="id:serviceCode"/>
				</xsd:sequence>
				<xsd:attribute ref="id:objectType" use="required"/>
			</xsd:complexType>
			<xsd:complexType name="ClientIdentifierType">
				<xsd:complexContent>
					<xsd:restriction base="id:IdentifierType">
						<xsd:sequence>
							<xsd:element ref="id:xRoadInstance"/>
							<xsd:element ref="id:memberCode"/>
						</xsd:sequence>
						<xsd:attribute ref="id:objectType" use="required"/>
					</xsd:restriction>
				</xsd:complexContent>
			</xsd:complexType>
		</xsd:schema>
	</wsdl:types>
	<wsdl:message name="XRoadHeader">
		<wsdl:part name="client" element="xroad:client"/>
		<wsdl:part name="id" element="xroad:id"/>
		<wsdl:part name="userId" element="xroad:userId"/>
	</wsdl:message>
	<wsdl:message name="query"><wsdl:part name="body" element="svc:query"/></wsdl:message>
	<wsdl:message name="queryResponse"><wsdl:part name="body" element="svc:queryResponse"/></wsdl:message>
	<wsdl:portType name="SvcPortType">
		<wsdl:operation name="query">
			<wsdl:input message="svc:query"/>
			<wsdl:output message="svc:queryResponse"/>
		</wsdl:operation>
	</wsdl:portType>
	<wsdl:binding name="SvcBinding" type="svc:SvcPortType">
		<soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
		<wsdl:operation name="query">
			<soap:operation soapAction=""/>
			<wsdl:input>
				<soap:body use="literal"/>
				<soap:header message="svc:XRoadHeader" part="client" use="literal"/>
				<soap:header message="svc:XRoadHeader" part="id" use="literal"/>
				<soap:header message="svc:XRoadHeader" part="userId" use="literal"/>
			</wsdl:input>
			<wsdl:output><soap:body use="literal"/></wsdl:output>
		</wsdl:operation>
	</wsdl:binding>
</wsdl:definitions>`

func TestGenerator_XRoad_MixedQualificationTypes(t *testing.T) {
	out, err := tryGenerate(t, writeWSDL(t, xroadMixedQualificationWSDL), WithPackage("svc"), WithExportAllTypes(true))
	require.NoError(t, err)
	out.client(t)
	code := string(out["types"])

	assert.Contains(t, code, "`xml:\"tns:query\"`", "unqualified operation schema must not be qualified by the inline X-Road schemas")
	assert.Contains(t, code, "func (t Query) TargetNamespace() string")
	assert.Contains(t, code, "`xml:\"http://example.com/xroad client\"`")
	assert.NotContains(t, code, "func (t Client) TargetNamespace() string")

	assert.Regexp(t, `XRoadInstance\s+XRoadInstance\s+`+"`"+`xml:"http://example.com/identifiers xRoadInstance,omitempty"`, code,
		"complexContent restriction must keep its particles, qualified with the identifiers namespace")
	assert.Regexp(t, `ObjectType\s+string\s+`+"`"+`xml:"http://example.com/identifiers objectType,attr,omitempty"`, code,
		"a ref to a global attribute must carry its namespace")
	assert.Contains(t, code, "return soap.MarshalParts(e, h)")
}

const xroadEnvelopeProgram = `package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"svc"

	"github.com/kastrul/gowsdl/soap"
)

func main() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		os.Stdout.Write(body)
		w.Header().Set("Content-Type", "text/xml")
		io.WriteString(w, ` + "`" + `<e:Envelope xmlns:e="http://schemas.xmlsoap.org/soap/envelope/"><e:Body><s:queryResponse xmlns:s="http://example.com/svc"><result>ok</result></s:queryResponse></e:Body></e:Envelope>` + "`" + `)
	}))
	defer srv.Close()

	client := soap.NewClient(srv.URL)
	client.AddHeader(svc.XRoadHeader{
		Client: svc.Client{ClientIdentifierType: svc.ClientIdentifierType{
			ObjectType: "SUBSYSTEM", XRoadInstance: "EE", MemberCode: "123",
		}},
		Id: "msg-1",
	})
	resp, err := svc.NewSvcPortType(client).Query(context.Background(), &svc.Query{Request: svc.QueryRequestType{Code: "X"}})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if resp.Result != "ok" {
		fmt.Fprintf(os.Stderr, "unexpected result %q\n", resp.Result)
		os.Exit(1)
	}
}
`

// xmlNodes lists the elements and attributes of doc in document order as
// "{namespace}local" for elements and "@{namespace}local" for attributes,
// skipping namespace declarations.
func xmlNodes(t *testing.T, doc string) []string {
	t.Helper()
	var nodes []string
	d := xml.NewDecoder(strings.NewReader(doc))
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return nodes
		}
		require.NoError(t, err)
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		nodes = append(nodes, "{"+start.Name.Space+"}"+start.Name.Local)
		for _, a := range start.Attr {
			if a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns") {
				continue
			}
			nodes = append(nodes, "@{"+a.Name.Space+"}"+a.Name.Local)
		}
	}
}

func TestGenerator_XRoad_MixedQualificationEnvelope(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go run of generated code in short mode")
	}

	out, err := tryGenerate(t, writeWSDL(t, xroadMixedQualificationWSDL), WithPackage("svc"), WithExportAllTypes(true))
	require.NoError(t, err)

	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	goSum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	require.NoError(t, err)

	dir := t.TempDir()
	goMod := "module svc\n\ngo 1.21\n\nrequire github.com/kastrul/gowsdl v0.0.0\n\nreplace github.com/kastrul/gowsdl => " + root + "\n"
	client := string(out["header"]) + string(out["types"]) + string(out["operations"])
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.sum"), goSum, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "svc.go"), []byte(client), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd", "envelope"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", "envelope", "main.go"), []byte(xroadEnvelopeProgram), 0o600))

	cmd := exec.Command("go", "run", "-mod=mod", "./cmd/envelope")
	cmd.Dir = dir
	cmd.Stderr = &strings.Builder{}
	stdout, err := cmd.Output()
	require.NoError(t, err, "running generated code failed:\n%s", cmd.Stderr)

	const (
		soapNS = "{http://schemas.xmlsoap.org/soap/envelope/}"
		xroad  = "{http://example.com/xroad}"
		ids    = "{http://example.com/identifiers}"
	)
	assert.Equal(t, []string{
		soapNS + "Envelope",
		soapNS + "Header",
		xroad + "client",
		"@" + ids + "objectType",
		ids + "xRoadInstance",
		ids + "memberCode",
		xroad + "id",
		soapNS + "Body",
		"{http://example.com/svc}query",
		"{}request",
		"{}code",
	}, xmlNodes(t, string(stdout)), "envelope:\n%s", stdout)
}
