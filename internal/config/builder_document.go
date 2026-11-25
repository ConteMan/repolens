package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

// BuilderSettings contains the trusted external fields editable in the desktop app.
type BuilderSettings struct {
	Source BuilderSourceSettings `yaml:"source"`
	Output BuilderOutputSettings `yaml:"output"`
	Access BuilderAccessSettings `yaml:"access"`
}

type BuilderSourceSettings struct {
	Repo *string `yaml:"repo,omitempty"`
	Ref  *string `yaml:"ref,omitempty"`
}

type BuilderOutputSettings struct {
	Dir      *string `yaml:"dir,omitempty"`
	BasePath *string `yaml:"base_path,omitempty"`
}

type BuilderAccessSettings struct {
	NoIndex *bool `yaml:"noindex,omitempty"`
}

// BuilderDocument retains the YAML and revision of a selected external file.
type BuilderDocument struct {
	document *RepositoryDocument
	Settings BuilderSettings
	Shared   RepositorySettings
}

// LoadBuilderDocument reads an explicit, existing external configuration file.
func LoadBuilderDocument(path string) (*BuilderDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read external config %q: %w", path, err)
	}
	document, err := parseRepositoryDocument(path, revisionFor(data), data)
	if err != nil {
		return nil, err
	}
	return builderFromDocument(document, data)
}

func builderFromDocument(document *RepositoryDocument, data []byte) (*BuilderDocument, error) {
	builder := &BuilderDocument{document: document, Shared: document.Settings}
	if err := yaml.Unmarshal(data, &builder.Settings); err != nil {
		return nil, fmt.Errorf("parse external config %q: %w", document.Path, err)
	}
	return builder, nil
}

func (b *BuilderDocument) Clone() (*BuilderDocument, error) {
	document, err := b.document.Clone()
	if err != nil {
		return nil, err
	}
	data, err := document.YAML()
	if err != nil {
		return nil, err
	}
	return builderFromDocument(document, data)
}

func (b *BuilderDocument) Replace(settings BuilderSettings) error {
	for _, section := range []struct {
		key    string
		values []settingValue
	}{
		{"source", []settingValue{{key: "repo", value: settings.Source.Repo}, {key: "ref", value: settings.Source.Ref}}},
		{"output", []settingValue{{key: "dir", value: settings.Output.Dir}, {key: "base_path", value: settings.Output.BasePath}}},
		{"access", []settingValue{{key: "noindex", value: settings.Access.NoIndex}}},
	} {
		if err := clearUnsetSettingsSection(b.document.root, section.key, section.values); err != nil {
			return err
		}
		if err := applySettingsSection(b.document.root, section.key, section.values); err != nil {
			return err
		}
	}
	data, err := b.document.YAML()
	if err != nil {
		return err
	}
	next, err := builderFromDocument(b.document, data)
	if err != nil {
		return err
	}
	b.Settings = next.Settings
	b.Shared = next.Shared
	return nil
}

func (b *BuilderDocument) ReplaceShared(settings RepositorySettings) error {
	if err := b.document.ReplaceSiteSettings(settings); err != nil {
		return err
	}
	b.Shared = b.document.Settings
	return nil
}

func (b *BuilderDocument) CheckRevision() error {
	return b.document.CheckRevision()
}

func (b *BuilderDocument) Write() error {
	return b.document.Write()
}

func (b *BuilderDocument) Preview() (string, error) {
	trusted, err := yaml.Marshal(b.Settings)
	if err != nil {
		return "", err
	}
	raw, err := b.document.YAML()
	if err != nil {
		return "", err
	}
	shared, err := RepositorySettingsPreview(raw, "site", "view", "theme", "render", "glossary", "agent", "ignore", "rules")
	return string(trusted) + shared, err
}
