package render

import (
	"fmt"
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	const maxFileSize int64 = 1024
	tests := []struct {
		name string
		path string
		size int64
		data []byte
		want Kind
	}{
		{
			name: "markdown extension",
			path: "README.markdown",
			size: 12,
			data: []byte("# Title\n"),
			want: KindMarkdown,
		},
		{
			name: "html extension",
			path: "docs/index.htm",
			size: 15,
			data: []byte("<h1>Title</h1>"),
			want: KindHTML,
		},
		{
			name: "code lexer extension",
			path: "cmd/main.go",
			size: 13,
			data: []byte("package main\n"),
			want: KindCode,
		},
		{
			name: "extensionless text",
			path: "LICENSE",
			size: 20,
			data: []byte("Permission is hereby granted\n"),
			want: KindCode,
		},
		{
			name: "image extension",
			path: "assets/logo.SVG",
			size: 11,
			data: []byte("<svg></svg>"),
			want: KindImage,
		},
		{
			name: "nul binary",
			path: "blob.dat",
			size: 4,
			data: []byte{0x01, 0x00, 0x02, 0x03},
			want: KindBinary,
		},
		{
			name: "oversized markdown degrades",
			path: "README.md",
			size: maxFileSize + 1,
			data: []byte("# Title\n"),
			want: KindBinary,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Classify(tt.path, tt.size, tt.data, maxFileSize); got != tt.want {
				t.Fatalf("Classify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClassifyNegativeExamples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		data    []byte
		notWant Kind
	}{
		{name: "markdown negative", path: "README.txt", data: []byte("hello\n"), notWant: KindMarkdown},
		{name: "html negative", path: "template.jsx", data: []byte("export const x = <div />\n"), notWant: KindHTML},
		// 扩展名不能命中任何 chroma lexer，验证"无 lexer + NUL 内容 → Binary"。
		{name: "code negative", path: "archive.opaqueblob", data: []byte{0x7f, 0x00, 0x01}, notWant: KindCode},
		{name: "image negative", path: "logo.svg.txt", data: []byte("not an image\n"), notWant: KindImage},
		{name: "binary negative", path: "notes", data: []byte("plain text\n"), notWant: KindBinary},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Classify(tt.path, int64(len(tt.data)), tt.data, 1024); got == tt.notWant {
				t.Fatalf("Classify() = %v, did not want %v", got, tt.notWant)
			}
		})
	}
}

func TestCodeGolden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		src      []byte
		lang     string
		lines    int
		golden   string
	}{
		{
			name:     "go",
			filename: "main.go",
			src:      fortyTwoLineGoSource(),
			lang:     "go",
			lines:    42,
			golden:   "code_go.html",
		},
		{
			name:     "javascript",
			filename: "app.js",
			src:      []byte("export function greet(name) {\n  return `hello ${name}`;\n}\n"),
			lang:     "js",
			lines:    3,
			golden:   "code_js.html",
		},
		{
			name:     "yaml",
			filename: "config.yaml",
			src:      []byte("site:\n  title: RepoLens\nrender:\n  code:\n    line_numbers: true\n"),
			lang:     "yaml",
			lines:    5,
			golden:   "code_yaml.html",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Code(tt.src, tt.filename, CodeOptions{LineNumbers: true, Theme: "github"})
			if err != nil {
				t.Fatalf("Code() error = %v", err)
			}
			if got.Lang != tt.lang {
				t.Fatalf("Lang = %q, want %q", got.Lang, tt.lang)
			}
			if got.Lines != tt.lines {
				t.Fatalf("Lines = %d, want %d", got.Lines, tt.lines)
			}
			html := string(got.HTML)
			requireContains(t, html, `class="chroma"`)
			requireContains(t, html, `class="ln"`)
			if tt.name == "go" {
				requireContains(t, html, `id="L42"`)
				requireContains(t, html, `href="#L42"`)
			}
			assertGolden(t, tt.golden, html)
		})
	}
}

func TestCodeUnknownExtensionFallsBackToPlaintext(t *testing.T) {
	t.Parallel()

	got, err := Code([]byte("just text\n"), "README.unknown-ext", CodeOptions{LineNumbers: false, Theme: "github"})
	if err != nil {
		t.Fatalf("Code() error = %v", err)
	}
	if got.Lang != "text" {
		t.Fatalf("Lang = %q, want text", got.Lang)
	}
	if got.Lines != 1 {
		t.Fatalf("Lines = %d, want 1", got.Lines)
	}
	html := string(got.HTML)
	requireContains(t, html, `class="chroma"`)
	if strings.Contains(html, `class="ln"`) {
		t.Fatalf("unexpected line numbers in plaintext fallback\n%s", html)
	}
}

func TestStylesCSS(t *testing.T) {
	t.Parallel()

	css, err := StylesCSS("github")
	if err != nil {
		t.Fatalf("StylesCSS(github) error = %v", err)
	}
	if strings.TrimSpace(css) == "" {
		t.Fatalf("StylesCSS(github) returned empty CSS")
	}
	requireContains(t, css, ".chroma")
	requireContains(t, css, ".ln")

	if _, err := StylesCSS("definitely-not-a-theme"); err == nil {
		t.Fatalf("StylesCSS(unknown) error = nil, want error")
	}
}

func fortyTwoLineGoSource() []byte {
	lines := make([]string, 0, 42)
	lines = append(lines, "package main", "", "func main() {")
	for i := 4; i <= 41; i++ {
		lines = append(lines, fmt.Sprintf("\tprintln(%d)", i))
	}
	lines = append(lines, "}")
	return []byte(strings.Join(lines, "\n") + "\n")
}
