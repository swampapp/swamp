package queryparser

// Token represents a lexical token.
type Token int

const (
	// Special tokens
	ILLEGAL Token = iota
	EOF
	WS

	// literals
	IDENT

	// Keywords
	TYPE
	UPDATED
	MODIFIED
	SIZE
)
