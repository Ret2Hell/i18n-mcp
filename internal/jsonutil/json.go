// Package jsonutil provides the project's standard-compatible JSON API.
// It uses Sonic on supported 64-bit architectures and encoding/json elsewhere.
package jsonutil

import (
	stdjson "encoding/json"
	"io"
)

// RawMessage is a raw encoded JSON value.
type RawMessage = stdjson.RawMessage

// Decoder describes the streaming decoder operations used by the project.
type Decoder interface {
	Decode(any) error
	Buffered() io.Reader
	DisallowUnknownFields()
	More() bool
	UseNumber()
}

// Encoder describes the streaming encoder operations used by the project.
type Encoder interface {
	Encode(any) error
	SetEscapeHTML(bool)
	SetIndent(string, string)
}
