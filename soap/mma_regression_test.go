package soap

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mmaResponseServer(rootContentType string) *httptest.Server {
	const boundary = "MMABoundary"
	body := fmt.Sprintf("--%[1]s\r\n"+
		"Content-Type: %[2]s\r\n"+
		"Content-ID: <soaprequest@gowsdl.lib>\r\n\r\n"+
		`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body>`+
		`<attachmentRequest xmlns="http://example.com/service.xsd"><name>reply</name><contentID>file1</contentID></attachmentRequest>`+
		"</soap:Body></soap:Envelope>\r\n"+
		"--%[1]s\r\n"+
		"Content-Type: application/octet-stream\r\n"+
		"Content-ID: <file1>\r\n\r\n"+
		"payload\r\n"+
		"--%[1]s--\r\n", boundary, rootContentType)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", fmt.Sprintf(mmaContentType, boundary))
		_, _ = w.Write([]byte(body))
	}))
}

func TestClient_MMA_AttachmentsWithoutAttachmentSlice(t *testing.T) {
	ts := mmaResponseServer("text/xml;charset=UTF-8")
	defer ts.Close()

	client := NewClient(ts.URL, WithMIMEMultipartAttachments())
	req := &AttachmentRequest{Name: "upload"}

	calls := map[string]func(reply *AttachmentRequest) error{
		"Call": func(reply *AttachmentRequest) error { return client.Call("''", req, reply) },
		"CallContext": func(reply *AttachmentRequest) error {
			return client.CallContext(context.Background(), "''", req, reply)
		},
		"CallWithFaultDetail": func(reply *AttachmentRequest) error {
			return client.CallWithFaultDetail("''", req, reply, nil)
		},
		"CallContextWithFaultDetail": func(reply *AttachmentRequest) error {
			return client.CallContextWithFaultDetail(context.Background(), "''", req, reply, nil)
		},
		"CallContextWithAttachmentsAndFaultDetail": func(reply *AttachmentRequest) error {
			return client.CallContextWithAttachmentsAndFaultDetail(context.Background(), "''", req, reply, nil, nil)
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			reply := &AttachmentRequest{}
			require.NoError(t, call(reply))
			assert.Equal(t, "reply", reply.Name)
		})
	}
}

func TestClient_MMA_RootPartContentTypeVariants(t *testing.T) {
	for _, ct := range []string{
		"text/xml;charset=UTF-8",
		"text/xml; charset=utf-8",
		`text/xml; charset="UTF-8"`,
		"text/xml",
		"TEXT/XML; Charset=UTF-8",
	} {
		t.Run(ct, func(t *testing.T) {
			ts := mmaResponseServer(ct)
			defer ts.Close()

			reply := &AttachmentRequest{}
			var attachments []MIMEMultipartAttachment
			require.NoError(t, NewClient(ts.URL, WithMIMEMultipartAttachments()).CallContextWithAttachmentsAndFaultDetail(
				context.Background(), "''", &AttachmentRequest{Name: "upload"}, reply, nil, &attachments))

			assert.Equal(t, "reply", reply.Name)
			assert.Equal(t, "file1", reply.ContentID)
			assert.Equal(t, []MIMEMultipartAttachment{{Name: "file1", Data: []byte("payload")}}, attachments)
		})
	}
}

func TestMMADecoder_LaterXMLPartStaysAttachment(t *testing.T) {
	const boundary = "MMABoundary"
	body := fmt.Sprintf("--%[1]s\r\n"+
		"Content-Type: text/xml; charset=utf-8\r\n\r\n"+
		`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body>`+
		`<attachmentRequest xmlns="http://example.com/service.xsd"><name>reply</name></attachmentRequest>`+
		"</soap:Body></soap:Envelope>\r\n"+
		"--%[1]s\r\n"+
		"Content-Type: text/xml; charset=utf-8\r\n"+
		"Content-ID: <doc.xml>\r\n\r\n"+
		"<doc/>\r\n"+
		"--%[1]s--\r\n", boundary)

	reply := &AttachmentRequest{}
	env := &SOAPEnvelopeResponse{Body: SOAPBodyResponse{Body: reply}}
	require.NoError(t, newMmaDecoder(strings.NewReader(body), boundary).Decode(env))

	assert.Equal(t, "reply", reply.Name)
	assert.Equal(t, []MIMEMultipartAttachment{{Name: "doc.xml", Data: []byte("<doc/>")}}, env.Attachments)
}
