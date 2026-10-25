package content

import (
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

var _ app.HTMLSanitizer = (*Sanitizer)(nil)

// Sanitizer owns policies built once and read concurrently thereafter.
type Sanitizer struct {
	policy, textPolicy *bluemonday.Policy
	color              *regexp.Regexp
}

func NewSanitizer() *Sanitizer {
	color := colorPattern()
	return &Sanitizer{policy: editorPolicy(color), textPolicy: bluemonday.StrictPolicy(), color: color}
}

func (s *Sanitizer) Clean(input string) domain.RichText {
	tokenizer := html.NewTokenizer(strings.NewReader(s.policy.Sanitize(input)))
	var output strings.Builder
	for {
		kind := tokenizer.Next()
		if kind == html.ErrorToken {
			break
		}
		token := tokenizer.Token()
		if kind == html.StartTagToken || kind == html.SelfClosingTagToken {
			attrs := token.Attr[:0]
			for _, attr := range token.Attr {
				if (token.Data == "image-component" && attr.Key == "src") || (token.Data == "div" && attr.Key == "data-emoji-url") {
					value, ok := safeURL(attr.Val)
					if !ok {
						continue
					}
					attr.Val = value
				}
				if editorColorAttribute(token.Data, attr.Key) {
					attr.Val = strings.TrimSpace(attr.Val)
					if !s.color.MatchString(attr.Val) && attr.Val != "light-blue" && attr.Val != "dark-blue" {
						continue
					}
				}
				attrs = append(attrs, attr)
			}
			token.Attr = attrs
		}
		output.WriteString(token.String())
	}
	return s.project(output.String())
}
