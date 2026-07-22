package config

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

// ErrRevisionConflict reports that a repository configuration changed after it
// was loaded and before it could be written.
var ErrRevisionConflict = errors.New("repository configuration revision conflict")

// RepositorySettings is the subset of repository configuration editable by
// the UI. A nil leaf is absent from a patch and leaves the existing YAML value
// unchanged. A non-nil leaf replaces the corresponding YAML value.
type RepositorySettings struct {
	Site   RepositorySiteSettings        `yaml:"site" json:"site"`
	Ignore *[]string                     `yaml:"ignore" json:"ignore"`
	Render RepositoryFileOptionsSettings `yaml:"render" json:"render"`
	Rules  *[]RepositoryRuleSettings     `yaml:"rules" json:"rules"`
	Theme  RepositoryThemeSettings       `yaml:"theme" json:"theme"`
	View   RepositoryViewSettings        `yaml:"view" json:"view"`
	Agent  RepositoryAgentSettings       `yaml:"agent" json:"agent"`
}

// RepositorySiteSettings contains the editable site metadata fields.
type RepositorySiteSettings struct {
	Title    *string `yaml:"title" json:"title"`
	Language *string `yaml:"language" json:"language"`
	Home     *string `yaml:"home" json:"home"`
}

// RepositoryFileOptionsSettings contains editable global rendering options.
// Its fields are pointers so an omitted field can be distinguished from an
// explicit false, zero, or empty value.
type RepositoryFileOptionsSettings struct {
	Render      *bool                             `yaml:"render" json:"render"`
	Markdown    RepositoryMarkdownOptionsSettings `yaml:"markdown" json:"markdown"`
	HTML        RepositoryHTMLOptionsSettings     `yaml:"html" json:"html"`
	Code        RepositoryCodeOptionsSettings     `yaml:"code" json:"code"`
	MaxFileSize *RepositoryByteSize               `yaml:"max_file_size" json:"max_file_size"`
}

// RepositoryMarkdownOptionsSettings contains editable Markdown options.
type RepositoryMarkdownOptionsSettings struct {
	TOC              *bool `yaml:"toc" json:"toc"`
	TOCMinHeadings   *int  `yaml:"toc_min_headings" json:"toc_min_headings"`
	Anchors          *bool `yaml:"anchors" json:"anchors"`
	Mermaid          *bool `yaml:"mermaid" json:"mermaid"`
	Math             *bool `yaml:"math" json:"math"`
	FrontmatterTitle *bool `yaml:"frontmatter_title" json:"frontmatter_title"`
}

// RepositoryHTMLOptionsSettings contains editable HTML rendering options.
type RepositoryHTMLOptionsSettings struct {
	View *string `yaml:"view" json:"view"`
}

// RepositoryCodeOptionsSettings contains editable code rendering options.
type RepositoryCodeOptionsSettings struct {
	LineNumbers *bool   `yaml:"line_numbers" json:"line_numbers"`
	Theme       *string `yaml:"theme" json:"theme"`
}

// RepositoryRuleSettings contains one ordered repository rendering rule.
// It intentionally contains only the fields accepted by the configuration
// schema; unknown YAML fields remain in the document AST when rules are
// updated through RepositoryDocument.
type RepositoryRuleSettings struct {
	Match       *string                           `yaml:"match" json:"match"`
	Render      *bool                             `yaml:"render" json:"render"`
	Markdown    RepositoryMarkdownOptionsSettings `yaml:"markdown" json:"markdown"`
	HTML        RepositoryHTMLOptionsSettings     `yaml:"html" json:"html"`
	Code        RepositoryCodeOptionsSettings     `yaml:"code" json:"code"`
	MaxFileSize *RepositoryByteSize               `yaml:"max_file_size" json:"max_file_size"`
}

// RepositoryThemeSettings contains editable repository theme inputs.
type RepositoryThemeSettings struct {
	Vars      *map[string]string `yaml:"vars" json:"vars"`
	CSS       *string            `yaml:"css" json:"css"`
	Templates *string            `yaml:"templates" json:"templates"`
}

// RepositoryByteSize is a byte count accepted in either numeric form or the
// human-readable form used by .repolens.yml, such as "5MB".
type RepositoryByteSize int64

// UnmarshalYAML accepts the same byte-size notation as the merged config.
func (s *RepositoryByteSize) UnmarshalYAML(raw []byte) error {
	var value byteSize
	if err := value.UnmarshalYAML(raw); err != nil {
		return err
	}
	*s = RepositoryByteSize(value)
	return nil
}

