package site

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/ConteMan/repolens/internal/render"
)

func TestGlossarySkillConsistency(t *testing.T) {
	t.Parallel()

	t.Run("field names match implementation", func(t *testing.T) {
		t.Parallel()
		skill := readGlossarySkill(t)
		library, frontMatter, err := parseSkillGlossaryFields(skill)
		if err != nil {
			t.Fatal(err)
		}
		assertSameFields(t, "glossary library", library, yamlFieldNames(glossaryFile{}))
		assertSameFields(t, "front matter overlay", frontMatter, render.GlossaryFrontMatterFields())
	})

	t.Run("example build", func(t *testing.T) {
		t.Parallel()
		repo := copyGlossaryExampleRepo(t)
		outDir, _, err := buildSite(t, repo)
		if err != nil {
			t.Fatalf("build example with strict refs: %v", err)
		}

		page := readOutput(t, outDir, "view/docs/iaa-data-flow.md/index.html")
		assertContains(t, page, `<h3 class="glossary-title">广告聚合</h3>`)
		assertContains(t, page, `<p class="glossary-alias">Ad mediation</p>`)
		assertContains(t, page, "行业通用能力：在多个广告源之间统一请求、竞价或瀑布选择")
		assertContains(t, page, "AdMob 既可能是聚合平台内的一个广告源")
		assertContains(t, page, `href="https://developers.applovin.com/en/max/getting-started/" rel="noreferrer"`)
		assertContains(t, page, `<h3 class="glossary-title">MMP</h3>`)
		assertContains(t, page, `<p class="glossary-alias">Mobile Measurement Partner / 移动归因平台</p>`)
		assertContains(t, page, `<code>af_ad_revenue</code>`)
		assertContains(t, page, `class="glossary-field glossary-page"`)
		assertContains(t, page, "主聚合平台决定展示哪家广告源的广告，并形成展示与收入事实")
		assertContains(t, page, `<h3 class="glossary-title">Organic</h3>`)
		assertContains(t, page, `<p class="glossary-alias">自然用户</p>`)
		assertContains(t, page, "不由付费投放带来的用户")
		assertContains(t, page, `data-glossary="organic"`)
		assertNotContains(t, page, `href="term:`)

		llms := readOutput(t, outDir, "llms.txt")
		idx := strings.Index(llms, "## 术语表")
		if idx < 0 {
			t.Fatalf("llms.txt missing glossary section:\n%s", llms)
		}
		section := llms[idx:]
		assertContains(t, section, "广告聚合")
		assertContains(t, section, "MMP")
		assertNotContains(t, section, "Organic")
		assertNotContains(t, section, "自然用户")
		assertNotContains(t, section, "organic")
		assertNotContains(t, section, "不由付费投放")

		assertNoTermHref(t, outDir)
	})
}

func readGlossarySkill(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(glossarySkillDir(t), "SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}
	return string(data)
}

func glossarySkillDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("..", "..", "skills", "glossary")
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
		t.Fatalf("skills/glossary/SKILL.md: %v", err)
	}
	return dir
}

func copyGlossaryExampleRepo(t *testing.T) string {
	t.Helper()
	src := filepath.Join(glossarySkillDir(t), "example")
	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		writeFileBytes(t, repo, filepath.ToSlash(rel), data)
		return nil
	})
	if err != nil {
		t.Fatalf("copy glossary example: %v", err)
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")
	return repo
}

var skillFieldRow = regexp.MustCompile("^\\| `([a-z][a-z0-9-]*)` \\|")

func parseSkillGlossaryFields(md string) (library, frontMatter []string, err error) {
	const header = "| 字段 | 必填 | 写法 |"
	started := false
	for _, line := range strings.Split(md, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if !started {
			if line == header {
				started = true
			}
			continue
		}
		if line == "" || !strings.HasPrefix(line, "|") {
			break
		}
		if strings.HasPrefix(strings.ReplaceAll(line, " ", ""), "|---") {
			continue
		}
		m := skillFieldRow.FindStringSubmatch(line)
		if m == nil {
			return nil, nil, fmt.Errorf("SKILL.md: unparseable field row: %s", line)
		}
		name := m[1]
		for _, existing := range frontMatter {
			if existing == name {
				return nil, nil, fmt.Errorf("SKILL.md: duplicate field %s", name)
			}
		}
		frontMatter = append(frontMatter, name)
		if skillFieldIsFrontMatterOnly(line) {
			continue
		}
		library = append(library, name)
	}
	if !started {
		return nil, nil, fmt.Errorf("SKILL.md: missing field table header")
	}
	if len(frontMatter) == 0 {
		return nil, nil, fmt.Errorf("SKILL.md: field table is empty")
	}
	return library, frontMatter, nil
}

func skillFieldIsFrontMatterOnly(row string) bool {
	return strings.Contains(row, "front matter") &&
		(strings.Contains(row, "只写在") || strings.Contains(row, "不写在术语库"))
}

func yamlFieldNames(v any) []string {
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	names := make([]string, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("yaml")
		name, _, _ := strings.Cut(tag, ",")
		if name == "" || name == "-" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func assertSameFields(t *testing.T, which string, skill, impl []string) {
	t.Helper()
	onlySkill, onlyImpl := fieldSetDiff(skill, impl)
	if len(onlySkill) == 0 && len(onlyImpl) == 0 {
		return
	}
	t.Fatalf("%s field set mismatch\n  in SKILL.md only: %v\n  in implementation only: %v\n  SKILL.md: %v\n  implementation: %v",
		which, onlySkill, onlyImpl, sortedCopy(skill), sortedCopy(impl))
}

func fieldSetDiff(a, b []string) (onlyA, onlyB []string) {
	inA := make(map[string]struct{}, len(a))
	inB := make(map[string]struct{}, len(b))
	for _, s := range a {
		inA[s] = struct{}{}
	}
	for _, s := range b {
		inB[s] = struct{}{}
	}
	for _, s := range a {
		if _, ok := inB[s]; !ok {
			onlyA = append(onlyA, s)
		}
	}
	for _, s := range b {
		if _, ok := inA[s]; !ok {
			onlyB = append(onlyB, s)
		}
	}
	sort.Strings(onlyA)
	sort.Strings(onlyB)
	return onlyA, onlyB
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func assertNoTermHref(t *testing.T, root string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".html", ".txt", ".json", ".xml":
		default:
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), `href="term:`) {
			rel, _ := filepath.Rel(root, path)
			t.Errorf("%s contains href=\"term:\"", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan output: %v", err)
	}
}
