package skill

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/goccy/go-yaml"
)

const (
	digestHexLen   = 12
	skillNameMin   = 1
	skillNameMax   = 64
	frontMatterEnd = "---"
)

var skillNameRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

type parsedSkill struct {
	Name             string
	Description      string
	Metadata         map[string]string
	hasMetadataField bool
	Body             []byte
}

func parseSkillFile(content []byte) (parsedSkill, error) {
	fm, body, _, err := splitFrontMatter(content)
	if err != nil {
		return parsedSkill{}, err
	}
	var raw map[string]any
	if err := yaml.Unmarshal(fm, &raw); err != nil {
		return parsedSkill{}, fmt.Errorf("parse front matter: %w", err)
	}
	if raw == nil {
		raw = map[string]any{}
	}
	out := parsedSkill{Body: body}
	out.Name, _ = anyString(raw["name"])
	out.Description, _ = anyString(raw["description"])
	if meta, ok := raw["metadata"]; ok {
		out.hasMetadataField = true
		out.Metadata = map[string]string{}
		switch m := meta.(type) {
		case map[string]any:
			for k, v := range m {
				if s, ok := anyString(v); ok {
					out.Metadata[k] = s
				}
			}
		case map[string]string:
			for k, v := range m {
				out.Metadata[k] = v
			}
		}
	}
	return out, nil
}

func (p parsedSkill) provenance() (name, version, digest string, ok bool) {
	if p.Metadata == nil {
		return "", "", "", false
	}
	name = strings.TrimSpace(p.Metadata["repolens.skill"])
	version = strings.TrimSpace(p.Metadata["repolens.version"])
	digest = strings.TrimSpace(p.Metadata["repolens.digest"])
	if name == "" || version == "" || digest == "" {
		return "", "", "", false
	}
	return name, version, digest, true
}

func anyString(v any) (string, bool) {
	switch t := v.(type) {
	case nil:
		return "", false
	case string:
		return t, true
	case int:
		return strconv.Itoa(t), true
	case int64:
		return strconv.FormatInt(t, 10), true
	case uint64:
		return strconv.FormatUint(t, 10), true
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(t), true
	default:
		return "", false
	}
}

func splitFrontMatter(content []byte) (fm, body []byte, closerStart int, err error) {
	lineNo := 0
	yamlStart := -1
	foundCloser := false
	walkErr := walkLines(content, func(start, end, next int, text []byte) bool {
		trimmed := bytes.TrimSuffix(text, []byte{'\r'})
		switch lineNo {
		case 0:
			if !bytes.Equal(trimmed, []byte(frontMatterEnd)) {
				err = fmt.Errorf("missing YAML front matter")
				return false
			}
			yamlStart = next
		default:
			if bytes.Equal(trimmed, []byte(frontMatterEnd)) {
				fm = content[yamlStart:start]
				body = content[next:]
				closerStart = start
				foundCloser = true
				return false
			}
		}
		lineNo++
		return next < len(content)
	})
	if walkErr != nil {
		return nil, nil, 0, walkErr
	}
	if err != nil {
		return nil, nil, 0, err
	}
	if !foundCloser {
		return nil, nil, 0, fmt.Errorf("unclosed YAML front matter")
	}
	return fm, body, closerStart, nil
}

func walkLines(content []byte, fn func(start, end, next int, text []byte) bool) error {
	start := 0
	for start <= len(content) {
		end := start
		for end < len(content) && content[end] != '\n' {
			end++
		}
		next := end
		if end < len(content) {
			next = end + 1
		}
		if !fn(start, end, next, content[start:end]) {
			return nil
		}
		if end >= len(content) {
			return nil
		}
		start = next
		if start == len(content) {
			return nil
		}
	}
	return nil
}

func injectProvenance(content []byte, skillName, version, digest string) ([]byte, error) {
	_, _, closerStart, err := splitFrontMatter(content)
	if err != nil {
		return nil, err
	}
	block := fmt.Sprintf("metadata:\n  repolens.skill: %s\n  repolens.version: %s\n  repolens.digest: %s\n",
		skillName, strconv.Quote(version), strconv.Quote(digest))
	before := content[:closerStart]
	if len(before) == 0 || before[len(before)-1] != '\n' {
		block = "\n" + block
	}
	out := make([]byte, 0, len(before)+len(block)+len(content)-closerStart)
	out = append(out, before...)
	out = append(out, block...)
	out = append(out, content[closerStart:]...)
	return out, nil
}

func shortDigest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])[:digestHexLen]
}

func materialize(s Skill, version string) ([]byte, error) {
	_, body, _, err := splitFrontMatter(s.Content)
	if err != nil {
		return nil, err
	}
	return injectProvenance(s.Content, s.Name, version, shortDigest(body))
}

func validateSkillName(name string) error {
	n := utf8.RuneCountInString(name)
	if n < skillNameMin || n > skillNameMax {
		return fmt.Errorf("name %q length %d is outside 1–64", name, n)
	}
	if strings.Contains(name, "--") {
		return fmt.Errorf("name %q contains consecutive hyphens", name)
	}
	if !skillNameRe.MatchString(name) {
		return fmt.Errorf("name %q is not lowercase alphanumeric with hyphens", name)
	}
	return nil
}
