package soap

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// SOAPEncoder encodes a SOAP envelope onto a request body.
type SOAPEncoder interface {
	Encode(v any) error
	Flush() error
}

// SOAPDecoder decodes a SOAP envelope from a response body.
type SOAPDecoder interface {
	Decode(v any) error
}

// SOAPEnvelopeResponse is the envelope of a SOAP response, including any
// MIME multipart attachments that came with it.
type SOAPEnvelopeResponse struct {
	XMLName     xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Header      *SOAPHeaderResponse
	Body        SOAPBodyResponse
	Attachments []MIMEMultipartAttachment `xml:"attachments,omitempty"`
}

// SOAPEnvelope is the envelope of an outgoing SOAP request.
type SOAPEnvelope struct {
	XMLName xml.Name `xml:"soap:Envelope"`
	XmlNS   string   `xml:"xmlns:soap,attr"`
	// Additional namespace declarations
	XMLNSXSI string `xml:"xmlns:xsi,attr,omitempty"`
	XMLNSXSD string `xml:"xmlns:xsd,attr,omitempty"`

	Header *SOAPHeader
	Body   SOAPBody
}

// SOAPHeader holds the header elements of an outgoing SOAP request.
type SOAPHeader struct {
	XMLName xml.Name `xml:"soap:Header"`

	Headers []any
}

// SOAPHeaderResponse holds the header elements of a SOAP response.
type SOAPHeaderResponse struct {
	XMLName xml.Name `xml:"Header"`

	Headers []any
}

// SOAPBody is the body of an outgoing SOAP message. Body is marshaled directly
// inside soap:Body, followed by Fault when it is set.
type SOAPBody struct {
	XMLName xml.Name `xml:"soap:Body"`

	Body any

	// faultOccurred indicates whether the XML body included a fault;
	// we cannot simply store SOAPFault as a pointer to indicate this, since
	// fault is initialized to non-nil with user-provided detail type.
	faultOccurred bool
	Fault         *SOAPFault `xml:",omitempty"`
}

// MarshalXML implements custom XML marshaling for SOAPBody to avoid wrapper elements
func (b SOAPBody) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Start the soap:Body element
	if err := e.EncodeToken(start); err != nil {
		return fmt.Errorf("encoding soap:Body start: %w", err)
	}

	// Encode the body content directly without wrapper
	if b.Body != nil {
		if err := e.Encode(b.Body); err != nil {
			return fmt.Errorf("encoding soap:Body content: %w", err)
		}
	}

	// Encode fault if present
	if b.Fault != nil {
		if err := e.Encode(b.Fault); err != nil {
			return fmt.Errorf("encoding soap:Fault: %w", err)
		}
	}

	// End the soap:Body element
	if err := e.EncodeToken(xml.EndElement{Name: start.Name}); err != nil {
		return fmt.Errorf("encoding soap:Body end: %w", err)
	}

	return nil
}

// SOAPBodyResponse is the body of a SOAP response. A soap:Fault element is
// decoded into Fault instead of Body.
type SOAPBodyResponse struct {
	XMLName xml.Name `xml:"Body"`

	Body any `xml:",omitempty"`

	// faultOccurred indicates whether the XML body included a fault;
	// we cannot simply store SOAPFault as a pointer to indicate this, since
	// fault is initialized to non-nil with user-provided detail type.
	faultOccurred bool
	Fault         *SOAPFault `xml:",omitempty"`
}

// MIMEMultipartAttachment is a named binary part of a MIME multipart message.
type MIMEMultipartAttachment struct {
	Name string
	Data []byte
}

// UnmarshalXML implements xml.Unmarshaler on SOAPBodyResponse. It decodes a
// soap:Fault into Fault and any other content into Body.
func (b *SOAPBodyResponse) UnmarshalXML(d *xml.Decoder, _ xml.StartElement) error {
	if b.Body == nil {
		return xml.UnmarshalError("Body must be a pointer to a struct")
	}

	var (
		token    xml.Token
		err      error
		consumed bool
	)

Loop:
	for {
		if token, err = d.Token(); err != nil {
			return fmt.Errorf("reading soap:Body: %w", err)
		}

		if token == nil {
			break
		}

		switch se := token.(type) {
		case xml.StartElement:
			if consumed {
				return xml.UnmarshalError("Found multiple elements inside SOAP body; not wrapped-document/literal WS-I compliant")
			} else if se.Name.Space == "http://schemas.xmlsoap.org/soap/envelope/" && se.Name.Local == "Fault" {
				b.Body = nil

				b.faultOccurred = true
				err = d.DecodeElement(b.Fault, &se)
				if err != nil {
					return fmt.Errorf("decoding soap:Fault: %w", err)
				}

				consumed = true
			} else {
				if err = d.DecodeElement(b.Body, &se); err != nil {
					return fmt.Errorf("decoding soap:Body content: %w", err)
				}

				consumed = true
			}
		case xml.EndElement:
			break Loop
		}
	}

	return nil
}