// RepositoryViewSettings contains the editable browser view fields.
type RepositoryViewSettings struct {
	TreePosition    *string `yaml:"tree_position" json:"tree_position"`
	TreeExpandDepth *int    `yaml:"tree_expand_depth" json:"tree_expand_depth"`
	TOCPanel        *string `yaml:"toc_panel" json:"toc_panel"`
	Search          *bool   `yaml:"search" json:"search"`
}

// RepositoryAgentSettings contains the editable agent output fields.
type RepositoryAgentSettings struct {
	LLMSTxt   *bool                           `yaml:"llms_txt" json:"llms_txt"`
	LLMSFull  RepositoryAgentFullTextSettings `yaml:"llms_full" json:"llms_full"`
	IndexJSON *bool                           `yaml:"index_json" json:"index_json"`
}

// RepositoryAgentFullTextSettings contains editable llms-full options.
type RepositoryAgentFullTextSettings struct {
	Enabled *bool               `yaml:"enabled" json:"enabled"`
	MaxSize *RepositoryByteSize `yaml:"max_size" json:"max_size"`
}

// RepositoryDocument is the unmerged .repolens.yml document in a repository.
// Its Settings expose only the UI-controlled fields. The remaining YAML AST is
// retained so unknown fields and unsupported configuration sections survive a
// settings update.
type RepositoryDocument struct {
	Path     string
	Revision string
	Settings RepositorySettings

	root *ast.MappingNode
}

// RepositoryValidationIssue describes one invalid editable field. Path uses
// YAML-style notation, for example rules[1].match.
type RepositoryValidationIssue struct {
	Path    string
	Code    string
	Message string
}

// RepositoryValidationError groups the issues found while validating editable
// repository settings.
type RepositoryValidationError struct {
	Issues []RepositoryValidationIssue
}

func (e *RepositoryValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "invalid repository settings"
	}
	issue := e.Issues[0]
	return fmt.Sprintf("%s: %s", issue.Path, issue.Message)
}

// ValidateRepositorySettings checks only fields editable in the repository
// trust domain. Unknown YAML nodes and prohibited trust-domain sections are
// intentionally outside this validation surface.
func ValidateRepositorySettings(settings RepositorySettings) []RepositoryValidationIssue {
	var issues []RepositoryValidationIssue
	for index, pattern := range dereferenceStrings(settings.Ignore) {
		validatePattern(&issues, fmt.Sprintf("ignore[%d]", index), pattern)
	}
	validateFileOptionsSettings(&issues, "render", settings.Render)
	if settings.Rules != nil {
		for index, rule := range *settings.Rules {
			path := fmt.Sprintf("rules[%d]", index)
			if rule.Match == nil || strings.TrimSpace(*rule.Match) == "" {
				appendValidationIssue(&issues, path+".match", "required", "match is required")
			} else {
				validatePattern(&issues, path+".match", *rule.Match)
			}
			validateRuleSettings(&issues, path, rule)
		}
	}
	if settings.View.TreeExpandDepth != nil && *settings.View.TreeExpandDepth < 0 {
		appendValidationIssue(&issues, "view.tree_expand_depth", "invalid_value", "must be non-negative")
	}
	if settings.View.TreePosition != nil && *settings.View.TreePosition != "left" && *settings.View.TreePosition != "right" {
		appendValidationIssue(&issues, "view.tree_position", "invalid_value", "must be left or right")
	}
	if settings.View.TOCPanel != nil && *settings.View.TOCPanel != "floating" && *settings.View.TOCPanel != "inline" {
		appendValidationIssue(&issues, "view.toc_panel", "invalid_value", "must be floating or inline")
	}
	if settings.Agent.LLMSFull.MaxSize != nil && *settings.Agent.LLMSFull.MaxSize < 0 {
		appendValidationIssue(&issues, "agent.llms_full.max_size", "invalid_value", "must be non-negative")
	}
	return issues
}

// ValidateRepositoryDocument validates the editable settings of document.
func ValidateRepositoryDocument(document *RepositoryDocument) []RepositoryValidationIssue {
	if document == nil {
		return []RepositoryValidationIssue{{
			Path:    "",
			Code:    "invalid_document",
			Message: "repository document is not initialized",
		}}
	}
	return ValidateRepositorySettings(document.Settings)
}

func validateRepositorySettings(settings RepositorySettings) error {
	issues := ValidateRepositorySettings(settings)
	if len(issues) == 0 {
		return nil
	}
	return &RepositoryValidationError{Issues: issues}
}

