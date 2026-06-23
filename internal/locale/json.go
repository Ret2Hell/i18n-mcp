package locale

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
)

type JSONDocument struct {
	Path  string          `json:"path"`
	Bytes int             `json:"bytes"`
	Raw   json.RawMessage `json:"raw,omitzero"`
	Value any             `json:"-"`
}

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

func ParseJSONFile(ctx context.Context, guard *fsutil.Guard, relPath string) (JSONDocument, error) {
	_ = ctx
	resolved, err := guard.ResolveExisting(relPath)
	if err != nil {
		return JSONDocument{}, err
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return JSONDocument{}, fmt.Errorf("read locale file %s: %w", relPath, err)
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var value any
	if err := dec.Decode(&value); err != nil {
		return JSONDocument{}, &ParseError{Path: relPath, Err: err}
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return JSONDocument{}, &ParseError{Path: relPath, Err: err}
	}

	return JSONDocument{
		Path:  relPath,
		Bytes: len(data),
		Raw:   json.RawMessage(bytes.Clone(data)),
		Value: value,
	}, nil
}
