package content

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type editorFixture struct {
	Name         string   `json:"name"`
	Input        string   `json:"input"`
	Sanitized    string   `json:"sanitized"`
	Reserialized string   `json:"reserialized"`
	Resanitized  string   `json:"resanitized"`
	Text         string   `json:"expected_text"`
	Mentions     []string `json:"expected_mentions"`
	Links        []string `json:"expected_links"`
	Checked      []bool   `json:"expected_checked"`
}

func TestCleanEditorHTML(t *testing.T) {
	data, err := os.ReadFile("../../../../../../api/testdata/issue-richtext.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures struct {
		Cases []editorFixture `json:"cases"`
	}
	if err = json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	sanitizer := NewSanitizer()
	for _, tc := range fixtures.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			for _, edge := range []struct{ input, want string }{{tc.Input, tc.Sanitized}, {tc.Reserialized, tc.Resanitized}} {
				got := sanitizer.Clean(edge.input)
				if got.HTML != edge.want {
					t.Errorf("HTML\ngot  %s\nwant %s", got.HTML, edge.want)
				}
				if got.Text != tc.Text {
					t.Errorf("text=%q want %q", got.Text, tc.Text)
				}
				if got.Identity.Text != tc.Text {
					t.Errorf("identity text=%q want %q", got.Identity.Text, tc.Text)
				}
				mentions := make([]string, 0, len(got.Identity.MentionIDs))
				for _, id := range got.Identity.MentionIDs {
					mentions = append(mentions, id.String())
				}
				if !reflect.DeepEqual(mentions, tc.Mentions) {
					t.Errorf("mentions=%v want %v", mentions, tc.Mentions)
				}
				if !reflect.DeepEqual(got.Identity.Links, tc.Links) {
					t.Errorf("links=%v want %v", got.Identity.Links, tc.Links)
				}
				if !reflect.DeepEqual(got.Identity.Checked, tc.Checked) {
					t.Errorf("checked=%v want %v", got.Identity.Checked, tc.Checked)
				}
			}
		})
	}
}

func TestCleanUnsafeHTML(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"event", `<p onerror="alert(1)">safe</p>`, `<p>safe</p>`},
		{"script", `<script>alert(1)</script><p>safe</p>`, `<p>safe</p>`},
		{"javascript link", `<a href="javascript:alert(1)">safe</a>`, `safe`},
		{"dangerous CSS", `<p style="background-image:url(javascript:alert(1)); color:expression(alert(1)); position:fixed">safe</p>`, `<p>safe</p>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := NewSanitizer().Clean(tc.input).HTML; got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}
