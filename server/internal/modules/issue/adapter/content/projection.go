package content

import (
	stdhtml "html"
	"strings"
	"uuid"

	"golang.org/x/net/html"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func (s *Sanitizer) project(clean string) domain.RichText {
	text := strings.Join(strings.Fields(stdhtml.UnescapeString(s.textPolicy.Sanitize(clean))), " ")
	result := domain.RichText{HTML: clean, Text: text, Identity: domain.ContentIdentity{Text: text, MentionIDs: []uuid.UUID{}, Links: []string{}, Checked: []bool{}}}
	root, err := html.Parse(strings.NewReader(clean))
	if err != nil {
		return result
	}
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode {
			attrs := make(map[string]string, len(node.Attr))
			for _, attr := range node.Attr {
				attrs[attr.Key] = attr.Val
			}
			switch node.Data {
			case "mention-component":
				if attrs["entity_name"] == "user_mention" {
					if id, err := uuid.Parse(attrs["entity_identifier"]); err == nil {
						result.Identity.MentionIDs = append(result.Identity.MentionIDs, id)
						result.HasMention = true
					}
				}
			case "img", "image-component":
				result.HasImage = true
			case "a":
				if href, ok := attrs["href"]; ok {
					result.Identity.Links = append(result.Identity.Links, href)
				}
			case "input":
				_, checked := attrs["checked"]
				result.Identity.Checked = append(result.Identity.Checked, checked)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	return result
}
