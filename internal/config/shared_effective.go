package config

import "github.com/goccy/go-yaml"

// ResolveSharedSettings applies the build configuration cascade to desktop drafts.
func ResolveSharedSettings(repository, external RepositorySettings) (RepositorySettings, error) {
	result := defaultConfig()
	for _, settings := range []RepositorySettings{repository, external} {
		raw, err := yaml.Marshal(settings)
		if err != nil {
			return RepositorySettings{}, err
		}
		var patch configPatch
		if err := yaml.Unmarshal(raw, &patch); err != nil {
			return RepositorySettings{}, err
		}
		applyPatch(&result, patch)
	}
	if err := normalizeAndValidateGlossary(&result); err != nil {
		return RepositorySettings{}, err
	}
	normalizeReadingView(&result.View)
	raw, err := yaml.Marshal(result)
	if err != nil {
		return RepositorySettings{}, err
	}
	var settings RepositorySettings
	err = yaml.Unmarshal(raw, &settings)
	return settings, err
}
