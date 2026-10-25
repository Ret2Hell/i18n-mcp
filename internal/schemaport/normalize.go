// Package schemaport preserves JSON Schema semantics while avoiding multi-type
// declarations that some tool/function clients cannot map to their dialects.
package schemaport

import "github.com/google/jsonschema-go/jsonschema"

// Normalize rewrites type arrays to equivalent single-type anyOf branches.
// Required, defaults, nullability and unrelated constraints are not changed.
func Normalize(root *jsonschema.Schema) {
	seen := map[*jsonschema.Schema]bool{}
	var walk func(*jsonschema.Schema)
	walk = func(s *jsonschema.Schema) {
		if s == nil || seen[s] {
			return
		}
		seen[s] = true
		if len(s.Types) > 0 {
			if len(s.Types) == 1 {
				s.Type = s.Types[0]
			} else {
				branches := make([]*jsonschema.Schema, 0, len(s.Types))
				for _, kind := range s.Types {
					branches = append(branches, &jsonschema.Schema{Type: kind})
				}
				if len(s.AnyOf) == 0 {
					s.AnyOf = branches
				} else {
					s.AllOf = append(s.AllOf, &jsonschema.Schema{AnyOf: branches})
				}
			}
			s.Types = nil
		}
		for _, group := range []map[string]*jsonschema.Schema{s.Defs, s.Definitions, s.DependencySchemas, s.Properties, s.PatternProperties, s.DependentSchemas} {
			for _, child := range group {
				walk(child)
			}
		}
		for _, group := range [][]*jsonschema.Schema{s.AllOf, s.AnyOf, s.OneOf, s.PrefixItems, s.ItemsArray} {
			for _, child := range group {
				walk(child)
			}
		}
		for _, child := range []*jsonschema.Schema{s.Items, s.AdditionalItems, s.Contains, s.UnevaluatedItems, s.AdditionalProperties, s.PropertyNames, s.UnevaluatedProperties, s.Not, s.If, s.Then, s.Else, s.ContentSchema} {
			walk(child)
		}
	}
	walk(root)
}
