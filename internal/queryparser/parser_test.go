package queryparser_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/swampapp/swamp/internal/queryparser"
)

func TestParser_ParseStatement(t *testing.T) {
	now := time.Now()
	bod := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Format(time.RFC3339)
	eod := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location()).Format(time.RFC3339)
	recently := time.Now().AddDate(0, 0, -15)
	rdate := time.Date(recently.Year(), recently.Month(), recently.Day(), 00, 00, 00, 0, now.Location()).Format(time.RFC3339)
	yest := time.Now().AddDate(0, 0, -1)
	ybod := time.Date(yest.Year(), yest.Month(), yest.Day(), 00, 00, 00, 0, now.Location()).Format(time.RFC3339)
	yeod := time.Date(yest.Year(), yest.Month(), yest.Day(), 23, 59, 59, 0, now.Location()).Format(time.RFC3339)

	var tests = []struct {
		q   string
		e   string
		err string
	}{
		{
			q: `foo bar stuff`,
			e: `+foo +bar +stuff`,
		},

		{
			q: `+foo bar`,
			e: `+foo +bar`,
		},

		{
			q: `bar~2`,
			e: `+bar~2`,
		},

		{
			q: `type:audio`,
			e: queryparser.TYPE_AUDIO,
		},

		{
			q: `type:video`,
			e: queryparser.TYPE_VIDEO,
		},

		{
			q: `type:doc`,
			e: queryparser.TYPE_DOC,
		},

		{
			q: `type:image`,
			e: queryparser.TYPE_IMAGE,
		},

		{
			q: `type:video foo`,
			e: fmt.Sprintf("%s +foo", queryparser.TYPE_VIDEO),
		},

		{
			q: `type:ebook`,
			e: queryparser.TYPE_EBOOK,
		},

		{
			q: `size:10 foo`,
			e: "size:10 +foo",
		},

		{
			q: `size:>=5gb`,
			e: `size:>=5368709120`,
		},

		{
			q: `size:1mb foo`,
			e: `size:1048576 +foo`,
		},

		{
			q: `+size:5gb foo`,
			e: `+size:5368709120 +foo`,
		},

		{
			q: `modified:today`,
			e: fmt.Sprintf(`+mtime:>="%s" +mtime:<="%s"`, bod, eod),
		},
		{
			q: `modified:recently`,
			e: fmt.Sprintf(`+mtime:>="%s"`, rdate),
		},
		{
			q: `modified:yesterday`,
			e: fmt.Sprintf(`+mtime:>="%s" +mtime:<="%s"`, ybod, yeod),
		},

		{
			q: `updated:today`,
			e: fmt.Sprintf(`+updated:>="%s" +updated:<="%s"`, bod, eod),
		},
		{
			q: `updated:recently`,
			e: fmt.Sprintf(`+updated:>="%s"`, rdate),
		},
		{
			q: `updated:yesterday`,
			e: fmt.Sprintf(`+updated:>="%s" +updated:<="%s"`, ybod, yeod),
		},

		{q: `size:bMB`, err: `invalid size 'size:bMB' specified`},
		{q: `size:cc`, err: `invalid size 'size:cc' specified`},
	}

	for i, tt := range tests {
		res, err := queryparser.NewParser(strings.NewReader(tt.q)).Parse()
		if !reflect.DeepEqual(tt.err, errstring(err)) {
			t.Errorf("%d. %q: error mismatch:\n  exp=%s\n  got=%s\n\n", i, tt.q, tt.err, err)
		} else if tt.err == "" && res != tt.e {
			t.Errorf("result\n'%s' does not match\n'%s'", res, tt.e)
		}
	}
}

func errstring(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
