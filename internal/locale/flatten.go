package locale

import (
	"sort"
	"strings"
)

func Flatten(ref FileRef, value any) FlattenResult {
	var result FlattenResult
	flattenValue(ref, nil, value, &result)
	sort.Slice(result.Units, func(i, j int) bool {
		if result.Units[i].Key != result.Units[j].Key {
			return result.Units[i].Key < result.Units[j].Key
		}
		return result.Units[i].FilePath < result.Units[j].FilePath
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

		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			flattenValue(ref, appendJSONPath(jsonPath, key), v[key], result)
		}
	case string:
		result.Units = append(result.Units, Unit{
			Locale:    ref.Locale,
			Namespace: ref.Namespace,
			Key:       keyFromPath(jsonPath),
			Value:     v,
			FilePath:  ref.Path,
			JSONPath:  append([]string(nil), jsonPath...),
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
		Key:       keyFromPath(jsonPath),
		JSONPath:  append([]string(nil), jsonPath...),
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

func appendJSONPath(path []string, next string) []string {
	out := make([]string, len(path)+1)
	copy(out, path)
	out[len(path)] = next
	return out
}

func keyFromPath(path []string) string {
	return strings.Join(path, ".")
}
