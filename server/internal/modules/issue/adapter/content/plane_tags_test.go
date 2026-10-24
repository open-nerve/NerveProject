package content

import (
	"strings"
	"testing"
)

// These expectations are pinned independently to Plane's nh3 0.2.18 / ammonia 4.0.0 defaults.
func TestCleanPlaneDefaultTags(t *testing.T) {
	for _, tag := range strings.Fields(`a abbr acronym article aside b bdi bdo blockquote caption center cite code data dd del details dfn div dl dt em figcaption figure footer h1 h2 h3 h4 h5 h6 header hgroup i ins kbd li map mark nav ol p pre q rp rt rtc ruby s samp small span strike strong sub summary sup table tbody td th thead time tr tt u ul var`) {
		t.Run(tag, func(t *testing.T) {
			input := `<` + tag + `>kept</` + tag + `>`
			switch tag {
			case "a":
				input = `<a href="https://example.test">kept</a>`
			case "caption":
				input = `<table><` + tag + `>kept</` + tag + `></table>`
			case "table":
				input = `<table><tbody><tr><td>kept</td></tr></tbody></table>`
			case "thead", "tbody":
				input = `<table><` + tag + `><tr><td>kept</td></tr></` + tag + `></table>`
			case "ol", "ul":
				input = `<` + tag + `><li>kept</li></` + tag + `>`
			case "dl":
				input = `<dl><dt>kept</dt></dl>`
			case "td", "th":
				input = `<table><tbody><tr><` + tag + `>kept</` + tag + `></tr></tbody></table>`
			case "tr":
				input = `<table><tbody><tr><td>kept</td></tr></tbody></table>`
			case "li":
				input = `<ul><li>kept</li></ul>`
			case "dd", "dt":
				input = `<dl><` + tag + `>kept</` + tag + `></dl>`
			case "hgroup":
				input = `<hgroup><h1>kept</h1></hgroup>`
			case "figcaption":
				input = `<figure><figcaption>kept</figcaption></figure>`
			case "summary":
				input = `<details><summary>kept</summary></details>`
			case "time":
				input = `<time>2026-10-25</time>`
			case "rp", "rt":
				input = `<ruby>字<rp>(</rp><rt>kept</rt><rp>)</rp></ruby>`
			case "rtc":
				input = `<ruby>字<rtc><rt>kept</rt></rtc></ruby>`
			}
			if got := NewSanitizer().Clean(input).HTML; got != input {
				t.Fatalf("tag %s or its text lost: got %q want %q", tag, got, input)
			}
		})
	}
	for _, input := range []string{
		`<p>kept<br>tail<wbr></p>`, `<hr>`, `<map><area></map>`,
		`<img src="https://example.test/image">`, `<table><colgroup><col></colgroup><tbody><tr><td>kept</td></tr></tbody></table>`,
	} {
		if got := NewSanitizer().Clean(input).HTML; got != input {
			t.Errorf("empty element or table structure lost: got %q want %q", got, input)
		}
	}
}

func TestCleanPlaneUnlistedTags(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{`<unknown>kept</unknown>`, `kept`},
		{`<script>unsafe</script><p>kept</p>`, `<p>kept</p>`},
		{`<table><tfoot><tr><td>kept</td></tr></tfoot></table>`, `<table><tr><td>kept</td></tr></table>`},
	} {
		if got := NewSanitizer().Clean(tc.input).HTML; got != tc.want {
			t.Errorf("unlisted tag survived: got %q want %q", got, tc.want)
		}
	}
}
