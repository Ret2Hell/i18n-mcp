package locale

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
)

// Preserve numeric literals at every depth without a float64 round-trip.
// The cloned raw value remains valid after the streaming decoder advances.
var exactNumbers = json.UnmarshalFromFunc(func(dec *jsontext.Decoder, out *any) error {
	if dec.PeekKind() != '0' {
		return errors.ErrUnsupported
	}
	raw, err := dec.ReadValue()
	if err == nil {
		*out = raw.Clone()
	}
	return err
})

// JSONDocument is a parsed locale JSON file with raw and decoded contents.
type JSONDocument struct {
	Path  string         `json:"path"`
	Bytes int            `json:"bytes"`
	Raw   jsontext.Value `json:"raw,omitzero"`
	Value any            `json:"-"`
}

// ParseError wraps JSON parsing errors with the source path.
type ParseError struct {
	Path string
	Err  error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse JSON locale file %s: %v", e.Path, e.Err)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// ParseJSONFile reads and parses a locale JSON file under guard.
func ParseJSONFile(ctx context.Context, guard *fsutil.Guard, relPath string) (JSONDocument, error) {
	if err := ctx.Err(); err != nil {
		return JSONDocument{}, err
	}
	resolved, err := guard.ResolveExisting(relPath)
	if err != nil {
		return JSONDocument{}, err
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return JSONDocument{}, fmt.Errorf("read locale file %s: %w", relPath, err)
	}

	if err := ctx.Err(); err != nil {
		return JSONDocument{}, err
	}
	var value any
	if err := json.Unmarshal(data, &value, json.WithUnmarshalers(exactNumbers)); err != nil {
		return JSONDocument{}, &ParseError{Path: relPath, Err: err}
	}

	if err := ctx.Err(); err != nil {
		return JSONDocument{}, err
	}
	return JSONDocument{
		Path:  relPath,
		Bytes: len(data),
		Raw:   jsontext.Value(bytes.Clone(data)),
		Value: value,
	}, nil
}
