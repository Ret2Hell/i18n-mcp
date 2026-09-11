//go:build !amd64 && !arm64

package jsonutil

import (
	"encoding/json"
	"io"
)

// Marshal returns the JSON encoding of value using encoding/json.
func Marshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

// MarshalIndent returns the indented JSON encoding of value using encoding/json.
func MarshalIndent(value any, prefix string, indent string) ([]byte, error) {
	return json.MarshalIndent(value, prefix, indent)
}

// Unmarshal decodes data into value using encoding/json.
func Unmarshal(data []byte, value any) error {
	return json.Unmarshal(data, value)
}

// NewDecoder returns an encoding/json streaming decoder.
func NewDecoder(reader io.Reader) Decoder {
	return json.NewDecoder(reader)
}

// NewEncoder returns an encoding/json streaming encoder.
func NewEncoder(writer io.Writer) Encoder {
	return json.NewEncoder(writer)
}

// Valid reports whether data is a valid JSON encoding.
func Valid(data []byte) bool {
	return json.Valid(data)
}