func validateFileOptionsSettings(issues *[]RepositoryValidationIssue, path string, settings RepositoryFileOptionsSettings) {
	if settings.MaxFileSize != nil && *settings.MaxFileSize < 0 {
		appendValidationIssue(issues, path+".max_file_size", "invalid_value", "must be non-negative")
	}
	if settings.Markdown.TOCMinHeadings != nil && *settings.Markdown.TOCMinHeadings < 0 {
		appendValidationIssue(issues, path+".markdown.toc_min_headings", "invalid_value", "must be non-negative")
	}
	validateHTMLView(issues, path+".html.view", settings.HTML.View)
}

func validateRuleSettings(issues *[]RepositoryValidationIssue, path string, rule RepositoryRuleSettings) {
	validateFileOptionsSettings(issues, path, RepositoryFileOptionsSettings{
		Render:      rule.Render,
		Markdown:    rule.Markdown,
		HTML:        rule.HTML,
		Code:        rule.Code,
		MaxFileSize: rule.MaxFileSize,
	})
}

func validatePattern(issues *[]RepositoryValidationIssue, path, pattern string) {
	if strings.TrimSpace(pattern) == "" {
		appendValidationIssue(issues, path, "required", "pattern must not be empty")
		return
	}
	if !doublestar.ValidatePattern(pattern) {
		appendValidationIssue(issues, path, "invalid_pattern", "must be a valid doublestar pattern")
	}
}

func validateHTMLView(issues *[]RepositoryValidationIssue, path string, view *string) {
	if view == nil {
		return
	}
	switch *view {
	case "embed", "direct", "source":
		return
	default:
		appendValidationIssue(issues, path, "invalid_value", "must be one of embed, direct, or source")
	}
}

func appendValidationIssue(issues *[]RepositoryValidationIssue, path, code, message string) {
	*issues = append(*issues, RepositoryValidationIssue{Path: path, Code: code, Message: message})
}

func dereferenceStrings(values *[]string) []string {
	if values == nil {
		return nil
	}
	return *values
}

// LoadRepositoryDocument reads the repository-local .repolens.yml without
// merging defaults or trusted external configuration. A missing file is an
// empty document with the revision of empty content.
func LoadRepositoryDocument(repoRoot string) (*RepositoryDocument, error) {
	filePath := filepath.Join(repoRoot, repoConfigName)
	data, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read repository config %q: %w", filePath, err)
	}
	if os.IsNotExist(err) {
		data = nil
	}

	doc := &RepositoryDocument{
		Path:     filePath,
		Revision: revisionFor(data),
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		root, err := newMappingNode(nil)
		if err != nil {
			return nil, fmt.Errorf("create repository config document: %w", err)
		}
		doc.root = root
		return doc, nil
	}

	file, err := parser.ParseBytes(data, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse repository config %q: %w", filePath, err)
	}
	if len(file.Docs) != 1 || file.Docs[0].Body == nil {
		return nil, fmt.Errorf("parse repository config %q: expected one YAML document", filePath)
	}
	root, ok := file.Docs[0].Body.(*ast.MappingNode)
	if !ok {
		return nil, fmt.Errorf("parse repository config %q: top-level YAML value must be a mapping", filePath)
	}
	if err := yaml.Unmarshal(data, &doc.Settings); err != nil {
		return nil, fmt.Errorf("parse editable repository settings %q: %w", filePath, err)
	}
	doc.root = root
	return doc, nil
}

// Apply updates the editable repository settings. It leaves trusted-only,
// unknown, and omitted editable YAML nodes unchanged.
func (d *RepositoryDocument) Apply(settings RepositorySettings) error {
	if d == nil || d.root == nil {
		return errors.New("repository document is not initialized")
	}
	if err := validateRepositorySettings(settings); err != nil {
		return err
	}
	if err := applySettingsSection(d.root, "site", []settingValue{
		{key: "title", value: settings.Site.Title},
		{key: "language", value: settings.Site.Language},
		{key: "home", value: settings.Site.Home},
	}); err != nil {
		return err
	}
	if settings.Ignore != nil {
		if err := applyValues(d.root, []settingValue{{key: "ignore", value: settings.Ignore}}); err != nil {
			return err
		}
	}
	if err := applyFileOptionsSettings(d.root, "render", settings.Render); err != nil {
		return err
	}
	if err := applyRules(d.root, settings.Rules); err != nil {
		return err
	}
	if err := applySettingsSection(d.root, "theme", []settingValue{
		{key: "vars", value: settings.Theme.Vars},
		{key: "css", value: settings.Theme.CSS},
		{key: "templates", value: settings.Theme.Templates},
	}); err != nil {
		return err
	}
	if err := applySettingsSection(d.root, "view", []settingValue{
		{key: "tree_position", value: settings.View.TreePosition},
		{key: "tree_expand_depth", value: settings.View.TreeExpandDepth},
		{key: "toc_panel", value: settings.View.TOCPanel},
		{key: "search", value: settings.View.Search},
	}); err != nil {
		return err
	}
	if err := applySettingsSection(d.root, "agent", []settingValue{
		{key: "llms_txt", value: settings.Agent.LLMSTxt},
		{key: "index_json", value: settings.Agent.IndexJSON},
	}); err != nil {
		return err
	}
	if err := applyNestedSettingsSection(d.root, "agent", "llms_full", []settingValue{
		{key: "enabled", value: settings.Agent.LLMSFull.Enabled},
		{key: "max_size", value: settings.Agent.LLMSFull.MaxSize},
	}); err != nil {
		return err
	}
	return d.reloadSettings()
}

