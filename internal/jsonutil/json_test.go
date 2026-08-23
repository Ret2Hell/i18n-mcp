package jsonutil

import (
	"bytes"
	stdjson "encoding/json"
	"reflect"
	"strings"
	"testing"
)

type compatibilityFixture struct {
	Name    string            `json:"name"`
	Labels  map[string]string `json:"labels"`
	Omitted []string          `json:"omitted,omitzero"`
	HTML    string            `json:"html"`
}

func TestMarshalCompatibility(t *testing.T) {
	fixture := compatibilityFixture{
		Name:   "fixture",
		Labels: map[string]string{"z": "last", "a": "first"},
		HTML:   "<strong>safe & deterministic</strong>",
	}
	tests := []struct {
		name string
		std  func(any) ([]byte, error)
		got  func(any) ([]byte, error)
	}{
		{
			name: "compact",
			std:  stdjson.Marshal,
			got:  Marshal,
		},
		{
			name: "indented",
			std: func(value any) ([]byte, error) {
				return stdjson.MarshalIndent(value, "", "  ")
			},
			got: func(value any) ([]byte, error) {
				return MarshalIndent(value, "", "  ")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want, err := test.std(fixture)
			if err != nil {
				t.Fatalf("encoding/json: %v", err)
			}
			got, err := test.got(fixture)
			if err != nil {
				t.Fatalf("jsonutil: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("encoding mismatch\nwant: %s\n got: %s", want, got)
			}
		})
	}
}

func TestUnmarshalCompatibility(t *testing.T) {
	data := []byte(`{"name":"fixture","labels":{"a":"first","z":"last"},"html":"<b>text</b>"}`)
	var want, got compatibilityFixture
	if err := stdjson.Unmarshal(data, &want); err != nil {
		t.Fatalf("encoding/json: %v", err)
	}
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("jsonutil: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decoded value mismatch\nwant: %#v\n got: %#v", want, got)
	}
}

func TestDecoderOptions(t *testing.T) {
	decoder := NewDecoder(strings.NewReader(`{"number":9007199254740993}`))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode: %v", err)
	}
	number, ok := value["number"].(stdjson.Number)
	if !ok || number.String() != "9007199254740993" {
		t.Fatalf("number = %#v, want an exact json.Number", value["number"])
	}
}

func TestEncoderIndent(t *testing.T) {
	fixture := compatibilityFixture{Name: "fixture", Labels: map[string]string{"a": "first"}}
	var want, got bytes.Buffer
	stdEncoder := stdjson.NewEncoder(&want)
	stdEncoder.SetIndent("", "  ")
	if err := stdEncoder.Encode(fixture); err != nil {
		t.Fatalf("encoding/json: %v", err)
	}
	encoder := NewEncoder(&got)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(fixture); err != nil {
		t.Fatalf("jsonutil: %v", err)
	}
	if got.String() != want.String() {
		t.Fatalf("encoding mismatch\nwant: %s\n got: %s", want.String(), got.String())
	}
}
