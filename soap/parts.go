package soap

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"
)

// MarshalParts writes each exported field of the struct v as its own element,
// with no element enclosing them, so that the parts of a SOAP header message
// become sibling header blocks. Each element is named by the field's xml tag,
// either "local" or "namespace local", or by the field name when the tag has no
// name. Fields tagged "-" and fields holding their zero value are skipped.
func MarshalParts(e *xml.Encoder, v any) error {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("soap: MarshalParts needs a struct, got %s", rv.Kind())
	}

	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}
		value := rv.Field(i)
		if value.IsZero() {
			continue
		}
		tag, _, _ := strings.Cut(field.Tag.Get("xml"), ",")
		if tag == "-" {
			continue
		}

		name := xml.Name{Local: field.Name}
		if tag != "" {
			if space, local, ok := strings.Cut(tag, " "); ok {
				name = xml.Name{Space: space, Local: local}
			} else {
				name.Local = tag
			}
		}
		if err := e.EncodeElement(value.Interface(), xml.StartElement{Name: name}); err != nil {
			return fmt.Errorf("soap: marshal part %s: %w", field.Name, err)
		}
	}
	return nil
}
