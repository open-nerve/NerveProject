package crawler

import (
	"bytes"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

func pageReader(body []byte, contentType string) io.Reader {
	encoding, name, certain := charset.DetermineEncoding(body, contentType)
	// The library only sniffs 1024 bytes and defaults to Windows-1252. Keep
	// unlabelled UTF-8 even when its first non-ASCII rune is beyond that prefix,
	// without overriding an explicit BOM, HTTP header or valid meta declaration.
	if !certain && name == "windows-1252" && utf8.Valid(body) && !declaredMetaCharset(body[:min(len(body), 1024)]) {
		return bytes.NewReader(body)
	}
	return encoding.NewDecoder().Reader(bytes.NewReader(body))
}

// declaredMetaCharset only distinguishes a declaration from the library's
// fallback. Encoding selection and decoding remain in html/charset.
func declaredMetaCharset(prefix []byte) bool {
	z := html.NewTokenizer(bytes.NewReader(prefix))
	for {
		switch z.Next() {
		case html.ErrorToken:
			return false
		case html.StartTagToken, html.SelfClosingTagToken:
			tag := z.Token()
			if tag.Data != "meta" {
				continue
			}
			seen := make(map[string]bool)
			valid, pragma, needsPragma := false, false, false
			for _, attr := range tag.Attr {
				if seen[attr.Key] {
					continue
				}
				seen[attr.Key] = true
				value := strings.Map(func(r rune) rune {
					if r >= 'A' && r <= 'Z' {
						return r + ('a' - 'A')
					}
					return r
				}, attr.Val)
				switch attr.Key {
				case "http-equiv":
					pragma = value == "content-type"
				case "charset":
					encoding, _ := charset.Lookup(value)
					valid, needsPragma = encoding != nil, false
				case "content":
					if !valid {
						encoding, _ := charset.Lookup(contentCharset(value))
						if encoding != nil {
							valid, needsPragma = true, true
						}
					}
				}
			}
			if valid && (!needsPragma || pragma) {
				return true
			}
		}
	}
}

func contentCharset(content string) string {
	const spaces = " \t\n\f\r"
	for {
		_, rest, found := strings.Cut(content, "charset")
		if !found {
			return ""
		}
		content = strings.TrimLeft(rest, spaces)
		if !strings.HasPrefix(content, "=") {
			continue
		}
		value := strings.TrimLeft(content[1:], spaces)
		if value == "" {
			return ""
		}
		if value[0] == '\'' || value[0] == '"' {
			label, _, closed := strings.Cut(value[1:], value[:1])
			if !closed {
				return ""
			}
			return label
		}
		if end := strings.IndexAny(value, spaces+";"); end >= 0 {
			value = value[:end]
		}
		return value
	}
}
