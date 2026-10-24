package content

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/microcosm-cc/bluemonday"
)

func editorPolicy(color *regexp.Regexp) *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("a", "abbr", "b", "blockquote", "br", "caption", "code", "col", "colgroup", "dd", "del", "details", "div", "dl", "dt", "em", "figcaption", "figure", "h1", "h2", "h3", "h4", "h5", "h6", "hr", "i", "img", "ins", "kbd", "li", "mark", "ol", "p", "pre", "s", "small", "span", "strike", "strong", "sub", "summary", "sup", "table", "tbody", "td", "th", "thead", "tfoot", "tr", "u", "ul", "mention-component", "image-component", "label", "input")
	p.AllowAttrs("class", "id", "title", "role", "aria-label", "aria-hidden", "start", "type", "xmlns", "data-tight", "data-node-type", "data-type", "data-checked", "data-background-color", "data-text-color", "data-name", "data-id").Globally()
	p.AllowAttrs("data-icon-name", "data-icon-color", "data-background", "data-emoji-unicode", "data-emoji-url", "data-logo-in-use", "data-block-type").OnElements("div")
	p.AllowNoAttrs().OnElements("label")
	p.AllowAttrs("href", "target", "rel").OnElements("a")
	p.AllowAttrs("width", "height", "aspectratio", "src", "alignment", "status", "alt", "title").OnElements("image-component", "img")
	p.AllowAttrs("entity_identifier", "entity_name").OnElements("mention-component")
	p.AllowAttrs("colspan", "rowspan", "colwidth", "background", "textcolor").OnElements("th", "td")
	p.AllowAttrs("background", "textcolor").OnElements("tr")
	p.AllowAttrs("language").OnElements("pre", "code")
	p.AllowAttrs("spellcheck").OnElements("code")
	p.AllowAttrs("checked").OnElements("input")
	p.AllowAttrs("style").OnElements("p", "h1", "h2", "h3", "h4", "h5", "h6", "span", "table", "tr", "td", "th", "col")
	p.AllowStyles("text-align").Matching(regexp.MustCompile(`^(left|right|center|justify)$`)).Globally()
	p.AllowStyles("color", "background-color").Matching(color).Globally()
	p.AllowStyles("width", "min-width").Matching(regexp.MustCompile(`^([0-9]+(\.[0-9]+)?(px|%)|auto)$`)).OnElements("table", "col", "td", "th")
	p.RequireParseableURLs(true)
	p.AllowRelativeURLs(true)
	p.AllowURLSchemes("http", "https", "mailto", "tel")
	return p
}

// safeURL also covers URL-bearing attributes not recognized by bluemonday.
func safeURL(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, `"'<>\`) {
		return "", false
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return "", false
		}
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "", "http", "https", "mailto", "tel":
	default:
		return "", false
	}
	return parsed.String(), true
}

// colorPattern is shared by CSS declarations and attributes consumed as CSS by the editor.
func colorPattern() *regexp.Regexp {
	return regexp.MustCompile(`(?i)^(#([0-9a-f]{3}|[0-9a-f]{4}|[0-9a-f]{6}|[0-9a-f]{8})|[a-z]+|rgba?\([0-9.,% /]+\)|hsla?\([0-9.,% /]+\))$`)
}

func editorColorAttribute(tag, key string) bool {
	return key == "data-text-color" || key == "data-background-color" || ((tag == "tr" || tag == "td" || tag == "th") && (key == "background" || key == "textcolor"))
}
