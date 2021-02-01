package queryparser

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
)

const (
	TYPE_AUDIO = "ext:wav ext:mp3 ext:ogg ext:flac"
	TYPE_VIDEO = "ext:mp4 ext:mkv ext:avi ext:webm ext:mov"
	TYPE_DOC   = "ext:doc ext:docm ext:pdf ext:docx ext:odf ext:pages ext:rtf"
	TYPE_IMAGE = "ext:jpg ext:jpeg ext:png ext:gif ext:tiff ext:eps ext:raw"
	TYPE_EBOOK = "ext:fb2 ext:ibook ext:cbr ext:djvu ext:epub ext:mobi"
)

var sizeRegexp = regexp.MustCompile(`(?i)(\+?size):([<>=]{0,2})(\d+)(b|kb|mb|gb|tb)?`)
var mtodayRegexp = regexp.MustCompile(`(?i)modified:today`)
var mrecentlyRegexp = regexp.MustCompile(`(?i)modified:recently`)
var myesterdayRegexp = regexp.MustCompile(`(?i)modified:yesterday`)
var utodayRegexp = regexp.MustCompile(`(?i)updated:today`)
var urecentlyRegexp = regexp.MustCompile(`(?i)updated:recently`)
var uyesterdayRegexp = regexp.MustCompile(`(?i)updated:yesterday`)

// Parser represents a parser.
type Parser struct {
	s   *Scanner
	buf struct {
		tok Token  // last read token
		lit string // last read literal
		n   int    // buffer size (max=1)
	}
}

func ParseQuery(q string) (string, error) {
	return NewParser(strings.NewReader(q)).Parse()
}

// NewParser returns a new instance of Parser.
func NewParser(r io.Reader) *Parser {
	return &Parser{s: NewScanner(r)}
}

// Parse parses a Swamp query
func (p *Parser) Parse() (string, error) {
	elements := []string{}

	// Next we should loop over all our comma-delimited fields.
	for {
		// Read a field.
		tok, lit := p.scanIgnoreWhitespace()
		if tok == EOF {
			break
		}

		if tok == IDENT {
			if !strings.HasPrefix(lit, "+") {
				lit = "+" + lit
			}
			elements = append(elements, lit)
		}

		if tok == TYPE {
			elements = append(elements, p.parseType(lit))
		}

		if tok == SIZE {
			lit, err := p.parseSize(lit)
			if err != nil {
				return "", err
			}
			elements = append(elements, lit)
		}

		if tok == MODIFIED {
			elements = append(elements, p.parseModified(lit))
		}

		if tok == UPDATED {
			lit, err := p.parseUpdated(lit)
			if err != nil {
				return "", err
			}
			elements = append(elements, lit)
		}
	}

	return strings.Join(elements, " "), nil
}

// TODO: Consider using https://github.com/olebedev/when enventually
func (p *Parser) parseUpdated(lit string) (string, error) {
	// 2006-01-02T15:04:05-0700
	now := time.Now()
	bod := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Format(time.RFC3339)
	eod := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location()).Format(time.RFC3339)

	if utodayRegexp.Match([]byte(lit)) {
		lit = fmt.Sprintf("+updated:>=\"%s\" +updated:<=\"%s\"", bod, eod)
	}

	if urecentlyRegexp.Match([]byte(lit)) {
		recently := time.Now().AddDate(0, 0, -15)
		rdate := time.Date(recently.Year(), recently.Month(), recently.Day(), 00, 00, 00, 0, now.Location()).Format(time.RFC3339)
		lit = fmt.Sprintf("+updated:>=\"%s\"", rdate)
	}

	if uyesterdayRegexp.Match([]byte(lit)) {
		yest := time.Now().AddDate(0, 0, -1)
		ybod := time.Date(yest.Year(), yest.Month(), yest.Day(), 00, 00, 00, 0, now.Location()).Format(time.RFC3339)
		yeod := time.Date(yest.Year(), yest.Month(), yest.Day(), 23, 59, 59, 0, now.Location()).Format(time.RFC3339)
		lit = fmt.Sprintf("+updated:>=\"%s\" +updated:<=\"%s\"", ybod, yeod)
	}

	return lit, nil
}

func (p *Parser) parseModified(lit string) string {
	now := time.Now()
	bod := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Format(time.RFC3339)
	eod := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location()).Format(time.RFC3339)

	if mtodayRegexp.Match([]byte(lit)) {
		lit = fmt.Sprintf("+mtime:>=\"%s\" +mtime:<=\"%s\"", bod, eod)
	}

	if mrecentlyRegexp.Match([]byte(lit)) {
		recently := time.Now().AddDate(0, 0, -15)
		rdate := time.Date(recently.Year(), recently.Month(), recently.Day(), 00, 00, 00, 0, now.Location()).Format(time.RFC3339)
		lit = fmt.Sprintf("+mtime:>=\"%s\"", rdate)
	}

	if myesterdayRegexp.Match([]byte(lit)) {
		yest := time.Now().AddDate(0, 0, -1)
		ybod := time.Date(yest.Year(), yest.Month(), yest.Day(), 00, 00, 00, 0, now.Location()).Format(time.RFC3339)
		yeod := time.Date(yest.Year(), yest.Month(), yest.Day(), 23, 59, 59, 0, now.Location()).Format(time.RFC3339)
		lit = fmt.Sprintf("+mtime:>=\"%s\" +mtime:<=\"%s\"", ybod, yeod)
	}

	return lit
}

func (p *Parser) parseSize(lit string) (string, error) {
	if !sizeRegexp.Match([]byte(lit)) {
		return "", fmt.Errorf("invalid size '%s' specified", lit)
	}
	res := sizeRegexp.FindAllStringSubmatch(lit, -1)
	hsize := fmt.Sprintf("%s%s", res[0][3], res[0][4])
	var bsize uint64
	var err error
	if res[0][4] != "" {
		bsize, err = bytefmt.ToBytes(hsize)
	} else {
		bsize, err = strconv.ParseUint(hsize, 10, 64)
	}
	if err != nil {
		return "", fmt.Errorf("invalid size '%s' specified", hsize)
	}

	hb := fmt.Sprintf("%s:%s%d", res[0][1], res[0][2], bsize)
	lit = sizeRegexp.ReplaceAllString(lit, hb)
	return lit, nil
}

func (p *Parser) parseType(lit string) string {
	t := strings.Split(lit, ":")
	if len(t) > 1 {
		switch t[1] {
		case "audio":
			lit = TYPE_AUDIO
		case "video":
			lit = TYPE_VIDEO
		case "document", "doc":
			lit = TYPE_DOC
		case "image":
			lit = TYPE_IMAGE
		case "ebook":
			lit = TYPE_EBOOK
		}
	}
	return lit
}

// scan returns the next token from the underlying scanner.
// If a token has been unscanned then read that instead.
func (p *Parser) scan() (tok Token, lit string) {
	// If we have a token on the buffer, then return it.
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}

	// Otherwise read the next token from the scanner.
	tok, lit = p.s.Scan()

	// Save it to the buffer in case we unscan later.
	p.buf.tok, p.buf.lit = tok, lit

	return
}

// scanIgnoreWhitespace scans the next non-whitespace token.
func (p *Parser) scanIgnoreWhitespace() (tok Token, lit string) {
	tok, lit = p.scan()
	if tok == WS {
		tok, lit = p.scan()
	}
	return
}
