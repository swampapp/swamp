package queryparser

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

// Scanner represents a lexical scanner.
type Scanner struct {
	r *bufio.Reader
}

// NewScanner returns a new instance of Scanner.
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r)}
}

// Scan returns the next token and literal value.
func (s *Scanner) Scan() (tok Token, lit string) {
	// Read the next rune.
	ch := s.read()

	// If we see whitespace then consume all contiguous whitespace.
	// If we see a letter then consume as an ident or reserved word.
	// If we see a digit then consume as a number.
	if isWhitespace(ch) {
		s.unread()
		return s.scanWhitespace()
	} else if isAllowed(ch) {
		s.unread()
		return s.scanIdent()
	}

	// Otherwise read the individual character.
	switch ch {
	case eof:
		return EOF, ""
	}

	return ILLEGAL, string(ch)
}

// scanWhitespace consumes the current rune and all contiguous whitespace.
func (s *Scanner) scanWhitespace() (tok Token, lit string) {
	// Create a buffer and read the current character into it.
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	// Read every subsequent whitespace character into the buffer.
	// Non-whitespace characters and EOF will cause the loop to exit.
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isWhitespace(ch) {
			s.unread()
			break
		} else {
			buf.WriteRune(ch)
		}
	}

	return WS, buf.String()
}

// scanIdent consumes the current rune and all contiguous ident runes.
func (s *Scanner) scanIdent() (tok Token, lit string) {
	// Create a buffer and read the current character into it.
	var isRequired = false
	var buf bytes.Buffer
	ch := s.read()
	if ch == '+' {
		isRequired = true
	} else {
		buf.WriteRune(ch)
	}

	// Read every subsequent ident character into the buffer.
	// Non-ident characters and EOF will cause the loop to exit.
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isAllowed(ch) {
			s.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}

	finalKey := func() string {
		if isRequired {
			return "+" + buf.String()
		}
		return buf.String()
	}

	// If the string matches a keyword then return that keyword.
	ls := strings.ToLower(buf.String())
	if strings.HasPrefix(ls, "type:") {
		return TYPE, finalKey()
	}
	if strings.HasPrefix(ls, "size:") {
		return SIZE, finalKey()
	}
	if strings.HasPrefix(ls, "modified:") {
		return MODIFIED, finalKey()
	}
	if strings.HasPrefix(ls, "updated:") {
		return UPDATED, finalKey()
	}

	// Otherwise return as a regular identifier.
	return IDENT, buf.String()
}

// read reads the next rune from the buffered reader.
// Returns the rune(0) if an error occurs (or io.EOF is returned).
func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

// unread places the previously read rune back on the reader.
func (s *Scanner) unread() { _ = s.r.UnreadRune() }

// isWhitespace returns true if the rune is a space, tab, or newline.
func isWhitespace(ch rune) bool { return ch == ' ' || ch == '\t' || ch == '\n' }

// isLetter returns true if the rune is a letter.
func isLetter(ch rune) bool { return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') }

func isAllowed(ch rune) bool {
	return isLetter(ch) ||
		isDigit(ch) ||
		ch == ':' ||
		ch == '_' ||
		ch == '"' ||
		ch == '+' ||
		ch == '~'
}

// isDigit returns true if the rune is a digit.
func isDigit(ch rune) bool { return (ch >= '0' && ch <= '9') }

// eof represents a marker rune for the end of the reader.
var eof = rune(0)
