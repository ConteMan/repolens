package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/goccy/go-yaml"
)

const repoConfigName = ".repolens.yml"

// Config is the fully merged repolens configuration.
type Config struct {
	Source Source      `yaml:"source"`
	Output Output      `yaml:"output"`
	Access Access      `yaml:"access"`
	Site   Site        `yaml:"site"`
	Ignore []string    `yaml:"ignore"`
	Render FileOptions `yaml:"render"`
	Rules  []Rule      `yaml:"rules"`
	Theme  Theme       `yaml:"theme"`
	View   View        `yaml:"view"`
	Agent  Agent       `yaml:"agent"`
}

// Source configures where content comes from. It is trusted-only.
type Source struct {
	Repo string `yaml:"repo"`
	Ref  string `yaml:"ref"`
}

// Output configures the generated site destination. It is trusted-only.
type Output struct {
	Dir      string `yaml:"dir"`
	BasePath string `yaml:"base_path"`
}

// Access configures soft protection and reserved encryption settings.
type Access struct {
	NoIndex bool    `yaml:"noindex"`
	Encrypt Encrypt `yaml:"encrypt"`
}

// Encrypt reserves the v1 client-side encryption schema.
type Encrypt struct {
	Enabled     bool     `yaml:"enabled"`
	Paths       []string `yaml:"paths"`
	PasswordEnv string   `yaml:"password_env"`
}

// Site configures site-level metadata.
type Site struct {
	Title    string `yaml:"title"`
	Language string `yaml:"language"`
	Home     string `yaml:"home"`
}

// Rule applies file option overrides to paths matching Match.
//
// Option fields hold only what the rule explicitly sets; unset fields are
// Go zero values, not effective defaults. Always resolve effective options
// through Config.OptionsFor rather than reading these fields directly.
type Rule struct {
	Match       string           `yaml:"match"`
	Render      *bool            `yaml:"render"`
	Markdown    *MarkdownOptions `yaml:"markdown"`
	HTML        *HTMLOptions     `yaml:"html"`
	Code        *CodeOptions     `yaml:"code"`
	MaxFileSize *int64           `yaml:"max_file_size"`

	markdownPatch *markdownOptionsPatch
	htmlPatch     *htmlOptionsPatch
	codePatch     *codeOptionsPatch
}

// FileOptions are the effective rendering options for one file.
type FileOptions struct {
	Render      bool            `yaml:"render"`
	Markdown    MarkdownOptions `yaml:"markdown"`
	HTML        HTMLOptions     `yaml:"html"`
	Code        CodeOptions     `yaml:"code"`
	MaxFileSize int64           `yaml:"max_file_size"`
}

// MarkdownOptions configures Markdown rendering.
type MarkdownOptions struct {
	TOC              bool `yaml:"toc"`
	TOCMinHeadings   int  `yaml:"toc_min_headings"`
	Anchors          bool `yaml:"anchors"`
	Mermaid          bool `yaml:"mermaid"`
	Math             bool `yaml:"math"`
	FrontmatterTitle bool `yaml:"frontmatter_title"`
}

// HTMLOptions configures browser pages for HTML files.
type HTMLOptions struct {
	View string `yaml:"view"`
}

// CodeOptions configures code rendering.
type CodeOptions struct {
	LineNumbers bool   `yaml:"line_numbers"`
	Theme       string `yaml:"theme"`
}

// Theme configures user-overridable theme inputs.
type Theme struct {
	Vars      map[string]string `yaml:"vars"`
	CSS       string            `yaml:"css"`
	Templates string            `yaml:"templates"`
}

// View configures browser UI layout.
type View struct {
	TreePosition    string `yaml:"tree_position"`
	TreeExpandDepth int    `yaml:"tree_expand_depth"`
	// TOCPanel controls Markdown TOC placement: "floating" or "inline".
	TOCPanel string `yaml:"toc_panel"`
	// Search controls the browser-layer filename and heading search UI.
	Search bool `yaml:"search"`
}

