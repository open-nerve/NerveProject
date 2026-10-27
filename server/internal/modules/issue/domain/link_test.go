package domain_test

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Prefix-only scheme detection, case folding and byte length limits each
// change one of these independently specified accepted or rejected values.
func TestLinkURLNormalization(t *testing.T) {
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
			got, err := domain.CheckNewLink(domain.NewLink{URL: tc.input})
			if err != nil || got.URL != tc.want {
				t.Fatalf("normalized URL = %q, %v; want %q", got.URL, err, tc.want)
			}
			original := tc.input
			patch, err := domain.CheckLinkPatch(domain.LinkPatch{URL: &tc.input})
			if err != nil || patch.URL == nil || *patch.URL != tc.want || tc.input != original {
				t.Fatalf("normalized patch = %#v, %v; want %q", patch, err, tc.want)
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
			_, err := domain.CheckNewLink(domain.NewLink{URL: input})
			wantLinkFields(t, err, [2]string{"url", shared.FieldNotAllowed})
			_, err = domain.CheckLinkPatch(domain.LinkPatch{URL: &input})
			wantLinkFields(t, err, [2]string{"url", shared.FieldNotAllowed})
		})
	}
}

// Losing SetTitle conflates omission with an explicit null; trimming title
// or counting its bytes also changes a caller-visible value.
func TestLinkTitlePatch(t *testing.T) {
	for _, title := range []*string{nil, linkString(""), linkString("  Keep me  "), linkString(strings.Repeat("界", 255))} {
		got, err := domain.CheckNewLink(domain.NewLink{URL: "example.com", Title: title})
		if err != nil || !reflect.DeepEqual(got.Title, title) {
			t.Fatalf("create title = %v, %v; want %v", got.Title, err, title)
		}
		for _, set := range []bool{false, true} {
			input := domain.LinkPatch{SetTitle: set, Title: title}
			got, err := domain.CheckLinkPatch(input)
			if err != nil || !reflect.DeepEqual(got, input) {
				t.Fatalf("patch = %#v, %v; want %#v", got, err, input)
			}
		}
	}
	for _, tc := range []struct{ title, code string }{{strings.Repeat("界", 256), shared.FieldTooLong}, {"a\x00b", shared.FieldInvalidFormat}} {
		_, err := domain.CheckNewLink(domain.NewLink{URL: "example.com", Title: &tc.title})
		wantLinkFields(t, err, [2]string{"title", tc.code})
		_, err = domain.CheckLinkPatch(domain.LinkPatch{SetTitle: true, Title: &tc.title})
		wantLinkFields(t, err, [2]string{"title", tc.code})
	}
	badURL, badTitle := "ftp://example.com", strings.Repeat("界", 256)
	_, err := domain.CheckNewLink(domain.NewLink{URL: badURL, Title: &badTitle})
	wantLinkFields(t, err, [2]string{"url", shared.FieldNotAllowed}, [2]string{"title", shared.FieldTooLong})
	_, err = domain.CheckLinkPatch(domain.LinkPatch{URL: &badURL, SetTitle: true, Title: &badTitle})
	wantLinkFields(t, err, [2]string{"url", shared.FieldNotAllowed}, [2]string{"title", shared.FieldTooLong})
}

func linkString(s string) *string { return &s }

func wantLinkFields(t *testing.T, err error, want ...[2]string) {
	t.Helper()
	var problem *shared.Error
	if !errors.As(err, &problem) || problem.Kind != shared.KindInvalid || problem.Code != shared.CodeValidationFailed || len(problem.Fields) != len(want) {
		t.Fatalf("problem = %#v (%v); want fields %v", problem, err, want)
	}
	for n, field := range problem.Fields {
		if field.Field != want[n][0] || field.Code != want[n][1] {
			t.Fatalf("field = %#v; want %v", field, want[n])
		}
	}
}
