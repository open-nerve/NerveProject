package content

import (
	"strings"
	"sync"
	"testing"
)

func TestCleanCustomURLs(t *testing.T) {
	for _, tag := range []struct{ open, attribute string }{{"image-component", "src"}, {"div", "data-emoji-url"}} {
		for _, tc := range []struct{ value, want string }{
			{"javascript:alert(1)", ""}, {"JaVaScRiPt:alert(1)", ""}, {"java&#10;script:alert(1)", ""},
			{" data:text/html,boom ", ""}, {"vbscript:boom", ""}, {"&#106;avascript:alert(1)", ""},
			{"https://example.test/a&quot; onerror=&quot;boom", ""}, {"https://example.test/&#10;a", ""},
			{"  https://example.test/image.png  ", "https://example.test/image.png"},
			{"https://example.test/image.png?a=1&amp;b=2", "https://example.test/image.png?a=1&amp;b=2"},
			{"asset-1", "asset-1"}, {"mailto:ada@example.test", "mailto:ada@example.test"}, {"tel:+1234", "tel:+1234"},
		} {
			t.Run(tag.open+"/"+tc.value, func(t *testing.T) {
				got := NewSanitizer().Clean(`<` + tag.open + ` ` + tag.attribute + `="` + tc.value + `">safe</` + tag.open + `>`).HTML
				if tc.want == "" {
					if strings.Contains(got, tag.attribute+"=") {
						t.Fatalf("unsafe custom URL survived: %s", got)
					}
				} else if !strings.Contains(got, tag.attribute+`="`+tc.want+`"`) {
					t.Fatalf("safe URL lost: %s", got)
				}
			})
		}
	}
}

func TestCleanPolicyPreservesEditorCSS(t *testing.T) {
	input := `<p style="text-align: center; color: rgb(255, 0, 0); background-color: #fff; position:fixed">safe</p>`
	want := `<p style="text-align: center; color: rgb(255, 0, 0); background-color: #fff">safe</p>`
	if got := NewSanitizer().Clean(input).HTML; got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestCleanConcurrent(t *testing.T) {
	sanitizer := NewSanitizer()
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 20 {
				if got := sanitizer.Clean(`<p onerror="boom">hello</p>`); got.HTML != "<p>hello</p>" || got.Text != "hello" {
					t.Errorf("unsafe concurrent result: %+v", got)
				}
			}
		})
	}
	wg.Wait()
}

func TestCleanTaskLabelAndLinkAttributes(t *testing.T) {
	input := `<label><input type="checkbox" checked="checked"><span></span></label><a href="https://example.test" target="_blank" rel="noopener noreferrer nofollow">link</a>`
	if got := NewSanitizer().Clean(input).HTML; got != input {
		t.Fatalf("editor attributes lost: %s", got)
	}
}

func TestCleanColorAttributesCannotIntroduceCSS(t *testing.T) {
	for _, attribute := range []string{"data-text-color", "data-background-color", "background", "textcolor"} {
		input := `<span ` + attribute + `="red; position: fixed; background-image: url(https://example.test/pixel)">Color</span>`
		if attribute == "background" || attribute == "textcolor" {
			input = `<table><tbody><tr><td ` + attribute + `="red; position: fixed; background-image: url(https://example.test/pixel)">Color</td></tr></tbody></table>`
		}
		if got := NewSanitizer().Clean(input).HTML; strings.Contains(got, "position") || strings.Contains(got, "url(") {
			t.Errorf("%s can introduce CSS: %s", attribute, got)
		}
	}
}

func TestCleanPlainText(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"<p> A &amp; B&nbsp;\n  你好 </p>", "A & B 你好"},
		{`<p>&lt;safe&gt; &amp;lt;</p>`, `<safe> &lt;`},
		{"<p> \t\n&nbsp; </p>", ""},
	} {
		if got := NewSanitizer().Clean(tc.input).Text; got != tc.want {
			t.Errorf("text=%q want %q", got, tc.want)
		}
	}
}