// ErrorFromFault returns Fault if the body is marked as containing a fault, and nil otherwise.
func (b *SOAPBody) ErrorFromFault() error {
	if b.faultOccurred {
		return b.Fault
	}
	b.Fault = nil
	return nil
}

// ErrorFromFault returns the SOAP fault found while unmarshaling the body, or nil if there was none.
func (b *SOAPBodyResponse) ErrorFromFault() error {
	if b.faultOccurred {
		return b.Fault
	}
	b.Fault = nil
	return nil
}

// DetailContainer wraps the contents of a SOAP fault's detail element.
type DetailContainer struct {
	Detail any
}

// FaultError is implemented by types that a SOAP fault's detail element is
// decoded into. See Client.CallContextWithFaultDetail.
type FaultError interface {
	// ErrorString should return a short version of the detail as a string,
	// which will be used in place of <faultstring> for the error message.
	// Set "HasData()" to always return false if <faultstring> error
	// message is preferred.
	ErrorString() string
	// HasData indicates whether the composite fault contains any data.
	HasData() bool
}

// SOAPFault is a SOAP 1.1 fault. It is returned as the error of a call
// whose response contains a soap:Fault.
type SOAPFault struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault"`

	Code   string     `xml:"faultcode,omitempty"`
	String string     `xml:"faultstring,omitempty"`
	Actor  string     `xml:"faultactor,omitempty"`
	Detail FaultError `xml:"detail,omitempty"`

	cause error
}

// Error returns the fault detail's ErrorString when the detail has data,
// and faultstring otherwise.
func (f *SOAPFault) Error() string {
	if f.Detail != nil && f.Detail.HasData() {
		return f.Detail.ErrorString()
	}
	return f.String
}

// Unwrap returns the transport error behind a fault that the client created
// for a failed call, such as the "Client.Transport" fault in a Result. It
// returns nil for a fault received from the server.
func (f *SOAPFault) Unwrap() error {
	return f.cause
}

// HTTPError is returned whenever the HTTP request to the server fails
type HTTPError struct {
	// StatusCode is the status code returned in the HTTP response
	StatusCode int
	// ResponseBody contains the body returned in the HTTP response
	ResponseBody []byte
}

// Error returns the HTTP status code and response body.
func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP Status %d: %s", e.StatusCode, string(e.ResponseBody))
}

// Namespace URIs used in WS-Security and SOAP envelopes.
const (
	// WssNsWSSE is the WS-Security extension (wsse) namespace.
	WssNsWSSE string = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd"
	// WssNsWSU is the WS-Security utility (wsu) namespace.
	WssNsWSU string = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd"
	// WssNsType is the type URI of a plain-text UsernameToken password.
	WssNsType       string = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordText"
	mtomContentType string = `multipart/related; start-info="application/soap+xml"; type="application/xop+xml"; boundary="%s"`
	// XmlNsSoapEnv is the SOAP 1.1 envelope namespace.
	XmlNsSoapEnv string = "http://schemas.xmlsoap.org/soap/envelope/"
)

// WSSSecurityHeader is a WS-Security wsse:Security header. Create one with
// NewWSSSecurityHeader.
type WSSSecurityHeader struct {
	XMLName   xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ wsse:Security"`
	XmlNSWsse string   `xml:"xmlns:wsse,attr"`

	MustUnderstand string `xml:"mustUnderstand,attr,omitempty"`

	Token *WSSUsernameToken `xml:",omitempty"`
}