// Agent configures machine-readable outputs.
type Agent struct {
	LLMSTxt   bool          `yaml:"llms_txt"`
	LLMSFull  AgentFullText `yaml:"llms_full"`
	IndexJSON bool          `yaml:"index_json"`
}

// AgentFullText configures llms-full.txt.
type AgentFullText struct {
	Enabled bool  `yaml:"enabled"`
	MaxSize int64 `yaml:"max_size"`
}

// Flags contains trusted CLI overrides. Empty string fields mean "not set".
type Flags struct {
	Repo      string
	Ref       string
	OutputDir string
	BasePath  string
}

// Warning is a recoverable configuration issue reported to the caller.
type Warning struct {
	Msg string
}

// Load reads and merges defaults, repo config, external config, and CLI flags.
func Load(repoRoot, externalPath string, flags Flags) (*Config, []Warning, error) {
	cfg := defaultConfig()
	var warnings []Warning

	if repoRoot != "" {
		repoPath := filepath.Join(repoRoot, repoConfigName)
		if patch, found, ws, err := loadPatch(repoPath, true); err != nil {
			return nil, warnings, err
		} else if found {
			warnings = append(warnings, ws...)
			applyPatch(&cfg, patch)
		}
	}

	if externalPath != "" {
		patch, found, ws, err := loadPatch(externalPath, false)
		if err != nil {
			return nil, warnings, err
		}
		if !found {
			return nil, warnings, fmt.Errorf("load config %q: file does not exist", externalPath)
		}
		warnings = append(warnings, ws...)
		applyPatch(&cfg, patch)
	}

	applyFlags(&cfg, flags)
	warnings = append(warnings, lintConfig(cfg)...)
	return &cfg, warnings, nil
}

// OptionsFor returns the effective rendering options for a repo-relative path.
func (c *Config) OptionsFor(filePath string) FileOptions {
	opts := c.Render
	name := cleanRepoPath(filePath)
	for _, rule := range c.Rules {
		matched, err := doublestar.Match(rule.Match, name)
		if err != nil || !matched {
			continue
		}
		applyRule(&opts, rule)
	}
	return opts
}

