package config

import (
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
)

// RepositorySettingsPreview removes trusted-only fields from a YAML preview.
// When keys are supplied, only those top-level settings appear in the preview.
func RepositorySettingsPreview(data []byte, keys ...string) (string, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return "", nil
	}
	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		return "", err
	}
	delete(document, "source")
	delete(document, "output")
	delete(document, "access")
	if len(keys) != 0 {
		selected := make(map[string]any, len(keys))
		for _, key := range keys {
			if value, exists := document[key]; exists {
				selected[key] = value
			}
		}
		if len(selected) == 0 {
			return "", nil
		}
		document = selected
	}
	preview, err := yaml.Marshal(document)
	if err != nil {
		return "", err
	}
	return string(preview), nil
}

// RepositorySettingsDiff renders the differences between two settings previews.
func RepositorySettingsDiff(before, after, filePath string) string {
	if before == after {
		return ""
	}
	beforeLines := strings.Split(strings.TrimSuffix(before, "\n"), "\n")
	afterLines := strings.Split(strings.TrimSuffix(after, "\n"), "\n")
	if before == "" {
		beforeLines = nil
	}
	if after == "" {
		afterLines = nil
	}
	var diff strings.Builder
	fmt.Fprintf(&diff, "--- %s\n+++ %s\n", filePath, filePath)
	fmt.Fprintf(&diff, "@@ -%d,%d +%d,%d @@\n", hunkStart(len(beforeLines)), len(beforeLines), hunkStart(len(afterLines)), len(afterLines))
	for _, line := range beforeLines {
		fmt.Fprintf(&diff, "-%s\n", line)
	}
	for _, line := range afterLines {
		fmt.Fprintf(&diff, "+%s\n", line)
	}
	return diff.String()
}

func hunkStart(lineCount int) int {
	if lineCount == 0 {
		return 0
	}
	return 1
}
