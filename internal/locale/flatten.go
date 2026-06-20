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
	}
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