// Replace applies a complete UI form submission. Unlike Apply, nil editable
// leaves explicitly remove their YAML nodes so the effective configuration
// returns to its default value. Trusted-only sections remain untouched; unknown
// nodes retain the same best-effort preservation behavior as Apply.
func (d *RepositoryDocument) Replace(settings RepositorySettings) error {
	if d == nil || d.root == nil {
		return errors.New("repository document is not initialized")
	}
	if err := validateRepositorySettings(settings); err != nil {
		return err
	}
	if err := d.clearUnsetSettings(settings); err != nil {
		return err
	}
	if settings.Rules != nil {
		if err := replaceRules(d.root, settings.Rules); err != nil {
			return err
		}
		settings.Rules = nil
	}
	return d.Apply(settings)
}

func (d *RepositoryDocument) clearUnsetSettings(settings RepositorySettings) error {
	if err := clearUnsetSettingsSection(d.root, "site", []settingValue{
		{key: "title", value: settings.Site.Title}, {key: "language", value: settings.Site.Language}, {key: "home", value: settings.Site.Home},
	}); err != nil {
		return err
	}
	if settings.Ignore == nil {
		removeMappingValue(d.root, "ignore")
	}
	if err := clearUnsetFileOptionsSettings(d.root, "render", settings.Render); err != nil {
		return err
	}
	if settings.Rules == nil {
		removeMappingValue(d.root, "rules")
	}
	if err := clearUnsetSettingsSection(d.root, "theme", []settingValue{
		{key: "vars", value: settings.Theme.Vars}, {key: "css", value: settings.Theme.CSS}, {key: "templates", value: settings.Theme.Templates},
	}); err != nil {
		return err
	}
	if err := clearUnsetSettingsSection(d.root, "view", []settingValue{
		{key: "tree_position", value: settings.View.TreePosition}, {key: "tree_expand_depth", value: settings.View.TreeExpandDepth},
		{key: "toc_panel", value: settings.View.TOCPanel}, {key: "search", value: settings.View.Search},
	}); err != nil {
		return err
	}
	if err := clearUnsetSettingsSection(d.root, "agent", []settingValue{
		{key: "llms_txt", value: settings.Agent.LLMSTxt}, {key: "index_json", value: settings.Agent.IndexJSON},
	}); err != nil {
		return err
	}
	return clearUnsetNestedSettingsSection(d.root, "agent", "llms_full", []settingValue{
		{key: "enabled", value: settings.Agent.LLMSFull.Enabled}, {key: "max_size", value: settings.Agent.LLMSFull.MaxSize},
	})
}

func clearUnsetFileOptionsSettings(root *ast.MappingNode, section string, settings RepositoryFileOptionsSettings) error {
	if err := clearUnsetSettingsSection(root, section, []settingValue{
		{key: "render", value: settings.Render}, {key: "max_file_size", value: settings.MaxFileSize},
	}); err != nil {
		return err
	}
	if err := clearUnsetNestedSettingsSection(root, section, "markdown", []settingValue{
		{key: "toc", value: settings.Markdown.TOC}, {key: "toc_min_headings", value: settings.Markdown.TOCMinHeadings},
		{key: "anchors", value: settings.Markdown.Anchors}, {key: "mermaid", value: settings.Markdown.Mermaid},
		{key: "math", value: settings.Markdown.Math}, {key: "frontmatter_title", value: settings.Markdown.FrontmatterTitle},
	}); err != nil {
		return err
	}
	if err := clearUnsetNestedSettingsSection(root, section, "html", []settingValue{{key: "view", value: settings.HTML.View}}); err != nil {
		return err
	}
	return clearUnsetNestedSettingsSection(root, section, "code", []settingValue{
		{key: "line_numbers", value: settings.Code.LineNumbers}, {key: "theme", value: settings.Code.Theme},
	})
}

