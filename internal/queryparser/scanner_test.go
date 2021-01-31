package queryparser_test

import (
	"strings"
	"testing"

	parser "github.com/swampapp/swamp/internal/queryparser"
)

// Ensure the scanner can scan tokens correctly.
func TestScanner_Scan(t *testing.T) {
	var tests = []struct {
		s   string
		tok parser.Token
		lit string
	}{
		// Special tokens (EOF, ILLEGAL, WS)
		{s: ``, tok: parser.EOF},
		{s: `#`, tok: parser.ILLEGAL, lit: `#`},
		{s: ` `, tok: parser.WS, lit: " "},
		{s: "\t", tok: parser.WS, lit: "\t"},
		{s: "\n", tok: parser.WS, lit: "\n"},
		{s: `*`, tok: parser.ILLEGAL, lit: `*`},

		// Literals
		{s: `foo`, tok: parser.IDENT, lit: `foo`},
		{s: `"foo"`, tok: parser.IDENT, lit: `"foo"`},
		{s: `bar~2`, tok: parser.IDENT, lit: `bar~2`},
		{s: `ext:mp3`, tok: parser.IDENT, lit: "ext:mp3"},

		// Keywords
		{s: `type:audio`, tok: parser.TYPE, lit: "type:audio"},
		{s: `size:128MB`, tok: parser.SIZE, lit: "size:128MB"},
		{s: `size:128`, tok: parser.SIZE, lit: "size:128"},
		{s: `size:>=128`, tok: parser.SIZE, lit: "size:>=128"},
		{s: `+size:128`, tok: parser.SIZE, lit: "+size:128"},
		{s: `+size:>128`, tok: parser.SIZE, lit: "+size:>128"},
	}

	for i, tt := range tests {
		s := parser.NewScanner(strings.NewReader(tt.s))
		tok, lit := s.Scan()
		if tt.tok != tok {
			t.Errorf("%d. %q token mismatch: exp=%d got=%d <%q>", i, tt.s, tt.tok, tok, lit)
		} else if tt.lit != lit {
			t.Errorf("%d. %q literal mismatch: exp=%q got=%q", i, tt.s, tt.lit, lit)
		}
	}
}
