package shared_test

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Prefix-only scheme detection, case folding and byte length limits each
// change one of these independently specified accepted or rejected values.
func TestCheckLinkURL(t *testing.T) {
	r := rand.New(rand.NewPCG(71, 27))
	long := []rune("http://169.254.169.254/")
	for len(long) < 2048 {
		long = append(long, rune(0x10000+r.IntN(0x100000)))
	}
	if utf8.RuneCountInString(string(long)) != 2048 || len(string(long)) <= 2704 {
		t.Fatal("need a high-entropy multibyte 2048-character fixture")
	}
	for _, tc := range []struct{ name, input, want string }{
		{"case", "  HTTP://Example.com/P?q=A  ", "HTTP://Example.com/P?q=A"},
		{"https", "hTtPs://Example.com:443/P#X", "hTtPs://Example.com:443/P#X"},
		{"bare", " example.com/p?next=ftp://x ", "http://example.com/p?next=ftp://x"},
		{"path scheme", "example.com/path/ftp://x", "http://example.com/path/ftp://x"},
		{"bare port", "127.0.0.1:4321/page", "http://127.0.0.1:4321/page"},
		{"zero port", "http://example.com:0", "http://example.com:0"},
		{"max port", "http://example.com:65535", "http://example.com:65535"},
		{"IPv6", "http://[::1]:4321/page", "http://[::1]:4321/page"},
		{"unicode", "http://例子.测试/路径", "http://例子.测试/路径"},
		{"2048 characters", string(long), string(long)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, problem := shared.CheckLinkURL(tc.input)
			if problem != nil || got != tc.want {
				t.Fatalf("normalized URL = %q, %#v; want %q", got, problem, tc.want)
			}
		})
	}
	for n, input := range []string{
		"", " ", "ftp://example.com", "git+ssh://example.com", "a.b-c://example.com",
		"http://", "http:///path", "http://:80", "http://u:p@example.com", "http://@example.com",
		"http://[fe80::1%25eth0]/", "http://example.com/\x00x", "http://example.com/\nx",
		"http://example.com/\u007fx", "http://example.com:65536", "http://example.com:-1",
		"http://example.com:abc", "http://example.com:", "http://[bad]/", string(long) + "界",
	} {
		t.Run(fmt.Sprintf("reject-%02d", n), func(t *testing.T) {
			_, problem := shared.CheckLinkURL(input)
			wantLinkField(t, problem, "url", shared.FieldNotAllowed)
		})
	}
}

// Trimming a title or counting its bytes changes a caller-visible value.
func TestCheckLinkTitle(t *testing.T) {
	for _, title := range []*string{nil, linkString(""), linkString("  Keep me  "), linkString(strings.Repeat("界", 255))} {
		if problem := shared.CheckLinkTitle(title); problem != nil {
			t.Fatalf("title %v refused: %#v", title, problem)
		}
	}
	for _, tc := range []struct{ title, code string }{{strings.Repeat("界", 256), shared.FieldTooLong}, {"a\x00b", shared.FieldInvalidFormat}} {
		wantLinkField(t, shared.CheckLinkTitle(&tc.title), "title", tc.code)
	}
}

// Pending metadata is an object; absent keys, nulls and string values remain
// distinct across the JSON boundary used by storage and the response.
func TestLinkMetadata(t *testing.T) {
	for _, raw := range []string{`{}`, `{"favicon":null,"favicon_url":null,"title":"Title","url":"http://example.com"}`, `{"error":"blocked","url":"http://example.com"}`} {
		var got shared.LinkMetadata
		if err := json.Unmarshal([]byte(raw), &got); err != nil || got == nil {
			t.Fatalf("metadata decode = %#v, %v", got, err)
		}
		encoded, err := json.Marshal(got)
		if err != nil || string(encoded) != raw {
			t.Fatalf("metadata JSON = %s, %v; want %s", encoded, err, raw)
		}
		var again shared.LinkMetadata
		if err := json.Unmarshal(encoded, &again); err != nil || !reflect.DeepEqual(got, again) {
			t.Fatalf("metadata round trip = %#v, %v", again, err)
		}
	}
	for _, raw := range []string{`{"title":17}`, `{"favicon":false}`, `{"url":[]}`, `{"error":{}}`} {
		var got shared.LinkMetadata
		if err := json.Unmarshal([]byte(raw), &got); err == nil {
			t.Fatalf("accepted metadata value %s", raw)
		}
	}
}

func linkString(s string) *string { return &s }

func wantLinkField(t *testing.T, problem *shared.FieldError, field, code string) {
	t.Helper()
	if problem == nil || problem.Field != field || problem.Code != code {
		t.Fatalf("problem = %#v; want %s %s", problem, field, code)
	}
}
