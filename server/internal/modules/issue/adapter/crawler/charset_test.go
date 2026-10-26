package crawler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCrawlerPageCharset(t *testing.T) {
	const gbkPage = "<meta charset=gbk><title>\xd6\xd0\xce\xc4</title>"
	cases := []struct {
		name, contentType, body, title, icon string
	}{
		{"header_latin1", "text/html; charset=iso-8859-1", "<title>caf\xe9</title>", "café", "/favicon.ico"},
		{"meta_gbk", "text/html", "<meta charset=gbk><title>\xd6\xd0\xce\xc4</title>", "中文", "/favicon.ico"},
		{"meta_pragma_gbk", "text/html", "<meta http-equiv='Content-Type' content='text/html; charset=gbk'><title>\xd6\xd0\xce\xc4</title>", "中文", "/favicon.ico"},
		{"bom_before_header_and_meta", "text/html; charset=iso-8859-1", "\xef\xbb\xbf<meta charset=gbk><title>中文</title>", "中文", "/favicon.ico"},
		{"header_before_meta", "text/html; charset=iso-8859-1", "<meta charset=gbk><title>caf\xe9</title>", "café", "/favicon.ico"},
		{"declared_header_before_valid_utf8", "text/html; charset=windows-1252", "<title>\xc3\xa9</title>", "Ã©", "/favicon.ico"},
		{"declared_meta_before_valid_utf8", "text/html", "<meta charset=windows-1252><title>\xc3\xa9</title>", "Ã©", "/favicon.ico"},
		{"meta_pragma_valid_utf8", "text/html", "<meta http-equiv='CONTENT-TYPE' content='text/html; charset=windows-1252'>" + "<title>\xc3\xa9</title>", "Ã©", "/favicon.ico"},
		{"meta_pragma_quoted", "text/html", "<meta content=\"text/html; charset='windows-1252'\" http-equiv=content-type>" + "<title>\xc3\xa9</title>", "Ã©", "/favicon.ico"},
		{"content_without_pragma", "text/html", "<meta content='text/html; charset=windows-1252'>" + "<title>\xc3\xa9</title>", "é", "/favicon.ico"},
		{"invalid_charset", "text/html", "<meta charset=unknown>" + "<title>\xc3\xa9</title>", "é", "/favicon.ico"},
		{"duplicate_charset_first_invalid", "text/html", "<meta charset=unknown charset=windows-1252>" + "<title>\xc3\xa9</title>", "é", "/favicon.ico"},
		{"duplicate_charset_first_valid", "text/html", "<meta charset=windows-1252 charset=unknown>" + "<title>\xc3\xa9</title>", "Ã©", "/favicon.ico"},
		{"invalid_charset_overrides_content", "text/html", "<meta http-equiv=content-type content='text/html; charset=windows-1252' charset=unknown>" + "<title>\xc3\xa9</title>", "é", "/favicon.ico"},
		{"content_after_invalid_charset", "text/html", "<meta http-equiv=content-type charset=unknown content='text/html; charset=windows-1252'>" + "<title>\xc3\xa9</title>", "Ã©", "/favicon.ico"},
		{"duplicate_pragma_first_invalid", "text/html", "<meta http-equiv=other http-equiv=content-type content='text/html; charset=windows-1252'>" + "<title>\xc3\xa9</title>", "é", "/favicon.ico"},
		{"content_scans_past_nonassignment", "text/html", "<meta http-equiv=content-type content='charset nonsense; charset = windows-1252 ; x'>" + "<title>\xc3\xa9</title>", "Ã©", "/favicon.ico"},
		{"unterminated_content_quote", "text/html", "<meta http-equiv=content-type content=\"charset='windows-1252\">" + "<title>\xc3\xa9</title>", "é", "/favicon.ico"},
		{"late_declaration_ignored", "text/html", strings.Repeat(" ", 1024) + "<meta charset=windows-1252>" + "<title>\xc3\xa9</title>", "é", "/favicon.ico"},
		{"undeclared_utf8", "text/html", "<title>中文🙂</title>", "中文🙂", "/favicon.ico"},
		{"late_undeclared_utf8", "text/html", strings.Repeat(" ", 1100) + "<title>中文🙂</title>", "中文🙂", "/favicon.ico"},
		{"gbk_icon_href", "text/html", "<meta charset=gbk><title>\xd6\xd0\xce\xc4</title><link rel=icon href='/\xd6\xd0\xce\xc4.png'>", "中文", "/%E4%B8%AD%E6%96%87.png"},
		{"entities_once", "text/html; charset=iso-8859-1", "<title> caf\xe9 &amp;amp; </title>", "café &amp;", "/favicon.ico"},
		{"decoded_rune_limit", "text/html", "<meta charset=gbk><title> " + strings.Repeat("\xd6\xd0", 256) + " </title>", strings.Repeat("中", 255), "/favicon.ico"},
		{"body_at_limit", "text/html", gbkPage + strings.Repeat(" ", (1<<20)-len(gbkPage)), "中文", "/favicon.ico"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/page" {
					response(w, 200, tc.contentType, tc.body)
					return
				}
				if r.URL.RequestURI() != tc.icon {
					t.Errorf("icon request=%q want %q", r.URL.RequestURI(), tc.icon)
				}
				response(w, 200, "image/png", "icon")
			}))
			defer s.Close()
			target := s.URL + "/page"
			got := newCrawler(t).Crawl(t.Context(), target)
			want := success(target, value(tc.title))
			want["favicon"], want["favicon_url"] = value("data:image/png;base64,aWNvbg=="), value(s.URL+tc.icon)
			wantMetadata(t, got, want)
			if got["title"] == nil || !utf8.ValidString(*got["title"]) {
				t.Fatal("title must be valid UTF-8")
			}
			encoded, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			var decoded map[string]*string
			if err := json.Unmarshal(encoded, &decoded); err != nil || decoded["title"] == nil || *decoded["title"] != tc.title {
				t.Errorf("JSON title=%v error=%v", decoded["title"], err)
			}
		})
	}
}
