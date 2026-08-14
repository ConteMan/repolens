package render

import (
	"testing"

	"github.com/yuin/goldmark/ast"
)

func TestHeadingIDsGenerate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "English compatibility", value: "Document Heading", want: "document-heading"},
		{name: "ASCII punctuation", value: "Hello, World!", want: "hello-world"},
		{name: "underscore", value: "API_v2", want: "api-v2"},
		{name: "outer spaces", value: "  Leading and trailing  ", want: "leading-and-trailing"},
		{name: "Chinese", value: "5.6 广告", want: "56-广告"},
		{name: "mixed language", value: "API 接口", want: "api-接口"},
		{name: "combining mark", value: "Cafe\u0301", want: "cafe\u0301"},
		{name: "empty result", value: "😀", want: "heading"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids := newHeadingIDs()
			if got := string(ids.Generate([]byte(tt.value), ast.KindHeading)); got != tt.want {
				t.Fatalf("Generate(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestHeadingIDsDeduplicate(t *testing.T) {
	t.Parallel()

	ids := newHeadingIDs()
	for i, want := range []string{"重复-标题", "重复-标题-1", "重复-标题-2"} {
		if got := string(ids.Generate([]byte("重复 标题"), ast.KindHeading)); got != want {
			t.Fatalf("Generate() call %d = %q, want %q", i+1, got, want)
		}
	}
}
