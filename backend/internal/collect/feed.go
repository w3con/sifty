package collect

import (
	"encoding/xml"
	"net/url"
	"strings"
	"time"
)

type feedItem struct {
	fields    map[string]string // local element name -> text (first occurrence kept)
	linkHref  string
	pubDate   string // RSS pubDate: a genuine publication timestamp
	published string // Atom published: a genuine publication timestamp
}

// parseFeedItems tokenises an RSS or Atom document and returns each item/entry with its direct
// child element texts, the link's href attribute, and the first publication timestamp seen. It
// mirrors cheerio's xml-mode selection: item OR entry, and child elements matched by local name.
func parseFeedItems(xmlText string) []feedItem {
	dec := xml.NewDecoder(strings.NewReader(xmlText))
	dec.Strict = false
	dec.Entity = xml.HTMLEntity

	var items []feedItem
	var cur *feedItem
	itemDepth := 0
	depth := 0
	field := ""
	var buf strings.Builder

	flushField := func() {
		if cur == nil || field == "" {
			return
		}
		val := buf.String()
		if _, exists := cur.fields[field]; !exists {
			cur.fields[field] = val
		}
		// updated is a modification timestamp, never a publication date; it is
		// deliberately not captured as one.
		if field == "pubDate" && cur.pubDate == "" {
			cur.pubDate = val
		}
		if field == "published" && cur.published == "" {
			cur.published = val
		}
		field = ""
		buf.Reset()
	}

	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			local := t.Name.Local
			if cur == nil && (local == "item" || local == "entry") {
				cur = &feedItem{fields: map[string]string{}}
				itemDepth = depth
				continue
			}
			if cur != nil && depth == itemDepth+1 {
				field = local
				buf.Reset()
				if local == "link" {
					for _, a := range t.Attr {
						if a.Name.Local == "href" {
							cur.linkHref = a.Value
						}
					}
				}
			}
		case xml.CharData:
			if cur != nil && field != "" {
				buf.Write(t)
			}
		case xml.EndElement:
			if cur != nil && depth == itemDepth+1 {
				flushField()
			}
			if cur != nil && depth == itemDepth {
				items = append(items, *cur)
				cur = nil
			}
			depth--
		}
	}
	return items
}

type hrefParts struct {
	host string
	path string
}

func parseHref(raw string) (hrefParts, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return hrefParts{}, err
	}
	return hrefParts{host: u.Hostname(), path: u.Path}, nil
}

// epochToISO converts a JSON number of Unix seconds to the millisecond ISO string the TypeScript
// application produces (Date.toISOString()); a missing value yields the empty string.
func epochToISO(x any) string {
	sec, ok := x.(float64)
	if !ok {
		return ""
	}
	return time.UnixMilli(int64(sec * 1000)).UTC().Format("2006-01-02T15:04:05.000Z07:00")
}