// AddRule appends a rendering rule while preserving the existing rule order.
func (d *RepositoryDocument) AddRule(rule RepositoryRuleSettings) error {
	if d == nil || d.root == nil {
		return errors.New("repository document is not initialized")
	}
	rules := cloneRepositoryRules(d.Settings.Rules)
	rules = append(rules, rule)
	return d.replaceRules(rules)
}

// RemoveRule removes the rendering rule at index.
func (d *RepositoryDocument) RemoveRule(index int) error {
	if d == nil || d.root == nil {
		return errors.New("repository document is not initialized")
	}
	rules := cloneRepositoryRules(d.Settings.Rules)
	if index < 0 || index >= len(rules) {
		return fmt.Errorf("rule index %d is out of range", index)
	}
	rules = append(rules[:index], rules[index+1:]...)
	return d.replaceRules(rules)
}

// MoveRule moves the rendering rule at from to its final position at to.
func (d *RepositoryDocument) MoveRule(from, to int) error {
	if d == nil || d.root == nil {
		return errors.New("repository document is not initialized")
	}
	rules := cloneRepositoryRules(d.Settings.Rules)
	if from < 0 || from >= len(rules) || to < 0 || to >= len(rules) {
		return fmt.Errorf("rule move %d to %d is out of range", from, to)
	}
	if from == to {
		return nil
	}
	rule := rules[from]
	if from < to {
		copy(rules[from:to], rules[from+1:to+1])
	} else {
		copy(rules[to+1:from+1], rules[to:from])
	}
	rules[to] = rule
	return d.replaceRules(rules)
}

func (d *RepositoryDocument) replaceRules(rules []RepositoryRuleSettings) error {
	settings := d.Settings
	settings.Rules = &rules
	if err := validateRepositorySettings(settings); err != nil {
		return err
	}
	if err := applyRules(d.root, &rules); err != nil {
		return err
	}
	return d.reloadSettings()
}

func (d *RepositoryDocument) reloadSettings() error {
	data, err := d.YAML()
	if err != nil {
		return err
	}
	var settings RepositorySettings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parse editable repository settings: %w", err)
	}
	d.Settings = settings
	return nil
}

// YAML returns the current document as YAML. It is intended for displaying a
// candidate write before Write is called.
func (d *RepositoryDocument) YAML() ([]byte, error) {
	if d == nil || d.root == nil {
		return nil, errors.New("repository document is not initialized")
	}
	return []byte(d.root.String() + "\n"), nil
}