// WSSUsernameToken is a WS-Security wsse:UsernameToken element.
type WSSUsernameToken struct {
	XMLName   xml.Name `xml:"wsse:UsernameToken"`
	XmlNSWsu  string   `xml:"xmlns:wsu,attr"`
	XmlNSWsse string   `xml:"xmlns:wsse,attr"`

	Id string `xml:"wsu:Id,attr,omitempty"`

	Username *WSSUsername `xml:",omitempty"`
	Password *WSSPassword `xml:",omitempty"`
}

// WSSUsername is the wsse:Username element of a UsernameToken.
type WSSUsername struct {
	XMLName   xml.Name `xml:"wsse:Username"`
	XmlNSWsse string   `xml:"xmlns:wsse,attr"`

	Data string `xml:",chardata"`
}

// WSSPassword is the wsse:Password element of a UsernameToken.
type WSSPassword struct {
	XMLName   xml.Name `xml:"wsse:Password"`
	XmlNSWsse string   `xml:"xmlns:wsse,attr"`
	XmlNSType string   `xml:"Type,attr"`

	Data string `xml:",chardata"`
}

// NewWSSSecurityHeader creates WSSSecurityHeader instance
func NewWSSSecurityHeader(user, pass, tokenID, mustUnderstand string) *WSSSecurityHeader {
	hdr := &WSSSecurityHeader{XmlNSWsse: WssNsWSSE, MustUnderstand: mustUnderstand}
	hdr.Token = &WSSUsernameToken{XmlNSWsu: WssNsWSU, XmlNSWsse: WssNsWSSE, Id: tokenID}
	hdr.Token.Username = &WSSUsername{XmlNSWsse: WssNsWSSE, Data: user}
	hdr.Token.Password = &WSSPassword{XmlNSWsse: WssNsWSSE, XmlNSType: WssNsType, Data: pass}
	return hdr
}

type basicAuth struct {
	Login    string
	Password string
}

type options struct {
	tlsCfg           *tls.Config
	auth             *basicAuth
	timeout          time.Duration
	contimeout       time.Duration
	tlshshaketimeout time.Duration
	client           HTTPClient
	httpHeaders      map[string]string
	mtom             bool
	mma              bool
}

var defaultOptions = options{
	timeout:          30 * time.Second,
	contimeout:       90 * time.Second,
	tlshshaketimeout: 15 * time.Second,
}

// Option sets options such as credentials, tls, etc.
type Option func(*options)

// WithHTTPClient is an Option to set the HTTP client to use
// This cannot be used with WithTLSHandshakeTimeout, WithTLS,
// WithTimeout options
func WithHTTPClient(c HTTPClient) Option {
	return func(o *options) {
		o.client = c
	}
}

// WithTLSHandshakeTimeout is an Option to set default tls handshake timeout
// This option cannot be used with WithHTTPClient
func WithTLSHandshakeTimeout(t time.Duration) Option {
	return func(o *options) {
		o.tlshshaketimeout = t
	}
}

// WithRequestTimeout is an Option to set default end-end connection timeout
// This option cannot be used with WithHTTPClient
func WithRequestTimeout(t time.Duration) Option {
	return func(o *options) {
		o.contimeout = t
	}
}

// WithBasicAuth is an Option to set BasicAuth
func WithBasicAuth(login, password string) Option {
	return func(o *options) {
		o.auth = &basicAuth{Login: login, Password: password}
	}
}

// WithTLS is an Option to set tls config
// This option cannot be used with WithHTTPClient
func WithTLS(tls *tls.Config) Option {
	return func(o *options) {
		o.tlsCfg = tls
	}
}

// WithTimeout is an Option to set default HTTP dial timeout
func WithTimeout(t time.Duration) Option {
	return func(o *options) {
		o.timeout = t
	}
}

// WithHTTPHeaders is an Option to set global HTTP headers for all requests
func WithHTTPHeaders(headers map[string]string) Option {
	return func(o *options) {
		o.httpHeaders = headers
	}
}

// WithMTOM is an Option to set Message Transmission Optimization Mechanism
// MTOM encodes fields of type Binary using XOP.
func WithMTOM() Option {
	return func(o *options) {
		o.mtom = true
	}
}

// WithMIMEMultipartAttachments is an Option to set SOAP MIME Multipart attachment support.
// Use Client.AddMIMEMultipartAttachment to add attachments of type MIMEMultipartAttachment to your SOAP request.
func WithMIMEMultipartAttachments() Option {
	return func(o *options) {
		o.mma = true
	}
}

