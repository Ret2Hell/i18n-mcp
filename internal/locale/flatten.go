package locale

import (
	"cmp"
	"maps"
	"slices"
	"strings"
)

// Flatten converts a parsed locale JSON value into translation units.
func Flatten(ref FileRef, value any) FlattenResult {
	var result FlattenResult
	flattenValue(ref, nil, value, &result)
	slices.SortFunc(result.Units, func(a, b Unit) int {
		return cmp.Or(
			cmp.Compare(a.Key, b.Key),
			cmp.Compare(a.FilePath, b.FilePath),
		)
	})
	return result
}

func flattenValue(ref FileRef, jsonPath []string, value any, result *FlattenResult) {
	switch v := value.(type) {
	case map[string]any:
		if len(jsonPath) > 0 && looksLikeRichObject(v) {
			result.Warnings = append(result.Warnings, newWarning(
				ref,
				"unsupported_rich_object",
				"rich message objects are not supported as translation units in the MVP",
				jsonPath,
			))
			return
		}

		keys := slices.Sorted(maps.Keys(v))
		for _, key := range keys {
			flattenValue(ref, append(slices.Clone(jsonPath), key), v[key], result)
		}
	case string:
		result.Units = append(result.Units, Unit{
			Locale:    ref.Locale,
			Namespace: ref.Namespace,
			Key:       strings.Join(jsonPath, "."),
			Value:     v,
			FilePath:  ref.Path,
			JSONPath:  slices.Clone(jsonPath),
		})
	case []any:
		result.Warnings = append(result.Warnings, newWarning(
			ref,
			"unsupported_array",
			"arrays are not supported as translation units in the MVP",
			jsonPath,
		))
	case nil:
		result.Warnings = append(result.Warnings, newWarning(
			ref,
			"unsupported_null",
			"null values are not supported as translation units in the MVP",
			jsonPath,
		))
	default:
		result.Warnings = append(result.Warnings, newWarning(
			ref,
			"unsupported_non_string",
			"non-string leaves are not supported as translation units in the MVP",
			jsonPath,
		))
	}
}

func newWarning(ref FileRef, code string, message string, jsonPath []string) Warning {
	return Warning{
		Code:      code,
		Message:   message,
		Locale:    ref.Locale,
		Namespace: ref.Namespace,
		FilePath:  ref.Path,
		Key:       strings.Join(jsonPath, "."),
		JSONPath:  slices.Clone(jsonPath),
	}
}

func looksLikeRichObject(value map[string]any) bool {
	if _, ok := value["defaultMessage"]; ok {
		return true
	}
	if _, ok := value["message"]; ok {
		if _, hasDescription := value["description"]; hasDescription {
			return true
		}
	}
	if _, ok := value["values"]; ok {
		if _, hasText := value["text"]; hasText {
			return true
		}
	}
	return false
}
