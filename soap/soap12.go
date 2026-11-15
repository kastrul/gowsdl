// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package soap

import (
	"context"
	"encoding/xml"
	"fmt"
	"mime"
)

// XmlNsSoap12Env is the SOAP 1.2 envelope namespace.
const XmlNsSoap12Env = "http://www.w3.org/2003/05/soap-envelope"

// WithSOAP12 is an Option that makes the client send SOAP 1.2 messages: the
// envelope is in the SOAP 1.2 namespace and the SOAP action travels as the
// action parameter of an application/soap+xml Content-Type instead of a
// SOAPAction header. It cannot be combined with WithMTOM or
// WithMIMEMultipartAttachments.
func WithSOAP12() Option {
	return func(o *options) {
		o.soap12 = true
	}
}

type soap12ContextKey struct{}

// ContextWithSOAP12 returns a copy of ctx that makes a call made with it send
// a SOAP 1.2 message, as WithSOAP12 does for every call of a client.
// Generated clients use it for port types bound only with SOAP 1.2.
func ContextWithSOAP12(ctx context.Context) context.Context {
	return context.WithValue(ctx, soap12ContextKey{}, true)
}

// usesSOAP12 reports whether a call made with ctx sends SOAP 1.2.
func (c *Client) usesSOAP12(ctx context.Context) bool {
	on, _ := ctx.Value(soap12ContextKey{}).(bool)
	return on || c.opts.soap12
}

// soap12ContentType returns the Content-Type of a SOAP 1.2 request with the
// given SOAP action (RFC 3902).
func soap12ContentType(action string) string {
	params := map[string]string{"charset": "utf-8"}
	if action != "" {
		params["action"] = action
	}
	return mime.FormatMediaType("application/soap+xml", params)
}

// UnmarshalXML implements xml.Unmarshaler. It accepts a SOAP 1.1 or SOAP 1.2
// envelope and rejects an element in any other namespace.
func (e *SOAPEnvelopeResponse) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if start.Name.Local != "Envelope" || (start.Name.Space != XmlNsSoapEnv && start.Name.Space != XmlNsSoap12Env) {
		return fmt.Errorf("soap: expected a SOAP 1.1 or 1.2 Envelope, got {%s}%s", start.Name.Space, start.Name.Local)
	}
	type envelope SOAPEnvelopeResponse
	start.Name.Space = XmlNsSoapEnv
	return d.DecodeElement((*envelope)(e), &start)
}

// soap12Fault is a SOAP 1.2 fault as it is received.
type soap12Fault struct {
	Code struct {
		Value string `xml:"http://www.w3.org/2003/05/soap-envelope Value"`
	} `xml:"http://www.w3.org/2003/05/soap-envelope Code"`
	Reason struct {
		Text []string `xml:"http://www.w3.org/2003/05/soap-envelope Text"`
	} `xml:"http://www.w3.org/2003/05/soap-envelope Reason"`
	Role   string     `xml:"http://www.w3.org/2003/05/soap-envelope Role"`
	Detail FaultError `xml:"http://www.w3.org/2003/05/soap-envelope Detail"`
}

// decodeSOAP12Fault decodes the SOAP 1.2 fault starting at start into f:
// Code from the code's value, String from the first reason text, Actor
// from the role and Detail from the detail element.
func decodeSOAP12Fault(d *xml.Decoder, start *xml.StartElement, f *SOAPFault) error {
	in := soap12Fault{Detail: f.Detail}
	if err := d.DecodeElement(&in, start); err != nil {
		return err
	}
	f.Code = in.Code.Value
	if len(in.Reason.Text) > 0 {
		f.String = in.Reason.Text[0]
	}
	f.Actor = in.Role
	return nil
}