// Client is a SOAP client.
type Client struct {
	url         string
	opts        *options
	headers     []any
	attachments []MIMEMultipartAttachment
}

// HTTPClient is a client which can make HTTP requests
// An example implementation is net/http.Client
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewClient creates new SOAP client instance
func NewClient(url string, opt ...Option) *Client {
	opts := defaultOptions
	for _, o := range opt {
		o(&opts)
	}
	return &Client{
		url:  url,
		opts: &opts,
	}
}

// AddHeader adds envelope header
// For correct behavior, every header must contain a `XMLName` field.  Refer to #121 for details
func (c *Client) AddHeader(header any) {
	c.headers = append(c.headers, header)
}

// AddMIMEMultipartAttachment adds an attachment to the client that will be sent only if the
// WithMIMEMultipartAttachments option is used
func (c *Client) AddMIMEMultipartAttachment(attachment MIMEMultipartAttachment) {
	c.attachments = append(c.attachments, attachment)
}

// SetHeaders sets envelope headers, overwriting any existing headers.
// For correct behavior, every header must contain a `XMLName` field.  Refer to #121 for details
func (c *Client) SetHeaders(headers ...any) {
	c.headers = headers
}

// GetHttpClientHeaders returns the extra HTTP headers the client sends with
// every request, for example an authentication header.
func (c *Client) GetHttpClientHeaders() map[string]string {
	return c.opts.httpHeaders
}

// SetHttpClientHeaders replaces the extra HTTP headers the client sends with
// every request.
func (c *Client) SetHttpClientHeaders(headers map[string]string) {
	c.opts.httpHeaders = headers
}

// URL returns the SOAP service URL
func (c *Client) URL() string {
	return c.url
}

// CallContext performs HTTP POST request with a context.
// A nil response makes it a one-way call: an empty reply, such as HTTP 202
// Accepted, succeeds, and any reply content other than a fault is ignored.
func (c *Client) CallContext(ctx context.Context, soapAction string, request, response any) error {
	return c.call(ctx, soapAction, request, response, nil, nil)
}

// Call performs HTTP POST request.
// Note that if the server returns a status code >= 400, a HTTPError will be returned
func (c *Client) Call(soapAction string, request, response any) error {
	return c.call(context.Background(), soapAction, request, response, nil, nil)
}

// CallContextWithAttachmentsAndFaultDetail performs HTTP POST request.
// Note that if SOAP fault is returned, it will be stored in the error.
// On top the attachments array will be filled with attachments returned from the SOAP request.
func (c *Client) CallContextWithAttachmentsAndFaultDetail(ctx context.Context, soapAction string, request,
	response any, faultDetail FaultError, attachments *[]MIMEMultipartAttachment) error {
	return c.call(ctx, soapAction, request, response, faultDetail, attachments)
}

// CallContextWithFaultDetail performs HTTP POST request.
// Note that if SOAP fault is returned, it will be stored in the error.
func (c *Client) CallContextWithFaultDetail(ctx context.Context, soapAction string, request, response any, faultDetail FaultError) error {
	return c.call(ctx, soapAction, request, response, faultDetail, nil)
}

// CallWithFaultDetail performs HTTP POST request.
// Note that if SOAP fault is returned, it will be stored in the error.
// the passed in fault detail is expected to implement FaultError interface,
// which allows to condense the detail into a short error message.
func (c *Client) CallWithFaultDetail(soapAction string, request, response any, faultDetail FaultError) error {
	return c.call(context.Background(), soapAction, request, response, faultDetail, nil)
}

// oneWayResponse accepts and discards whatever element the SOAP body of a
// reply to a one-way call holds.
type oneWayResponse struct {
	Content []byte `xml:",innerxml"`
}

