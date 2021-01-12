package queries

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/rs/zerolog/log"
)

func Parse(q string) string {
	q = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(q, "*", ""), ".", " "))
	// less than one week ago
	// Time format: 2006-01-02T15:04:05Z RFC3339
	var qdate, nq string
	r := regexp.MustCompile(`(?i)added:recently`)
	if r.Match([]byte(q)) {
		qdate = time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
		nq = fmt.Sprintf("updated:>\"%s\"", qdate)
		q = r.ReplaceAllString(q, nq)
	}

	r = regexp.MustCompile(`(?i)added:today`)
	if r.Match([]byte(q)) {
		qdate = time.Now().AddDate(0, 0, -1).Format(time.RFC3339)
		tomorrow := time.Now().AddDate(0, 0, +1).Format(time.RFC3339)
		nq = fmt.Sprintf("updated:>\"%s\" updated:<\"%s\"", qdate, tomorrow)
		q = r.ReplaceAllString(q, nq)
	}

	r = regexp.MustCompile(`(?i)added:yesterday`)
	if r.Match([]byte(q)) {
		qdate = time.Now().AddDate(0, 0, -1).Format(time.RFC3339)
		nq = fmt.Sprintf("updated:>=\"%s\" updated:<=\"%s\"", qdate, time.Now())
		q = r.ReplaceAllString(q, nq)
	}

	r = regexp.MustCompile(`(?i)modified:recently`)
	if r.Match([]byte(q)) {
		qdate = time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
		nq = fmt.Sprintf("mtime:>\"%s\"", qdate)
		q = r.ReplaceAllString(q, nq)
	}

	r = regexp.MustCompile(`(?i)modified:today`)
	if r.Match([]byte(q)) {
		qdate = time.Now().AddDate(0, 0, -1).Format(time.RFC3339)
		tomorrow := time.Now().AddDate(0, 0, +1).Format(time.RFC3339)
		nq = fmt.Sprintf("mtime:>\"%s\" mtime:<\"%s\"", qdate, tomorrow)
		q = r.ReplaceAllString(q, nq)
	}

	r = regexp.MustCompile(`(?i)modified:yesterday`)
	if r.Match([]byte(q)) {
		qdate = time.Now().AddDate(0, 0, -1).Format(time.RFC3339)
		nq = fmt.Sprintf("mtime:>=\"%s\" mtime:<=\"%s\"", qdate, time.Now())
		q = r.ReplaceAllString(q, nq)
	}

	r = regexp.MustCompile(`(?i)size:([<>=]{0,2})(\d*)(b|kb|mb|gb|tb)?`)
	res := r.FindAllStringSubmatch(q, -1)
	if r.Match([]byte(q)) {
		hsize := fmt.Sprintf("%s%s", res[0][2], res[0][3])
		bsize, _ := bytefmt.ToBytes(hsize)
		hb := fmt.Sprintf("size:%s%d", res[0][1], bsize)
		q = r.ReplaceAllString(q, hb)
	}

	r = regexp.MustCompile(`\+|:|-|"`)
	if !r.Match([]byte(q)) {
		tokens := strings.Split(q, " ")
		if len(tokens) > 1 {
			nq := []string{}
			for _, t := range tokens {
				nq = append(nq, fmt.Sprintf("+%s", t))
			}
			log.Print("searching for ", nq)
			return strings.Join(nq, " ")
		}
	}

	if q, yes := needsReplace(q, `type:ebook`, `ext:fb2 ext:ibook ext:cbr ext:djvu ext:epub ext:mobi`); yes {
		return q
	}

	if q, yes := needsReplace(q, `type:document`, `ext:doc ext:docm ext:pdf ext:docx ext:odf ext:pages ext:rtf`); yes {
		return q
	}

	if q, yes := needsReplace(q, `type:image`, `ext:jp(e)?g ext:png ext:gif ext:tiff ext:eps ext:raw`); yes {
		return q
	}

	if q, yes := needsReplace(q, `type:audio`, `ext:wav ext:mp3 ext:ogg ext:flac`); yes {
		return q
	}

	if q, yes := needsReplace(q, `type:video`, `ext:webm ext:mov ext:avi ext:mkv ext:mp4`); yes {
		return q
	}

	log.Print("queries: searching for ", q)
	return q
}

func needsReplace(q, re, replacement string) (string, bool) {
	r := regexp.MustCompile(re)
	if r.Match([]byte(q)) {
		nq := r.ReplaceAllString(q, replacement)
		log.Debug().Msgf("replaced '%s' with '%s'", q, nq)
		return nq, true
	}
	return q, false
}
