package soap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"net/textproto"
)

const mmaContentType string = `multipart/related; start="<soaprequest@gowsdl.lib>"; type="text/xml"; boundary="%s"`

type mmaEncoder struct {
	multipartEncoder
	attachments []MIMEMultipartAttachment
}

type mmaDecoder struct {
	multipartDecoder
}

func newMmaEncoder(w io.Writer, attachments []MIMEMultipartAttachment) *mmaEncoder {
	return &mmaEncoder{
		multipartEncoder: newMultipartEncoder(w),
		attachments:      attachments,
	}
}

func newMmaDecoder(r io.Reader, boundary string) *mmaDecoder {
	return &mmaDecoder{newMultipartDecoder(r, boundary)}
}

// ContentType returns the Content-Type header of the encoded request.
func (e *mmaEncoder) ContentType() string {
	return fmt.Sprintf(mmaContentType, e.Boundary())
}

func (e *mmaEncoder) Encode(v interface{}) error {
	var err error
	var soapPartWriter io.Writer

	// 1. write SOAP envelope part
	headers := make(textproto.MIMEHeader)
	headers.Set("Content-Type", `text/xml;charset=UTF-8`)
	headers.Set("Content-Transfer-Encoding", "8bit")
	headers.Set("Content-ID", "<soaprequest@gowsdl.lib>")
	if soapPartWriter, err = e.writer.CreatePart(headers); err != nil {
		return err
	}
	xmlEncoder := xml.NewEncoder(soapPartWriter)
	if err := xmlEncoder.Encode(v); err != nil {
		return err
	}

	// 2. write attachments parts
	for _, attachment := range e.attachments {
		attHeader := make(textproto.MIMEHeader)
		attHeader.Set("Content-Type", fmt.Sprintf("application/octet-stream; name=%s", attachment.Name))
		attHeader.Set("Content-Transfer-Encoding", "binary")
		attHeader.Set("Content-ID", fmt.Sprintf("<%s>", attachment.Name))
		attHeader.Set("Content-Disposition",
			fmt.Sprintf("attachment; name=\"%s\"; filename=\"%s\"", attachment.Name, attachment.Name))
		var attachmentPartWriter io.Writer
		attachmentPartWriter, err := e.writer.CreatePart(attHeader)
		if err != nil {
			return err
		}
		_, err = io.Copy(attachmentPartWriter, bytes.NewReader(attachment.Data))
		if err != nil {
			return err
		}
	}

	return nil
}

func getMmaHeader(contentType string) (string, error) {
	boundary, _, err := multipartParams(contentType)
	return boundary, err
}

func (d *mmaDecoder) Decode(v interface{}) error {
	soapEnvResp := v.(*SOAPEnvelopeResponse)
	attachments := make([]MIMEMultipartAttachment, 0)
	rootDecoded := false
	isRoot := func(contentType string) bool {
		mediaType, _, _ := mime.ParseMediaType(contentType)
		if rootDecoded || mediaType != "text/xml" {
			return false
		}
		rootDecoded = true
		return true
	}
	attach := func(contentID, _ string, content []byte) {
		attachments = append(attachments, MIMEMultipartAttachment{
			Name: contentID,
			Data: content,
		})
	}
	if err := d.decodeParts(v, isRoot, attach); err != nil {
		return err
	}
	if len(attachments) > 0 {
		soapEnvResp.Attachments = attachments
	}

	return nil
}
