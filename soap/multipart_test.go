package soap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultipartParams(t *testing.T) {
	boundary, params, err := multipartParams(`multipart/related; type="text/xml"; boundary="abc"`)
	require.NoError(t, err)
	assert.Equal(t, "abc", boundary)
	assert.Equal(t, "text/xml", params["type"])

	boundary, params, err = multipartParams(`text/xml; charset=utf-8`)
	require.NoError(t, err)
	assert.Empty(t, boundary)
	assert.Nil(t, params)

	_, _, err = multipartParams(`multipart/related; type="text/xml"`)
	assert.EqualError(t, err, "invalid multipart boundary: ")

	_, _, err = multipartParams(`;;;`)
	assert.Error(t, err)
}

func TestNewRequestEncoder_SelectsModeAndContentType(t *testing.T) {
	plain, err := NewClient("http://example.com").newRequestEncoder(new(bytes.Buffer))
	require.NoError(t, err)
	assert.IsType(t, xmlRequestEncoder{}, plain)
	assert.Equal(t, `text/xml; charset="utf-8"`, plain.ContentType())

	mtom, err := NewClient("http://example.com", WithMTOM()).newRequestEncoder(new(bytes.Buffer))
	require.NoError(t, err)
	require.IsType(t, &mtomEncoder{}, mtom)
	assert.Equal(t, fmt.Sprintf(mtomContentType, mtom.(*mtomEncoder).Boundary()), mtom.ContentType())

	mma, err := NewClient("http://example.com", WithMIMEMultipartAttachments()).newRequestEncoder(new(bytes.Buffer))
	require.NoError(t, err)
	require.IsType(t, &mmaEncoder{}, mma)
	assert.Equal(t, fmt.Sprintf(mmaContentType, mma.(*mmaEncoder).Boundary()), mma.ContentType())

	_, err = NewClient("http://example.com", WithMTOM(), WithMIMEMultipartAttachments()).newRequestEncoder(new(bytes.Buffer))
	assert.EqualError(t, err, "cannot use MTOM (XOP) and MMA (MIME Multipart Attachments) option at the same time")
}

func TestNewResponseDecoder_SelectsModeFromContentType(t *testing.T) {
	const mtomCT = `multipart/related; type="application/xop+xml"; start-info="text/xml"; boundary="b"`
	const mmaCT = `multipart/related; type="text/xml"; boundary="b"`
	plainClient := NewClient("http://example.com")
	mmaClient := NewClient("http://example.com", WithMIMEMultipartAttachments())

	decoderFor := func(c *Client, contentType string) SOAPDecoder {
		newDecoder, err := c.newResponseDecoder(contentType)
		require.NoError(t, err)
		return newDecoder(strings.NewReader(""))
	}

	assert.IsType(t, &xml.Decoder{}, decoderFor(plainClient, "text/xml"))
	assert.IsType(t, &mtomDecoder{}, decoderFor(plainClient, mtomCT), "MTOM responses are recognised without WithMTOM")
	assert.IsType(t, &xml.Decoder{}, decoderFor(plainClient, mmaCT), "MMA responses are only recognised with the MMA option")
	assert.IsType(t, &mmaDecoder{}, decoderFor(mmaClient, mmaCT))
	assert.IsType(t, &mtomDecoder{}, decoderFor(mmaClient, mtomCT))

	_, err := plainClient.newResponseDecoder(`multipart/related; type="application/xop+xml"`)
	assert.Error(t, err)
}

func TestMultipartDecoder_DecodeParts(t *testing.T) {
	body := "--b\r\nContent-Type: text/xml\r\n\r\n<root><name>n</name></root>\r\n" +
		"--b\r\nContent-Type: image/png\r\nContent-Id: <part1>\r\n\r\nDATA\r\n--b--\r\n"

	type attached struct{ id, contentType, content string }
	var got []attached
	var root struct {
		Name string `xml:"name"`
	}
	d := newMultipartDecoder(strings.NewReader(body), "b")
	err := d.decodeParts(&root,
		func(contentType string) bool { return contentType == "text/xml" },
		func(id, contentType string, content []byte) {
			got = append(got, attached{id, contentType, string(content)})
		})

	require.NoError(t, err)
	assert.Equal(t, "n", root.Name)
	assert.Equal(t, []attached{{"part1", "image/png", "DATA"}}, got)

	d = newMultipartDecoder(strings.NewReader("--b\r\nContent-Type: image/png\r\n\r\nDATA\r\n--b--\r\n"), "b")
	err = d.decodeParts(&root, func(string) bool { return false }, func(string, string, []byte) {})
	assert.EqualError(t, err, "invalid multipart content ID")
}
