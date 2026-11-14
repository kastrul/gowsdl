package soap

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"strings"
)

// requestEncoder is a SOAPEncoder that knows the Content-Type header of the
// request it encodes.
type requestEncoder interface {
	SOAPEncoder
	ContentType() string
}

// xmlRequestEncoder encodes a plain, non-multipart SOAP request.
type xmlRequestEncoder struct {
	*xml.Encoder
}

func (xmlRequestEncoder) ContentType() string {
	return `text/xml; charset="utf-8"`
}

// multipartEncoder is the part of a multipart/related request encoder that
// MTOM and MMA share.
type multipartEncoder struct {
	writer *multipart.Writer
}

func newMultipartEncoder(w io.Writer) multipartEncoder {
	return multipartEncoder{writer: multipart.NewWriter(w)}
}

func (e *multipartEncoder) Flush() error {
	return e.writer.Close()
}

func (e *multipartEncoder) Boundary() string {
	return e.writer.Boundary()
}

// multipartDecoder is the part of a multipart/related response decoder that
// MTOM and MMA share.
type multipartDecoder struct {
	reader *multipart.Reader
}

func newMultipartDecoder(r io.Reader, boundary string) multipartDecoder {
	return multipartDecoder{reader: multipart.NewReader(r, boundary)}
}

// decodeParts reads every part of the message. A part for which isRoot
// returns true, given its Content-Type header, is XML-decoded into v. Every
// other part must carry a Content-ID and is passed to attach with that ID
// (without angle brackets), its Content-Type header and its content.
func (d *multipartDecoder) decodeParts(v any, isRoot func(contentType string) bool, attach func(contentID, contentType string, content []byte)) error {
	for {
		p, err := d.reader.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		contentType := p.Header.Get("Content-Type")
		if isRoot(contentType) {
			if err := NewDecoder(p).Decode(v); err != nil {
				return err
			}
			continue
		}

		contentID := p.Header.Get("Content-Id")
		if contentID == "" {
			return errors.New("invalid multipart content ID")
		}
		content, err := io.ReadAll(p)
		if err != nil {
			return err
		}
		attach(strings.Trim(contentID, "<>"), contentType, content)
	}
}

// multipartParams parses a Content-Type header. For a multipart media type it
// returns the boundary, which must be present, and all parameters; for any
// other media type it returns an empty boundary.
func multipartParams(contentType string) (boundary string, params map[string]string, err error) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", nil, err
	}

	if !strings.HasPrefix(mediaType, "multipart/") {
		return "", nil, nil
	}

	boundary, ok := params["boundary"]
	if !ok || boundary == "" {
		return "", nil, fmt.Errorf("invalid multipart boundary: %s", boundary)
	}
	return boundary, params, nil
}

// newRequestEncoder selects the request encoding once, from the client's
// MTOM and MMA options.
func (s *Client) newRequestEncoder(w io.Writer) (requestEncoder, error) {
	switch {
	case s.opts.mtom && s.opts.mma:
		return nil, fmt.Errorf("cannot use MTOM (XOP) and MMA (MIME Multipart Attachments) option at the same time")
	case s.opts.mtom:
		return newMtomEncoder(w), nil
	case s.opts.mma:
		return newMmaEncoder(w, s.attachments), nil
	default:
		return xmlRequestEncoder{xml.NewEncoder(w)}, nil
	}
}

// newResponseDecoder selects the response decoding from the response's
// Content-Type header. An MTOM response is recognised whatever the client's
// options; an MMA response only when the client uses MMA.
func (s *Client) newResponseDecoder(contentType string) (func(io.Reader) SOAPDecoder, error) {
	mtomBoundary, err := getMtomHeader(contentType)
	if err != nil {
		return nil, err
	}

	var mmaBoundary string
	if s.opts.mma {
		mmaBoundary, err = getMmaHeader(contentType)
		if err != nil {
			return nil, err
		}
	}

	switch {
	case mtomBoundary != "":
		return func(r io.Reader) SOAPDecoder { return newMtomDecoder(r, mtomBoundary) }, nil
	case mmaBoundary != "":
		return func(r io.Reader) SOAPDecoder { return newMmaDecoder(r, mmaBoundary) }, nil
	default:
		return func(r io.Reader) SOAPDecoder { return NewDecoder(r) }, nil
	}
}
