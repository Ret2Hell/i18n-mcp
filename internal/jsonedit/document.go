package jsonedit

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Document is an ordered JSON document plus the formatting preferences used to render it.
type Document struct {
	Root   *Value
	Format Format
}

// Kind identifies the JSON value representation stored in Value.
type Kind int

const (
	KindObject Kind = iota
	KindArray
	KindString
	KindRaw
)

var ErrPathExists = errors.New("json path already exists")

// Value represents an ordered JSON value. Non-string scalar values are kept as raw JSON.
type Value struct {
	Kind    Kind
	Object  []Member
	Array   []*Value
	String  string
	RawJSON string
}

// Member is an object member in source order.
type Member struct {
	Key   string
	Value *Value
}

// Format describes JSON rendering preferences.
type Format struct {
	Indent          string
	TrailingNewline bool
	SortKeys        bool
}

// RenderOptions describes optional render overrides.
type RenderOptions struct {
	SortKeys bool
}

// Parse parses data into an ordered JSON document and detects its formatting.
func Parse(data []byte, fallbackIndent int, sortKeys bool) (*Document, error) {
	format := DetectFormat(data, fallbackIndent, sortKeys)
	value, err := parseOrderedValue(data)
	if err != nil {
		return nil, err
	}
	return &Document{Root: value, Format: format}, nil
}

// NewObject creates an empty object document with fallback formatting.
func NewObject(fallbackIndent int, sortKeys bool) *Document {
	indent := fallbackIndent
	if indent <= 0 {
		indent = 2
	}
	return &Document{
		Root:   &Value{Kind: KindObject},
		Format: Format{Indent: strings.Repeat(" ", indent), TrailingNewline: true, SortKeys: sortKeys},
	}
}

// SetString sets a nested string value, preserving existing object member order.
func (d *Document) SetString(path []string, text string) error {
	if len(path) == 0 {
		return fmt.Errorf("empty JSON path")
	}
	if d.Root == nil || d.Root.Kind != KindObject {
		return fmt.Errorf("document root is not an object")
	}
	return setString(d.Root, path, text)
}

// String returns the string value at a nested path.
func (d *Document) String(path []string) (string, bool, error) {
	value, ok, err := valueAt(d.Root, path)
	if err != nil || !ok {
		return "", false, err
	}
	if value.Kind != KindString {
		return "", false, nil
	}
	return value.String, true, nil
}

// Exists reports whether any JSON value exists at a nested path.
func (d *Document) Exists(path []string) (bool, error) {
	_, ok, err := valueAt(d.Root, path)
	return ok, err
}

// RenameString moves a string value from one nested path to another.
func (d *Document) RenameString(from []string, to []string, overwrite bool) (bool, error) {
	if samePath(from, to) {
		return false, nil
	}
	if pathPrefix(from, to) || pathPrefix(to, from) {
		return false, fmt.Errorf("cannot rename between ancestor and descendant paths")
	}
	oldValue, ok, err := d.String(from)
	if err != nil || !ok {
		return false, err
	}
	if exists, err := d.Exists(to); err != nil {
		return false, err
	} else if exists && !overwrite {
		return false, ErrPathExists
	}
	if err := d.SetString(to, oldValue); err != nil {
		return false, err
	}
	deleted, err := d.Delete(from)
	if err != nil {
		return false, err
	}
	return deleted, nil
}

// Delete removes a nested value and prunes empty parent objects created by deletion.
func (d *Document) Delete(path []string) (bool, error) {
	if len(path) == 0 {
		return false, fmt.Errorf("empty JSON path")
	}
	if d.Root == nil || d.Root.Kind != KindObject {
		return false, fmt.Errorf("document root is not an object")
	}
	return deletePath(d.Root, path)
}

// Render serializes the document with its configured formatting.
func (d *Document) Render() ([]byte, error) {
	var b strings.Builder
	renderValue(&b, d.Root, 0, d.Format)
	if d.Format.TrailingNewline {
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}

func valueAt(value *Value, path []string) (*Value, bool, error) {
	if value == nil || len(path) == 0 {
		return value, value != nil, nil
	}
	if value.Kind != KindObject {
		return nil, false, nil
	}
	member := findMember(value, path[0])
	if member == nil {
		return nil, false, nil
	}
	return valueAt(member.Value, path[1:])
}

func samePath(a []string, b []string) bool {
	return slices.Equal(a, b)
}

func pathPrefix(prefix []string, path []string) bool {
	return len(prefix) < len(path) && slices.Equal(prefix, path[:len(prefix)])
}

func setString(value *Value, path []string, text string) error {
	if value.Kind != KindObject {
		return fmt.Errorf("path parent is not an object")
	}
	key := path[0]
	if len(path) == 1 {
		member := findMember(value, key)
		if member == nil {
			value.Object = append(value.Object, Member{Key: key, Value: &Value{Kind: KindString, String: text}})
			return nil
		}
		member.Value = &Value{Kind: KindString, String: text}
		return nil
	}
	member := findMember(value, key)
	if member == nil {
		child := &Value{Kind: KindObject}
		value.Object = append(value.Object, Member{Key: key, Value: child})
		return setString(child, path[1:], text)
	}
	if member.Value.Kind != KindObject {
		return fmt.Errorf("path segment %q is not an object", key)
	}
	return setString(member.Value, path[1:], text)
}

func deletePath(value *Value, path []string) (bool, error) {
	if value.Kind != KindObject {
		return false, fmt.Errorf("path parent is not an object")
	}
	key := path[0]
	for i := range value.Object {
		member := &value.Object[i]
		if member.Key != key {
			continue
		}
		if len(path) == 1 {
			value.Object = append(value.Object[:i], value.Object[i+1:]...)
			return true, nil
		}
		deleted, err := deletePath(member.Value, path[1:])
		if err != nil || !deleted {
			return deleted, err
		}
		if member.Value.Kind == KindObject && len(member.Value.Object) == 0 {
			value.Object = append(value.Object[:i], value.Object[i+1:]...)
		}
		return true, nil
	}
	return false, nil
}

func findMember(value *Value, key string) *Member {
	for i := range value.Object {
		if value.Object[i].Key == key {
			return &value.Object[i]
		}
	}
	return nil
}
