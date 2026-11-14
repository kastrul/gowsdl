package soap

import (
	"bytes"
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type partsIdentifier struct {
	XMLName    xml.Name `xml:"urn:hdr client"`
	Instance   string   `xml:"http://example.com/id xRoadInstance,omitempty"`
	ObjectType string   `xml:"http://example.com/id objectType,attr,omitempty"`
}

type partsID string

type partsHash struct {
	Value     string `xml:",chardata"`
	Algorithm string `xml:"algorithmId,attr,omitempty"`
}

type partsHeader struct {
	Client   partsIdentifier `xml:"urn:hdr client,omitempty"`
	ID       partsID         `xml:"urn:hdr id,omitempty"`
	UserID   partsID         `xml:"urn:hdr userId,omitempty"`
	Hash     partsHash       `xml:"urn:hdr requestHash,omitempty"`
	Plain    string          `xml:"plain"`
	Untagged string
	Skipped  string `xml:"-"`
	hidden   string
}

func (h partsHeader) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	return MarshalParts(e, h)
}

func TestMarshalParts_WritesNonZeroPartsAsSiblings(t *testing.T) {
	hdr := SOAPHeader{Headers: []any{partsHeader{
		Client:   partsIdentifier{Instance: "EE", ObjectType: "SUBSYSTEM"},
		ID:       "msg-1",
		Untagged: "u",
		Skipped:  "never",
		hidden:   "never",
	}}}

	out, err := xml.Marshal(hdr)
	require.NoError(t, err)

	assert.Equal(t,
		`<soap:Header>`+
			`<client xmlns="urn:hdr" xmlns:id="http://example.com/id" id:objectType="SUBSYSTEM"><xRoadInstance xmlns="http://example.com/id">EE</xRoadInstance></client>`+
			`<id xmlns="urn:hdr">msg-1</id>`+
			`<Untagged>u</Untagged>`+
			`</soap:Header>`,
		string(out))
}

func TestMarshalParts_PointerAndNonStruct(t *testing.T) {
	encode := func(v any) (string, error) {
		var buf bytes.Buffer
		e := xml.NewEncoder(&buf)
		if err := MarshalParts(e, v); err != nil {
			return "", err
		}
		err := e.Flush()
		return buf.String(), err
	}

	out, err := encode((*partsHeader)(nil))
	require.NoError(t, err)
	assert.Empty(t, out)

	out, err = encode(&partsHeader{Plain: "p"})
	require.NoError(t, err)
	assert.Equal(t, `<plain>p</plain>`, out)

	_, err = encode("not a struct")
	assert.Error(t, err)
}
