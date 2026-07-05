package report

import "encoding/json"

// RenderJSON renders report as indented JSON.
func RenderJSON(report Report) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}
