package generator

import (
	"context"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateCode runs the generator on wsdlPath and returns the gofmt'ed client code
// together with the raw generator output.
func generateCode(t *testing.T, wsdlPath string, opts ...Option) (string, map[string][]byte) {
	t.Helper()

	g, err := New(wsdlPath, opts...)
	require.NoError(t, err)

	out, err := g.Generate(context.Background())
	require.NoError(t, err)

	code := string(out["header"]) + string(out["types"]) + string(out["operations"])
	formatted, err := format.Source([]byte(code))
	require.NoError(t, err)
	return string(formatted), out
}

// assertGeneratedCodeBuilds writes the generator output into a temporary module that
// depends on this repository and checks that it compiles with go build.
func assertGeneratedCodeBuilds(t *testing.T, module string, out map[string][]byte) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping go build of generated code in short mode")
	}

	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)

	dir := t.TempDir()
	client := string(out["header"]) + string(out["types"]) + string(out["operations"])
	require.NoError(t, os.WriteFile(filepath.Join(dir, "client.go"), []byte(client), 0o600))
	if len(out["server"]) > 0 {
		server := string(out["server_header"]) + string(out["server_wsdl"]) + string(out["server"])
		require.NoError(t, os.WriteFile(filepath.Join(dir, "server.go"), []byte(server), 0o600))
	}

	goMod := "module " + module + "\n\ngo 1.21\n\nrequire github.com/kastrul/gowsdl v0.0.0\n\nreplace github.com/kastrul/gowsdl => " + root + "\n"
	goSum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.sum"), goSum, 0o600))

	cmd := exec.Command("go", "build", "-mod=mod", "./...")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err, "generated code should compile:\n%s", output)
}

// structBody returns the body of the struct declaration for name in gofmt'ed code.
func structBody(t *testing.T, code, name string) string {
	t.Helper()
	m := regexp.MustCompile(`(?ms)^type ` + regexp.QuoteMeta(name) + ` struct \{\n(.*?)^\}`).FindStringSubmatch(code)
	require.NotNil(t, m, "struct %s not found", name)
	return m[1]
}

func chromeDataWSDL() string {
	return filepath.Join("..", "..", "fixtures", "chromedata.wsdl")
}

func TestGenerator_ChromeData_GeneratedCodeCompiles(t *testing.T) {
	_, out := generateCode(t, chromeDataWSDL(), WithPackage("chromedata"), WithServerGeneration(true))
	assertGeneratedCodeBuilds(t, "chromedata", out)
}

func TestGenerator_ChromeData_InlineComplexFieldTypes(t *testing.T) {
	code, _ := generateCode(t, chromeDataWSDL(), WithPackage("chromedata"))

	assert.Regexp(t, `BodyType\s+\[\]StyleBodyTypeType\s`, structBody(t, code, "Style"))
	assert.Regexp(t, `Item\s+\[\]ConsumerInformationItemType\s`, structBody(t, code, "ConsumerInformation"))
	gallery := structBody(t, code, "MediaGallery")
	assert.Regexp(t, `View\s+\[\]MediaGalleryViewType\s`, gallery)
	assert.Regexp(t, `Colorized\s+\[\]MediaGalleryColorizedType\s`, gallery)

	bodyType := structBody(t, code, "StyleBodyTypeType")
	assert.Contains(t, bodyType, "IdentifiedString")
	assert.Regexp(t, `Primary\s+bool\s+`+"`"+`xml:"primary,attr`, bodyType)

	stockImage := structBody(t, code, "StyleStockImageType")
	assert.Contains(t, stockImage, "Image")
	assert.Regexp(t, `Filename\s+string\s+`+"`"+`xml:"filename,attr`, stockImage)

	colorized := structBody(t, code, "MediaGalleryColorizedType")
	assert.Regexp(t, `PrimaryColorOptionCode\s+string`, colorized)
	assert.Regexp(t, `ShotCode\s+string`, colorized)
}

const anonymousElementInlineWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/" xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/"
             xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:tns="urn:inline" targetNamespace="urn:inline">
  <types>
    <xs:schema targetNamespace="urn:inline" elementFormDefault="qualified">
      <xs:complexType name="Base">
        <xs:attribute name="url" type="xs:string"/>
      </xs:complexType>
      <xs:element name="GetThing">
        <xs:complexType>
          <xs:sequence>
            <xs:element name="opt" minOccurs="0">
              <xs:complexType>
                <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
              </xs:complexType>
            </xs:element>
            <xs:element name="many" maxOccurs="unbounded">
              <xs:complexType>
                <xs:complexContent>
                  <xs:extension base="tns:Base">
                    <xs:sequence><xs:element name="note" type="xs:string"/></xs:sequence>
                    <xs:attribute name="extra" type="xs:string"/>
                  </xs:extension>
                </xs:complexContent>
              </xs:complexType>
            </xs:element>
            <xs:element name="attrsOnly">
              <xs:complexType><xs:attribute name="x" type="xs:int"/></xs:complexType>
            </xs:element>
          </xs:sequence>
        </xs:complexType>
      </xs:element>
      <xs:complexType name="Holder">
        <xs:choice>
          <xs:element name="picked">
            <xs:complexType><xs:attribute name="y" type="xs:string"/></xs:complexType>
          </xs:element>
          <xs:element name="other" type="xs:string"/>
        </xs:choice>
      </xs:complexType>
      <xs:element name="GetThingResponse">
        <xs:complexType>
          <xs:sequence><xs:element name="holder" type="tns:Holder"/></xs:sequence>
        </xs:complexType>
      </xs:element>
    </xs:schema>
  </types>
  <message name="GetThingIn"><part name="parameters" element="tns:GetThing"/></message>
  <message name="GetThingOut"><part name="parameters" element="tns:GetThingResponse"/></message>
  <portType name="ThingPort">
    <operation name="GetThing">
      <input message="tns:GetThingIn"/>
      <output message="tns:GetThingOut"/>
    </operation>
  </portType>
  <binding name="ThingBinding" type="tns:ThingPort">
    <soap:binding style="document" transport="http://schemas.xmlsoap.org/soap/http"/>
    <operation name="GetThing">
      <soap:operation soapAction="urn:GetThing"/>
      <input><soap:body use="literal"/></input>
      <output><soap:body use="literal"/></output>
    </operation>
  </binding>
  <service name="ThingService">
    <port name="ThingPort" binding="tns:ThingBinding">
      <soap:address location="http://localhost/thing"/>
    </port>
  </service>
</definitions>`

func TestGenerator_InlineComplexTypes_AnonymousElement_Compiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inline.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(anonymousElementInlineWSDL), 0o600))

	code, out := generateCode(t, path, WithPackage("inline"), WithServerGeneration(true))

	for _, name := range []string{"GetThing", "GetThingType"} {
		body := structBody(t, code, name)
		assert.Regexp(t, `Opt\s+\*GetThingOptType\s`, body, name)
		assert.Regexp(t, `Many\s+\[\]GetThingManyType\s`, body, name)
		assert.Regexp(t, `AttrsOnly\s+GetThingAttrsOnlyType\s`, body, name)
	}

	many := structBody(t, code, "GetThingManyType")
	assert.Contains(t, many, "Base")
	assert.Regexp(t, `Note\s+string`, many)
	assert.Regexp(t, `Extra\s+string\s+`+"`"+`xml:"extra,attr`, many)

	assert.Regexp(t, `Picked\s+HolderPickedType\s`, structBody(t, code, "Holder"))

	assertGeneratedCodeBuilds(t, "inline", out)
}
