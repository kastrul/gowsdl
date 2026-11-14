package soap

import (
	"encoding/base64"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type binaryHolder struct {
	XMLName xml.Name `xml:"holder"`
	Data    *Binary  `xml:"data"`
}

func TestBinary_MarshalXML_WithoutMTOMEncodesBase64(t *testing.T) {
	payload := []byte("hello <world> & \x00\xff")

	out, err := xml.Marshal(binaryHolder{Data: NewBinary(payload)})
	require.NoError(t, err)

	want := base64.StdEncoding.EncodeToString(payload)
	assert.Equal(t, "<holder><data>"+want+"</data></holder>", string(out))

	var decoded binaryHolder
	require.NoError(t, xml.Unmarshal(out, &decoded))
	assert.Equal(t, want, string(decoded.Data.Bytes()), "UnmarshalXML keeps the element's character data as-is")
	raw, err := base64.StdEncoding.DecodeString(string(decoded.Data.Bytes()))
	require.NoError(t, err)
	assert.Equal(t, payload, raw)
}

func TestClient_Call_BinaryWithoutMTOM(t *testing.T) {
	var got []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body></soap:Body></soap:Envelope>`))
	}))
	defer ts.Close()

	req := &PingRequest{Attachment: NewBinary([]byte("Attached data"))}
	require.NoError(t, NewClient(ts.URL).Call("GetData", req, &PingResponse{}))
	assert.Contains(t, string(got), "<Attachment>"+base64.StdEncoding.EncodeToString([]byte("Attached data"))+"</Attachment>")
}

type multiBinaryRequest struct {
	XMLName xml.Name  `xml:"http://example.com/service.xsd Upload"`
	Files   []*Binary `xml:"File"`
}

func echoServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range r.Header {
			w.Header().Set(k, v[0])
		}
		body, _ := io.ReadAll(r.Body)
		_, _ = w.Write(body)
	}))
}

func TestClient_MTOM_BinarySlice(t *testing.T) {
	ts := echoServer()
	defer ts.Close()

	req := &multiBinaryRequest{Files: []*Binary{
		NewBinary([]byte("first")).SetContentType("text/plain"),
		NewBinary([]byte("second")).SetContentType("application/pdf"),
	}}
	reply := &multiBinaryRequest{}
	require.NoError(t, NewClient(ts.URL, WithMTOM()).Call("Upload", req, reply))

	require.Len(t, reply.Files, 2)
	assert.Equal(t, []byte("first"), reply.Files[0].Bytes())
	assert.Equal(t, "text/plain", reply.Files[0].ContentType())
	assert.Equal(t, []byte("second"), reply.Files[1].Bytes())
	assert.Equal(t, "application/pdf", reply.Files[1].ContentType())
}