// Ignored reports whether a repo-relative path is excluded from all outputs.
func (c *Config) Ignored(filePath string) bool {
	name := cleanRepoPath(filePath)
	if name == repoConfigName || name == ".git" || strings.HasPrefix(name, ".git/") {
		return true
	}
	for _, pattern := range c.Ignore {
		matched, err := doublestar.Match(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func defaultConfig() Config {
	return Config{
		Output: Output{
			Dir:      "dist",
			BasePath: "/",
		},
		Access: Access{
			NoIndex: true,
		},
		Site: Site{
			Language: "zh-CN",
			Home:     "README.md",
		},
		Ignore: []string{
			"node_modules/**",
			".github/**",
		},
		Render: FileOptions{
			Render: true,
			Markdown: MarkdownOptions{
				TOC:              true,
				TOCMinHeadings:   3,
				Anchors:          true,
				Mermaid:          true,
				FrontmatterTitle: true,
			},
			HTML: HTMLOptions{
				View: "embed",
			},
			Code: CodeOptions{
				LineNumbers: true,
				Theme:       "github",
			},
			MaxFileSize: int64(5 * 1024 * 1024),
		},
		View: View{
			TreePosition:    "left",
			TreeExpandDepth: 2,
			TOCPanel:        "floating",
			Search:          true,
		},
		Agent: Agent{
			LLMSTxt: true,
			LLMSFull: AgentFullText{
				Enabled: true,
				MaxSize: int64(2 * 1024 * 1024),
			},
			IndexJSON: true,
		},
	}
}

func cleanRepoPath(filePath string) string {
	name := filepath.ToSlash(filePath)
	name = strings.TrimPrefix(name, "./")
	name = strings.TrimPrefix(name, "/")
	if name == "" {
		return ""
	}
	cleaned := path.Clean(name)
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func applyFlags(cfg *Config, flags Flags) {
	if flags.Repo != "" {
		cfg.Source.Repo = flags.Repo
	}
	if flags.Ref != "" {
		cfg.Source.Ref = flags.Ref
	}
	if flags.OutputDir != "" {
		cfg.Output.Dir = flags.OutputDir
	}
	if flags.BasePath != "" {
		cfg.Output.BasePath = flags.BasePath
	}
}

func applyRule(opts *FileOptions, rule Rule) {
	if rule.Render != nil {
		opts.Render = *rule.Render
	}
	if rule.Markdown != nil {
		if rule.markdownPatch != nil {
			applyMarkdownPatch(&opts.Markdown, rule.markdownPatch)
		} else {
			opts.Markdown = *rule.Markdown
		}
	}
	if rule.HTML != nil {
		if rule.htmlPatch != nil {
			applyHTMLPatch(&opts.HTML, rule.htmlPatch)
		} else {
			opts.HTML = *rule.HTML
		}
	}
	if rule.Code != nil {
		if rule.codePatch != nil {
			applyCodePatch(&opts.Code, rule.codePatch)
		} else {
			opts.Code = *rule.Code
		}
	}
	if rule.MaxFileSize != nil {
		opts.MaxFileSize = *rule.MaxFileSize
	}
}

func lintConfig(cfg Config) []Warning {
	warnings := lintViewConfig(cfg.View)
	if !cfg.Access.Encrypt.Enabled {
		return warnings
	}
	agentPaths := enabledAgentPaths(cfg.Agent)
	if len(agentPaths) == 0 {
		return warnings
	}
	if len(cfg.Access.Encrypt.Paths) == 0 {
		return append(warnings, Warning{Msg: "access.encrypt covers the whole site while agent outputs are enabled"})
	}
	for _, pattern := range cfg.Access.Encrypt.Paths {
		for _, agentPath := range agentPaths {
			matched, err := doublestar.Match(pattern, agentPath)
			if err == nil && matched {
				return append(warnings, Warning{Msg: "access.encrypt paths overlap enabled agent outputs"})
			}
		}
	}
	return warnings
}

func lintViewConfig(view View) []Warning {
	if view.TOCPanel != "floating" && view.TOCPanel != "inline" {
		return []Warning{{Msg: fmt.Sprintf("view.toc_panel %q is invalid; expected floating or inline", view.TOCPanel)}}
	}
	return nil
}

func enabledAgentPaths(agent Agent) []string {
	var paths []string
	if agent.LLMSTxt {
		paths = append(paths, "llms.txt")
	}
	if agent.LLMSFull.Enabled {
		paths = append(paths, "llms-full.txt")
	}
	if agent.IndexJSON {
		paths = append(paths, "index.json")
	}
	return paths
}

type byteSize int64

func (s *byteSize) UnmarshalYAML(raw []byte) error {
	var n int64
	if err := yaml.Unmarshal(raw, &n); err == nil {
		if n < 0 {
			return fmt.Errorf("size must be non-negative")
		}
		*s = byteSize(n)
		return nil
	}

	var text string
	if err := yaml.Unmarshal(raw, &text); err != nil {
		return err
	}
	n, err := parseByteSize(text)
	if err != nil {
		return err
	}
	*s = byteSize(n)
	return nil
}

func parseByteSize(text string) (int64, error) {
	input := strings.TrimSpace(text)
	if input == "" {
		return 0, fmt.Errorf("size is empty")
	}
	i := 0
	for i < len(input) && (unicode.IsDigit(rune(input[i])) || input[i] == '.') {
		i++
	}
	numberPart := strings.TrimSpace(input[:i])
	unitPart := strings.ToUpper(strings.TrimSpace(input[i:]))
	if numberPart == "" {
		return 0, fmt.Errorf("invalid size %q", text)
	}
	value, err := strconv.ParseFloat(numberPart, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q: %w", text, err)
	}
	if value < 0 {
		return 0, fmt.Errorf("size must be non-negative")
	}
	multiplier, ok := map[string]float64{
		"":   1,
		"B":  1,
		"KB": 1024,
		"K":  1024,
		"MB": 1024 * 1024,
		"M":  1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"G":  1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
		"T":  1024 * 1024 * 1024 * 1024,
	}[unitPart]
	if !ok {
		return 0, fmt.Errorf("unknown size unit %q", unitPart)
	}
	return int64(value * multiplier), nil
}

type configPatch struct {
	Source *sourcePatch      `yaml:"source"`
	Output *outputPatch      `yaml:"output"`
	Access *accessPatch      `yaml:"access"`
	Site   *sitePatch        `yaml:"site"`
	Ignore *[]string         `yaml:"ignore"`
	Render *fileOptionsPatch `yaml:"render"`
	Rules  *[]rulePatch      `yaml:"rules"`
	Theme  *themePatch       `yaml:"theme"`
	View   *viewPatch        `yaml:"view"`
	Agent  *agentPatch       `yaml:"agent"`
}

type sourcePatch struct {
	Repo *string `yaml:"repo"`
	Ref  *string `yaml:"ref"`
}

type outputPatch struct {
	Dir      *string `yaml:"dir"`
	BasePath *string `yaml:"base_path"`
}

type accessPatch struct {
	NoIndex *bool         `yaml:"noindex"`
	Encrypt *encryptPatch `yaml:"encrypt"`
}

type encryptPatch struct {
	Enabled     *bool     `yaml:"enabled"`
	Paths       *[]string `yaml:"paths"`
	PasswordEnv *string   `yaml:"password_env"`
}

type sitePatch struct {
	Title    *string `yaml:"title"`
	Language *string `yaml:"language"`
	Home     *string `yaml:"home"`
}

type fileOptionsPatch struct {
	Render      *bool                 `yaml:"render"`
	Markdown    *markdownOptionsPatch `yaml:"markdown"`
	HTML        *htmlOptionsPatch     `yaml:"html"`
	Code        *codeOptionsPatch     `yaml:"code"`
	MaxFileSize *byteSize             `yaml:"max_file_size"`
}

type markdownOptionsPatch struct {
	TOC              *bool `yaml:"toc"`
	TOCMinHeadings   *int  `yaml:"toc_min_headings"`
	Anchors          *bool `yaml:"anchors"`
	Mermaid          *bool `yaml:"mermaid"`
	Math             *bool `yaml:"math"`
	FrontmatterTitle *bool `yaml:"frontmatter_title"`
}

type htmlOptionsPatch struct {
	View *string `yaml:"view"`
}

type codeOptionsPatch struct {
	LineNumbers *bool   `yaml:"line_numbers"`
	Theme       *string `yaml:"theme"`
}

type rulePatch struct {
	Match       *string               `yaml:"match"`
	Render      *bool                 `yaml:"render"`
	Markdown    *markdownOptionsPatch `yaml:"markdown"`
	HTML        *htmlOptionsPatch     `yaml:"html"`
	Code        *codeOptionsPatch     `yaml:"code"`
	MaxFileSize *byteSize             `yaml:"max_file_size"`
}

type themePatch struct {
	Vars      *map[string]string `yaml:"vars"`
	CSS       *string            `yaml:"css"`
	Templates *string            `yaml:"templates"`
}

type viewPatch struct {
	TreePosition    *string `yaml:"tree_position"`
	TreeExpandDepth *int    `yaml:"tree_expand_depth"`
	TOCPanel        *string `yaml:"toc_panel"`
	Search          *bool   `yaml:"search"`
}

type agentPatch struct {
	LLMSTxt   *bool               `yaml:"llms_txt"`
	LLMSFull  *agentFullTextPatch `yaml:"llms_full"`
	IndexJSON *bool               `yaml:"index_json"`
}

type agentFullTextPatch struct {
	Enabled *bool     `yaml:"enabled"`
	MaxSize *byteSize `yaml:"max_size"`
}

func loadPatch(filePath string, repoDomain bool) (configPatch, bool, []Warning, error) {
	var patch configPatch
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return patch, false, nil, nil
		}
		return patch, false, nil, fmt.Errorf("read config %q: %w", filePath, err)
	}

	var warnings []Warning
	var strict configPatch
	if err := yaml.UnmarshalWithOptions(data, &strict, yaml.DisallowUnknownField()); err != nil {
		warnings = append(warnings, Warning{Msg: fmt.Sprintf("%s: unknown field or schema warning: %v", filePath, err)})
	}

	if err := yaml.Unmarshal(data, &patch); err != nil {
		return patch, true, warnings, fmt.Errorf("parse config %q: %w", filePath, err)
	}

	if repoDomain {
		if patch.Source != nil {
			warnings = append(warnings, Warning{Msg: fmt.Sprintf("%s: source is ignored in repository config", filePath)})
			patch.Source = nil
		}
		if patch.Output != nil {
			warnings = append(warnings, Warning{Msg: fmt.Sprintf("%s: output is ignored in repository config", filePath)})
			patch.Output = nil
		}
		if patch.Access != nil {
			warnings = append(warnings, Warning{Msg: fmt.Sprintf("%s: access is ignored in repository config", filePath)})
			patch.Access = nil
		}
	}
	return patch, true, warnings, nil
}

func applyPatch(cfg *Config, patch configPatch) {
	if patch.Source != nil {
		applySourcePatch(&cfg.Source, patch.Source)
	}
	if patch.Output != nil {
		applyOutputPatch(&cfg.Output, patch.Output)
	}
	if patch.Access != nil {
		applyAccessPatch(&cfg.Access, patch.Access)
	}
	if patch.Site != nil {
		applySitePatch(&cfg.Site, patch.Site)
	}
	if patch.Ignore != nil {
		cfg.Ignore = append([]string(nil), (*patch.Ignore)...)
	}
	if patch.Render != nil {
		applyFileOptionsPatch(&cfg.Render, patch.Render)
	}
	if patch.Rules != nil {
		for _, rule := range *patch.Rules {
			cfg.Rules = append(cfg.Rules, rule.toRule())
		}
	}
	if patch.Theme != nil {
		applyThemePatch(&cfg.Theme, patch.Theme)
	}
	if patch.View != nil {
		applyViewPatch(&cfg.View, patch.View)
	}
	if patch.Agent != nil {
		applyAgentPatch(&cfg.Agent, patch.Agent)
	}
}

func applySourcePatch(dst *Source, patch *sourcePatch) {
	if patch.Repo != nil {
		dst.Repo = *patch.Repo
	}
	if patch.Ref != nil {
		dst.Ref = *patch.Ref
	}
}

func applyOutputPatch(dst *Output, patch *outputPatch) {
	if patch.Dir != nil {
		dst.Dir = *patch.Dir
	}
	if patch.BasePath != nil {
		dst.BasePath = *patch.BasePath
	}
}

func applyAccessPatch(dst *Access, patch *accessPatch) {
	if patch.NoIndex != nil {
		dst.NoIndex = *patch.NoIndex
	}
	if patch.Encrypt != nil {
		applyEncryptPatch(&dst.Encrypt, patch.Encrypt)
	}
}

func applyEncryptPatch(dst *Encrypt, patch *encryptPatch) {
	if patch.Enabled != nil {
		dst.Enabled = *patch.Enabled
	}
	if patch.Paths != nil {
		dst.Paths = append([]string(nil), (*patch.Paths)...)
	}
	if patch.PasswordEnv != nil {
		dst.PasswordEnv = *patch.PasswordEnv
	}
}

func applySitePatch(dst *Site, patch *sitePatch) {
	if patch.Title != nil {
		dst.Title = *patch.Title
	}
	if patch.Language != nil {
		dst.Language = *patch.Language
	}
	if patch.Home != nil {
		dst.Home = *patch.Home
	}
}

func applyFileOptionsPatch(dst *FileOptions, patch *fileOptionsPatch) {
	if patch.Render != nil {
		dst.Render = *patch.Render
	}
	if patch.Markdown != nil {
		applyMarkdownPatch(&dst.Markdown, patch.Markdown)
	}
	if patch.HTML != nil {
		applyHTMLPatch(&dst.HTML, patch.HTML)
	}
	if patch.Code != nil {
		applyCodePatch(&dst.Code, patch.Code)
	}
	if patch.MaxFileSize != nil {
		dst.MaxFileSize = int64(*patch.MaxFileSize)
	}
}

func applyMarkdownPatch(dst *MarkdownOptions, patch *markdownOptionsPatch) {
	if patch.TOC != nil {
		dst.TOC = *patch.TOC
	}
	if patch.TOCMinHeadings != nil {
		dst.TOCMinHeadings = *patch.TOCMinHeadings
	}
	if patch.Anchors != nil {
		dst.Anchors = *patch.Anchors
	}
	if patch.Mermaid != nil {
		dst.Mermaid = *patch.Mermaid
	}
	if patch.Math != nil {
		dst.Math = *patch.Math
	}
	if patch.FrontmatterTitle != nil {
		dst.FrontmatterTitle = *patch.FrontmatterTitle
	}
}

func applyHTMLPatch(dst *HTMLOptions, patch *htmlOptionsPatch) {
	if patch.View != nil {
		dst.View = *patch.View
	}
}

func applyCodePatch(dst *CodeOptions, patch *codeOptionsPatch) {
	if patch.LineNumbers != nil {
		dst.LineNumbers = *patch.LineNumbers
	}
	if patch.Theme != nil {
		dst.Theme = *patch.Theme
	}
}

func applyThemePatch(dst *Theme, patch *themePatch) {
	if patch.Vars != nil {
		dst.Vars = make(map[string]string, len(*patch.Vars))
		for key, value := range *patch.Vars {
			dst.Vars[key] = value
		}
	}
	if patch.CSS != nil {
		dst.CSS = *patch.CSS
	}
	if patch.Templates != nil {
		dst.Templates = *patch.Templates
	}
}

func applyViewPatch(dst *View, patch *viewPatch) {
	if patch.TreePosition != nil {
		dst.TreePosition = *patch.TreePosition
	}
	if patch.TreeExpandDepth != nil {
		dst.TreeExpandDepth = *patch.TreeExpandDepth
	}
	if patch.TOCPanel != nil {
		dst.TOCPanel = *patch.TOCPanel
	}
	if patch.Search != nil {
		dst.Search = *patch.Search
	}
}

func applyAgentPatch(dst *Agent, patch *agentPatch) {
	if patch.LLMSTxt != nil {
		dst.LLMSTxt = *patch.LLMSTxt
	}
	if patch.LLMSFull != nil {
		applyAgentFullTextPatch(&dst.LLMSFull, patch.LLMSFull)
	}
	if patch.IndexJSON != nil {
		dst.IndexJSON = *patch.IndexJSON
	}
}

func applyAgentFullTextPatch(dst *AgentFullText, patch *agentFullTextPatch) {
	if patch.Enabled != nil {
		dst.Enabled = *patch.Enabled
	}
	if patch.MaxSize != nil {
		dst.MaxSize = int64(*patch.MaxSize)
	}
}

func (patch rulePatch) toRule() Rule {
	rule := Rule{
		Render:        patch.Render,
		markdownPatch: patch.Markdown,
		htmlPatch:     patch.HTML,
		codePatch:     patch.Code,
	}
	if patch.Match != nil {
		rule.Match = *patch.Match
	}
	if patch.Markdown != nil {
		md := MarkdownOptions{}
		applyMarkdownPatch(&md, patch.Markdown)
		rule.Markdown = &md
	}
	if patch.HTML != nil {
		html := HTMLOptions{}
		applyHTMLPatch(&html, patch.HTML)
		rule.HTML = &html
	}
	if patch.Code != nil {
		code := CodeOptions{}
		applyCodePatch(&code, patch.Code)
		rule.Code = &code
	}
	if patch.MaxFileSize != nil {
		size := int64(*patch.MaxFileSize)
		rule.MaxFileSize = &size
	}
	return rule
}
