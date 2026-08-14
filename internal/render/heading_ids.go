package render

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/util"
)

type headingIDs struct {
	used map[string]struct{}
}

func newHeadingIDs() *headingIDs {
	return &headingIDs{used: make(map[string]struct{})}
}

func (ids *headingIDs) Generate(value []byte, kind ast.NodeKind) []byte {
	value = util.TrimLeftSpace(value)
	value = util.TrimRightSpace(value)

	var slug strings.Builder
	for len(value) > 0 {
		r, size := utf8.DecodeRune(value)
		value = value[size:]
		if r == utf8.RuneError && size == 1 {
			continue
		}

		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r):
			slug.WriteRune(unicode.ToLower(r))
		case unicode.IsSpace(r) || r == '-' || r == '_':
			slug.WriteByte('-')
		}
	}

	base := slug.String()
	if base == "" {
		base = "id"
		if kind == ast.KindHeading {
			base = "heading"
		}
	}
	return []byte(ids.reserve(base))
}

func (ids *headingIDs) Put(value []byte) {
	ids.used[string(value)] = struct{}{}
}

func (ids *headingIDs) reserve(base string) string {
	if _, exists := ids.used[base]; !exists {
		ids.used[base] = struct{}{}
		return base
	}
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, exists := ids.used[candidate]; !exists {
			ids.used[candidate] = struct{}{}
			return candidate
		}
	}
}
