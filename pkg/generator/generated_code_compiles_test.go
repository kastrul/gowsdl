package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chromeDataWSDL() string {
	return filepath.Join("..", "..", "fixtures", "chromedata.wsdl")
}

func TestGenerator_ChromeData_GeneratedCodeCompiles(t *testing.T) {
	out := generate(t, chromeDataWSDL(), WithPackage("chromedata"), WithServerGeneration(true))
	assertGeneratedCodeBuilds(t, "chromedata", out)
}

func TestGenerator_ChromeData_InlineComplexFieldTypes(t *testing.T) {
	code := generate(t, chromeDataWSDL(), WithPackage("chromedata")).client(t)

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

	out := generate(t, path, WithPackage("inline"), WithServerGeneration(true))
	code := out.client(t)

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

	assert.Regexp(t, `Picked\s+\*HolderPickedType\s`, structBody(t, code, "Holder"))

	assertGeneratedCodeBuilds(t, "inline", out)
}

const soapImport = `"github.com/kastrul/gowsdl/soap"`

func TestGenerator_NoOperations_GeneratedCodeCompiles(t *testing.T) {
	path := filepath.Join("..", "..", "fixtures", "nested_complex_type_test.wsdl")
	out := generate(t, path, WithPackage("nested"))

	header := string(out["header"])
	assert.NotContains(t, header, `"context"`)
	assert.NotContains(t, header, soapImport)

	assertGeneratedCodeBuilds(t, "nested", out)
}

const typesOnlyDateTimeWSDL = `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns="http://schemas.xmlsoap.org/wsdl/"
             xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:tns="urn:typesonly" targetNamespace="urn:typesonly">
  <types>
    <xs:schema targetNamespace="urn:typesonly" elementFormDefault="qualified">
      <xs:complexType name="Event">
        <xs:sequence>
          <xs:element name="name" type="xs:string"/>
          <xs:element name="at" type="xs:dateTime"/>
        </xs:sequence>
      </xs:complexType>
    </xs:schema>
  </types>
</definitions>`

func TestGenerator_NoOperations_SOAPTypesKeepSOAPImport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "typesonly.wsdl")
	require.NoError(t, os.WriteFile(path, []byte(typesOnlyDateTimeWSDL), 0o600))

	out := generate(t, path, WithPackage("typesonly"))
	code := out.client(t)
	require.Contains(t, code, "soap.XSDDateTime")

	header := string(out["header"])
	assert.NotContains(t, header, `"context"`)
	assert.Contains(t, header, soapImport)

	assertGeneratedCodeBuilds(t, "typesonly", out)
}

func TestGenerator_WithOperations_ImportsContextAndSOAP(t *testing.T) {
	out := generate(t, chromeDataWSDL(), WithPackage("chromedata"))

	header := string(out["header"])
	assert.Contains(t, header, `"context"`)
	assert.Contains(t, header, soapImport)
}
