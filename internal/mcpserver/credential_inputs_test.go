package mcpserver

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

var credentialFieldMarkers = []string{"apikey", "api_key", "secret", "accesstoken", "credential"}

func TestMCPToolInputsContainNoCredentialFields(t *testing.T) {
	inputTypes := []reflect.Type{
		reflect.TypeFor[TranslationGenerateInput](),
	}
	for _, typ := range inputTypes {
		for i := range typ.NumField() {
			field := typ.Field(i)
			name := strings.ToLower(field.Name)
			if slices.ContainsFunc(credentialFieldMarkers, func(marker string) bool {
				return strings.Contains(name, marker)
			}) {
				t.Fatalf("%s contains credential-like field %s", typ.Name(), field.Name)
			}
		}
	}
}
