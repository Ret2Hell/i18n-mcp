package jsonedit

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"slices"
	"strings"
)

func renderValue(b *strings.Builder, value *Value, depth int, format Format) error {
	if value == nil {
		b.WriteString("null")
		return nil
	}
	switch value.Kind {
	case KindObject:
		return renderObject(b, value, depth, format)
	case KindArray:
		return renderArray(b, value, depth, format)
	case KindString:
		text, err := quoteString(value.String)
		if err != nil {
			return err
		}
		b.WriteString(text)
	case KindRaw:
		if value.RawJSON == "" {
			b.WriteString("null")
			return nil
		}
		b.WriteString(value.RawJSON)
	default:
		return fmt.Errorf("invalid JSON value kind %d", value.Kind)
	}
	return nil
}

func renderObject(b *strings.Builder, value *Value, depth int, format Format) error {
	if len(value.Object) == 0 {
		b.WriteString("{}")
		return nil
	}
	members := value.Object
	if format.SortKeys {
		members = slices.Clone(value.Object)
		slices.SortStableFunc(members, func(a, b Member) int {
			return cmp.Compare(a.Key, b.Key)
		})
	}
	b.WriteByte('{')
	for i, member := range members {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
		writeIndent(b, depth+1, format.Indent)
		key, err := quoteString(member.Key)
		if err != nil {
			return err
		}
		b.WriteString(key)
		b.WriteString(": ")
		if err := renderValue(b, member.Value, depth+1, format); err != nil {
			return err
		}
	}
	b.WriteByte('\n')
	writeIndent(b, depth, format.Indent)
	b.WriteByte('}')
	return nil
}

func renderArray(b *strings.Builder, value *Value, depth int, format Format) error {
	if len(value.Array) == 0 {
		b.WriteString("[]")
		return nil
	}
	b.WriteByte('[')
	for i, item := range value.Array {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
		writeIndent(b, depth+1, format.Indent)
		if err := renderValue(b, item, depth+1, format); err != nil {
			return err
		}
	}
	b.WriteByte('\n')
	writeIndent(b, depth, format.Indent)
	b.WriteByte(']')
	return nil
}

func quoteString(value string) (string, error) {
	data, err := json.Marshal(value, jsontext.EscapeForHTML(true), jsontext.EscapeForJS(true))
	return string(data), err
}

func writeIndent(b *strings.Builder, depth int, indent string) {
	for range depth {
		b.WriteString(indent)
	}
}
