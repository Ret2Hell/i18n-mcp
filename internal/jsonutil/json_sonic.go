//go:build amd64 || arm64

package jsonutil

import (
	"io"

	"github.com/bytedance/sonic"
)

// Marshal returns the JSON encoding of value using Sonic's standard-compatible configuration.
func Marshal(value any) ([]byte, error) {
	return sonic.ConfigStd.Marshal(value)
}

// MarshalIndent returns the indented JSON encoding of value using Sonic's standard-compatible configuration.
func MarshalIndent(value any, prefix string, indent string) ([]byte, error) {
	return sonic.ConfigStd.MarshalIndent(value, prefix, indent)
}

// Unmarshal decodes data into value using Sonic's standard-compatible configuration.
func Unmarshal(data []byte, value any) error {
	return sonic.ConfigStd.Unmarshal(data, value)
}

// NewDecoder returns a standard-compatible Sonic streaming decoder.
func NewDecoder(reader io.Reader) Decoder {
	return sonic.ConfigStd.NewDecoder(reader)
}

// NewEncoder returns a standard-compatible Sonic streaming encoder.
func NewEncoder(writer io.Writer) Encoder {
	return sonic.ConfigStd.NewEncoder(writer)
}

// Valid reports whether data is a valid JSON encoding.
func Valid(data []byte) bool {
	return sonic.ConfigStd.Valid(data)
}