// Write atomically commits the current document if the file revision has not
// changed since LoadRepositoryDocument. It updates Revision on success.
func (d *RepositoryDocument) Write() error {
	if d == nil {
		return errors.New("repository document is not initialized")
	}
	data, err := d.YAML()
	if err != nil {
		return err
	}
	current, err := os.ReadFile(d.Path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read repository config %q before write: %w", d.Path, err)
	}
	if os.IsNotExist(err) {
		current = nil
	}
	if revisionFor(current) != d.Revision {
		return fmt.Errorf("write repository config %q: %w", d.Path, ErrRevisionConflict)
	}

	perm := os.FileMode(0o644)
	if info, err := os.Stat(d.Path); err == nil {
		perm = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat repository config %q: %w", d.Path, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(d.Path), ".repolens.yml-*")
	if err != nil {
		return fmt.Errorf("create repository config temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("set repository config temporary file mode: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write repository config temporary file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync repository config temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close repository config temporary file: %w", err)
	}
	if err := os.Rename(tmpPath, d.Path); err != nil {
		return fmt.Errorf("replace repository config %q: %w", d.Path, err)
	}
	d.Revision = revisionFor(data)
	return nil
}

type settingValue struct {
	key   string
	value any
}

func applyFileOptionsSettings(root *ast.MappingNode, section string, settings RepositoryFileOptionsSettings) error {
	if !hasFileOptionsSettings(settings) {
		return nil
	}
	if err := applySettingsSection(root, section, []settingValue{
		{key: "render", value: settings.Render},
		{key: "max_file_size", value: settings.MaxFileSize},
	}); err != nil {
		return err
	}
	if err := applyNestedSettingsSection(root, section, "markdown", []settingValue{
		{key: "toc", value: settings.Markdown.TOC},
		{key: "toc_min_headings", value: settings.Markdown.TOCMinHeadings},
		{key: "anchors", value: settings.Markdown.Anchors},
		{key: "mermaid", value: settings.Markdown.Mermaid},
		{key: "math", value: settings.Markdown.Math},
		{key: "frontmatter_title", value: settings.Markdown.FrontmatterTitle},
	}); err != nil {
		return err
	}
	if err := applyNestedSettingsSection(root, section, "html", []settingValue{
		{key: "view", value: settings.HTML.View},
	}); err != nil {
		return err
	}
	return applyNestedSettingsSection(root, section, "code", []settingValue{
		{key: "line_numbers", value: settings.Code.LineNumbers},
		{key: "theme", value: settings.Code.Theme},
	})
}

func applyRules(root *ast.MappingNode, rules *[]RepositoryRuleSettings) error {
	return applyRulesWithMode(root, rules, false)
}

func replaceRules(root *ast.MappingNode, rules *[]RepositoryRuleSettings) error {
	return applyRulesWithMode(root, rules, true)
}

func applyRulesWithMode(root *ast.MappingNode, rules *[]RepositoryRuleSettings, replace bool) error {
	if rules == nil {
		return nil
	}

	item := findMappingValue(root, "rules")
	var sequence *ast.SequenceNode
	if item == nil {
		var err error
		sequence, err = newSequenceNode()
		if err != nil {
			return err
		}
		if err := appendMappingValue(root, "rules", sequence); err != nil {
			return err
		}
	} else {
		var ok bool
		sequence, ok = item.Value.(*ast.SequenceNode)
		if !ok {
			return errors.New("rules must be a YAML sequence")
		}
	}

	oldValues := sequence.Values
	oldComments := sequence.ValueHeadComments
	used := make([]bool, len(oldValues))
	oldIndexes := make([]int, len(*rules))
	for index := range oldIndexes {
		oldIndexes[index] = -1
	}

	// Bind matching rules before building the replacement sequence so an
	// inserted rule cannot consume an existing node and inherit its unknown
	// fields.
	for index, rule := range *rules {
		oldIndex := findRuleNode(oldValues, used, rule)
		if oldIndex >= 0 {
			used[oldIndex] = true
			oldIndexes[index] = oldIndex
		}
	}

	values := make([]ast.Node, 0, len(*rules))
	comments := make([]*ast.CommentGroupNode, 0, len(*rules))
	for index, rule := range *rules {
		oldIndex := oldIndexes[index]
		var node *ast.MappingNode
		if oldIndex >= 0 {
			var ok bool
			node, ok = oldValues[oldIndex].(*ast.MappingNode)
			if !ok {
				return fmt.Errorf("rules[%d] must be a YAML mapping", oldIndex)
			}
		} else {
			var err error
			node, err = newMappingNode(nil)
			if err != nil {
				return err
			}
		}
		if replace {
			if err := clearUnsetRuleSettings(node, rule); err != nil {
				return fmt.Errorf("rules[%d]: %w", index, err)
			}
		}
		if err := applyRuleSettings(node, rule); err != nil {
			return fmt.Errorf("rules[%d]: %w", index, err)
		}
		values = append(values, node)
		if oldIndex >= 0 && len(oldComments) == len(oldValues) {
			comments = append(comments, oldComments[oldIndex])
		}
	}
	sequence.Values = values
	if len(comments) == len(values) {
		sequence.ValueHeadComments = comments
	} else {
		sequence.ValueHeadComments = nil
	}
	return nil
}

func applyRuleSettings(node *ast.MappingNode, rule RepositoryRuleSettings) error {
	if err := applyValues(node, []settingValue{
		{key: "match", value: rule.Match},
		{key: "render", value: rule.Render},
		{key: "max_file_size", value: rule.MaxFileSize},
	}); err != nil {
		return err
	}
	if err := applyNestedValues(node, "markdown", []settingValue{
		{key: "toc", value: rule.Markdown.TOC},
		{key: "toc_min_headings", value: rule.Markdown.TOCMinHeadings},
		{key: "anchors", value: rule.Markdown.Anchors},
		{key: "mermaid", value: rule.Markdown.Mermaid},
		{key: "math", value: rule.Markdown.Math},
		{key: "frontmatter_title", value: rule.Markdown.FrontmatterTitle},
	}); err != nil {
		return err
	}
	if err := applyNestedValues(node, "html", []settingValue{{key: "view", value: rule.HTML.View}}); err != nil {
		return err
	}
	return applyNestedValues(node, "code", []settingValue{
		{key: "line_numbers", value: rule.Code.LineNumbers},
		{key: "theme", value: rule.Code.Theme},
	})
}

func clearUnsetRuleSettings(node *ast.MappingNode, rule RepositoryRuleSettings) error {
	clearUnsetValues(node, []settingValue{
		{key: "match", value: rule.Match}, {key: "render", value: rule.Render}, {key: "max_file_size", value: rule.MaxFileSize},
	})
	if err := clearUnsetNestedValues(node, "markdown", []settingValue{
		{key: "toc", value: rule.Markdown.TOC}, {key: "toc_min_headings", value: rule.Markdown.TOCMinHeadings},
		{key: "anchors", value: rule.Markdown.Anchors}, {key: "mermaid", value: rule.Markdown.Mermaid},
		{key: "math", value: rule.Markdown.Math}, {key: "frontmatter_title", value: rule.Markdown.FrontmatterTitle},
	}); err != nil {
		return err
	}
	if err := clearUnsetNestedValues(node, "html", []settingValue{{key: "view", value: rule.HTML.View}}); err != nil {
		return err
	}
	return clearUnsetNestedValues(node, "code", []settingValue{
		{key: "line_numbers", value: rule.Code.LineNumbers}, {key: "theme", value: rule.Code.Theme},
	})
}

func findRuleNode(values []ast.Node, used []bool, rule RepositoryRuleSettings) int {
	if rule.Match == nil {
		return -1
	}
	for index, value := range values {
		if used[index] {
			continue
		}
		node, ok := value.(*ast.MappingNode)
		if !ok {
			continue
		}
		match := findMappingValue(node, "match")
		if match != nil && match.Value.String() == *rule.Match {
			return index
		}
	}
	return -1
}

func cloneRepositoryRules(rules *[]RepositoryRuleSettings) []RepositoryRuleSettings {
	if rules == nil {
		return nil
	}
	return append([]RepositoryRuleSettings(nil), (*rules)...)
}

func hasFileOptionsSettings(settings RepositoryFileOptionsSettings) bool {
	return hasSettingValue([]settingValue{
		{value: settings.Render},
		{value: settings.MaxFileSize},
		{value: settings.Markdown.TOC},
		{value: settings.Markdown.TOCMinHeadings},
		{value: settings.Markdown.Anchors},
		{value: settings.Markdown.Mermaid},
		{value: settings.Markdown.Math},
		{value: settings.Markdown.FrontmatterTitle},
		{value: settings.HTML.View},
		{value: settings.Code.LineNumbers},
		{value: settings.Code.Theme},
	})
}

func applyNestedValues(parent *ast.MappingNode, key string, values []settingValue) error {
	item := findMappingValue(parent, key)
	if item == nil && !hasSettingValue(values) {
		return nil
	}
	node, err := ensureMapping(parent, key)
	if err != nil {
		return err
	}
	return applyValues(node, values)
}

func applyNestedSettingsSection(root *ast.MappingNode, section, nested string, values []settingValue) error {
	sectionItem := findMappingValue(root, section)
	if sectionItem == nil && !hasSettingValue(values) {
		return nil
	}
	sectionNode, err := ensureMapping(root, section)
	if err != nil {
		return err
	}
	if err := applyNestedValues(sectionNode, nested, values); err != nil {
		return fmt.Errorf("%s.%s: %w", section, nested, err)
	}
	return nil
}

func applySettingsSection(root *ast.MappingNode, section string, values []settingValue) error {
	if findMappingValue(root, section) == nil && !hasSettingValue(values) {
		return nil
	}
	sectionNode, err := ensureMapping(root, section)
	if err != nil {
		return fmt.Errorf("%s: %w", section, err)
	}
	return applyValues(sectionNode, values)
}

func clearUnsetSettingsSection(root *ast.MappingNode, section string, values []settingValue) error {
	item := findMappingValue(root, section)
	if item == nil {
		return nil
	}
	node, ok := item.Value.(*ast.MappingNode)
	if !ok {
		return fmt.Errorf("%s: must be a YAML mapping", section)
	}
	clearUnsetValues(node, values)
	if len(node.Values) == 0 {
		removeMappingValue(root, section)
	}
	return nil
}

func clearUnsetNestedSettingsSection(root *ast.MappingNode, section, nested string, values []settingValue) error {
	sectionItem := findMappingValue(root, section)
	if sectionItem == nil {
		return nil
	}
	sectionNode, ok := sectionItem.Value.(*ast.MappingNode)
	if !ok {
		return fmt.Errorf("%s: must be a YAML mapping", section)
	}
	nestedItem := findMappingValue(sectionNode, nested)
	if nestedItem == nil {
		return nil
	}
	nestedNode, ok := nestedItem.Value.(*ast.MappingNode)
	if !ok {
		return fmt.Errorf("%s.%s: must be a YAML mapping", section, nested)
	}
	clearUnsetValues(nestedNode, values)
	if len(nestedNode.Values) == 0 {
		removeMappingValue(sectionNode, nested)
	}
	if len(sectionNode.Values) == 0 {
		removeMappingValue(root, section)
	}
	return nil
}

func clearUnsetNestedValues(parent *ast.MappingNode, key string, values []settingValue) error {
	item := findMappingValue(parent, key)
	if item == nil {
		return nil
	}
	node, ok := item.Value.(*ast.MappingNode)
	if !ok {
		return fmt.Errorf("%s: must be a YAML mapping", key)
	}
	clearUnsetValues(node, values)
	if len(node.Values) == 0 {
		removeMappingValue(parent, key)
	}
	return nil
}

func hasSettingValue(values []settingValue) bool {
	for _, setting := range values {
		if !isUnsetSettingValue(setting.value) {
			return true
		}
	}
	return false
}

func applyValues(node *ast.MappingNode, values []settingValue) error {
	for _, setting := range values {
		if isUnsetSettingValue(setting.value) {
			continue
		}
		if err := setMappingValue(node, setting.key, setting.value); err != nil {
			return fmt.Errorf("set %s: %w", setting.key, err)
		}
	}
	return nil
}

func clearUnsetValues(node *ast.MappingNode, values []settingValue) {
	for _, setting := range values {
		if isUnsetSettingValue(setting.value) {
			removeMappingValue(node, setting.key)
		}
	}
}

func isUnsetSettingValue(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	return v.Kind() == reflect.Ptr && v.IsNil()
}

func ensureMapping(parent *ast.MappingNode, key string) (*ast.MappingNode, error) {
	item := findMappingValue(parent, key)
	if item == nil {
		child, err := newMappingNode(nil)
		if err != nil {
			return nil, err
		}
		if err := appendMappingValue(parent, key, child); err != nil {
			return nil, err
		}
		item = findMappingValue(parent, key)
	}
	child, ok := item.Value.(*ast.MappingNode)
	if !ok {
		return nil, errors.New("must be a YAML mapping")
	}
	return child, nil
}

func setMappingValue(parent *ast.MappingNode, key string, value any) error {
	node, err := yaml.ValueToNode(value)
	if err != nil {
		return err
	}
	if item := findMappingValue(parent, key); item != nil {
		return item.Replace(node)
	}
	return appendMappingValue(parent, key, node)
}

func appendMappingValue(parent *ast.MappingNode, key string, value ast.Node) error {
	entry, err := newMappingValue(key, "")
	if err != nil {
		return err
	}
	entry.Value = value
	column := parent.GetToken().Position.Column
	if len(parent.Values) > 0 {
		column = parent.Values[0].Key.GetToken().Position.Column
	}
	entry.AddColumn(column - entry.Key.GetToken().Position.Column)
	if mapping, ok := value.(*ast.MappingNode); ok {
		childColumn := entry.Key.GetToken().Position.Column + 2
		mapping.AddColumn(childColumn - mapping.GetToken().Position.Column)
	}
	parent.Values = append(parent.Values, entry)
	return nil
}

func newMappingNode(values yaml.MapSlice) (*ast.MappingNode, error) {
	node, err := yaml.ValueToNode(values)
	if err != nil {
		return nil, err
	}
	mapping, ok := node.(*ast.MappingNode)
	if !ok {
		return nil, errors.New("generated YAML value is not a mapping")
	}
	mapping.IsFlowStyle = false
	return mapping, nil
}

func newSequenceNode() (*ast.SequenceNode, error) {
	node, err := yaml.ValueToNode([]any{})
	if err != nil {
		return nil, err
	}
	sequence, ok := node.(*ast.SequenceNode)
	if !ok {
		return nil, errors.New("generated YAML value is not a sequence")
	}
	sequence.IsFlowStyle = false
	return sequence, nil
}

func newMappingValue(key string, value any) (*ast.MappingValueNode, error) {
	mapping, err := newMappingNode(yaml.MapSlice{{Key: key, Value: value}})
	if err != nil {
		return nil, err
	}
	if len(mapping.Values) != 1 {
		return nil, errors.New("generated YAML mapping has no value")
	}
	return mapping.Values[0], nil
}

func findMappingValue(node *ast.MappingNode, key string) *ast.MappingValueNode {
	for _, item := range node.Values {
		if item.Key.String() == key {
			return item
		}
	}
	return nil
}

func removeMappingValue(node *ast.MappingNode, key string) {
	for index, item := range node.Values {
		if item.Key.String() == key {
			node.Values = append(node.Values[:index], node.Values[index+1:]...)
			return
		}
	}
}

func revisionFor(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}