func (c *Client) call(ctx context.Context, soapAction string, request, response any, faultDetail FaultError,
	retAttachments *[]MIMEMultipartAttachment) error {
	var envelope any

	// Try to extract namespace from request to determine which envelope to use
	targetNamespace := ""
	if req, ok := request.(interface{ TargetNamespace() string }); ok {
		targetNamespace = req.TargetNamespace()
	}

	if targetNamespace != "" {
		// Use enhanced envelope with namespace declarations
		enhancedEnv := NewEnhancedSOAPEnvelope(targetNamespace)
		if len(c.headers) > 0 {
			enhancedEnv.Header = &SOAPHeader{Headers: c.headers}
		}
		enhancedEnv.Body.Body = request
		envelope = enhancedEnv
	} else {
		// Use standard envelope (backward compatibility)
		standardEnv := &SOAPEnvelope{XmlNS: XmlNsSoapEnv}
		if len(c.headers) > 0 {
			standardEnv.Header = &SOAPHeader{Headers: c.headers}
		}
		standardEnv.Body.Body = request
		envelope = standardEnv
	}

	buffer := new(bytes.Buffer)
	encoder, err := c.newRequestEncoder(buffer)
	if err != nil {
		return fmt.Errorf("soap: creating request encoder: %w", err)
	}

	if err := encoder.Encode(envelope); err != nil {
		return fmt.Errorf("soap: encoding request: %w", err)
	}

	if err := encoder.Flush(); err != nil {
		return fmt.Errorf("soap: encoding request: %w", err)
	}

	req, err := http.NewRequest("POST", c.url, buffer)
	if err != nil {
		return fmt.Errorf("soap: building request: %w", err)
	}
	if c.opts.auth != nil {
		req.SetBasicAuth(c.opts.auth.Login, c.opts.auth.Password)
	}

	req = req.WithContext(ctx)

	req.Header.Add("Content-Type", encoder.ContentType())
	req.Header.Add("SOAPAction", soapAction)
	req.Header.Set("User-Agent", "gowsdl/0.1")
	if c.opts.httpHeaders != nil {
		for k, v := range c.opts.httpHeaders {
			req.Header.Set(k, v)
		}
	}
	req.Close = true

	client := c.opts.client
	if client == nil {
		tr := &http.Transport{
			TLSClientConfig: c.opts.tlsCfg,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				d := net.Dialer{Timeout: c.opts.timeout}
				return d.DialContext(ctx, network, addr)
			},
			TLSHandshakeTimeout: c.opts.tlshshaketimeout,
		}
		client = &http.Client{Timeout: c.opts.contimeout, Transport: tr}
	}

	res, err := client.Do(req)
	if err != nil {
		// Returned as is: callers may type-assert on the transport error.
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode >= 400 && res.StatusCode != 500 {
		body, _ := io.ReadAll(res.Body)
		return &HTTPError{
			StatusCode:   res.StatusCode,
			ResponseBody: body,
		}
	}

	// A nil response is a one-way call: the reply carries nothing to decode,
	// but a fault in it must still be reported. An empty reply, often without
	// a Content-Type, succeeds unless its status says otherwise.
	if response == nil {
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(data)) == 0 {
			if res.StatusCode >= 300 {
				return &HTTPError{StatusCode: res.StatusCode, ResponseBody: data}
			}
			return nil
		}
		res.Body = io.NopCloser(bytes.NewReader(data))
		response = new(oneWayResponse)
	}

	// xml Decoder (used with and without MTOM) cannot handle namespace prefixes (yet),
	// so we have to use a namespace-less response envelope
	respEnvelope := new(SOAPEnvelopeResponse)
	respEnvelope.Body = SOAPBodyResponse{
		Body: response,
		Fault: &SOAPFault{
			Detail: faultDetail,
		},
	}

	newDecoder, err := c.newResponseDecoder(res.Header.Get("Content-Type"))
	if err != nil {
		return fmt.Errorf("soap: decoding response: %w", err)
	}

	// we need to store the body in case of an error
	// to return the right HTTPError/ResponseBody
	body := res.Body
	var cachedErrorBody []byte
	if res.StatusCode == 500 {
		cachedErrorBody, err = io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		body = io.NopCloser(bytes.NewReader(cachedErrorBody))
	}

	dec := newDecoder(body)

	if err := dec.Decode(respEnvelope); err != nil {
		// the response doesn't contain a Fault/SOAPBody, so we return the original body
		if res.StatusCode == 500 {
			return &HTTPError{
				StatusCode:   res.StatusCode,
				ResponseBody: cachedErrorBody,
			}
		}
		return fmt.Errorf("soap: decoding response: %w", err)
	}

	if respEnvelope.Attachments != nil && retAttachments != nil {
		*retAttachments = respEnvelope.Attachments
	}
	return respEnvelope.Body.ErrorFromFault()
}
