package jsonedit

import (
	"encoding/json"
	"sort"
	"strings"
)

func renderValue(b *strings.Builder, value *Value, depth int, format Format) {
	if value == nil {
		b.WriteString("null")
		return
	}
	switch value.Kind {
	case KindObject:
		renderObject(b, value, depth, format)
	case KindArray:
		renderArray(b, value, depth, format)
	case KindString:
		b.WriteString(quoteString(value.String))
	case KindRaw:
		if value.RawJSON == "" {
			b.WriteString("null")
			return
		}
		b.WriteString(value.RawJSON)
	}
}

func renderObject(b *strings.Builder, value *Value, depth int, format Format) {
	if len(value.Object) == 0 {
		b.WriteString("{}")
		return
	}
	members := value.Object
	if format.SortKeys {
		members = append([]Member(nil), value.Object...)
		sort.SliceStable(members, func(i, j int) bool {
			return members[i].Key < members[j].Key
		})
	}
	b.WriteByte('{')
	for i, member := range members {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
		writeIndent(b, depth+1, format.Indent)
		b.WriteString(quoteString(member.Key))
		b.WriteString(": ")
		renderValue(b, member.Value, depth+1, format)
	}
	b.WriteByte('\n')
	writeIndent(b, depth, format.Indent)
	b.WriteByte('}')
}

func renderArray(b *strings.Builder, value *Value, depth int, format Format) {
	if len(value.Array) == 0 {
		b.WriteString("[]")
		return
	}
	b.WriteByte('[')
	for i, item := range value.Array {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
		writeIndent(b, depth+1, format.Indent)
		renderValue(b, item, depth+1, format)
	}
	b.WriteByte('\n')
	writeIndent(b, depth, format.Indent)
	b.WriteByte(']')
}

func quoteString(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(data)
}

func writeIndent(b *strings.Builder, depth int, indent string) {
	for range depth {
		b.WriteString(indent)
	}
}
