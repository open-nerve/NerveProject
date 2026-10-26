package crawler

import (
	"strings"

	"golang.org/x/net/html"
)

func pageMetadata(body []byte, contentType string) (*string, string, error) {
	// HTML parsing decodes entities exactly once and rejects excessive nesting.
	document, err := html.Parse(pageReader(body, contentType))
	if err != nil {
		return nil, "", err
	}
	var title *string
	var href string
	titleFound, bestRank := false, 4
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode {
			if node.Data == "title" && !titleFound {
				titleFound = true
				text := strings.TrimSpace(nodeText(node))
				runes := []rune(text)
				if len(runes) > 255 {
					text = string(runes[:255])
				}
				if text != "" {
					title = &text
				}
			}
			if node.Data == "link" {
				rel, candidate := "", ""
				for _, attr := range node.Attr {
					switch attr.Key {
					case "rel":
						rel = attr.Val
					case "href":
						candidate = strings.TrimSpace(attr.Val)
					}
				}
				if rank := iconRank(rel); candidate != "" && rank < bestRank {
					bestRank, href = rank, candidate
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(document)
	return title, href, nil
}

func nodeText(node *html.Node) string {
	var text strings.Builder
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(node)
	return text.String()
}

func iconRank(rel string) int {
	icon, shortcut, apple, precomposed := false, false, false, false
	for _, token := range strings.FieldsFunc(strings.ToLower(rel), func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f'
	}) {
		switch token {
		case "icon":
			icon = true
		case "shortcut":
			shortcut = true
		case "apple-touch-icon":
			apple = true
		case "apple-touch-icon-precomposed":
			precomposed = true
		}
	}
	switch {
	case icon && !shortcut:
		return 0
	case icon:
		return 1
	case apple:
		return 2
	case precomposed:
		return 3
	default:
		return 4
	}
}
