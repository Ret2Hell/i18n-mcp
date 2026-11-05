package report

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// RenderJSON renders report as indented JSON.
func RenderJSON(report Report) (string, error) {
	data, err := json.Marshal(report, jsontext.WithIndent("  "), json.Deterministic(true), jsontext.EscapeForHTML(true), jsontext.EscapeForJS(true))
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}
