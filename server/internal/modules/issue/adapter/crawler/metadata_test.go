package crawler_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestCrawlerMetadata(t *testing.T) {
	cases := []struct {
		name, html string
		title      *string
		icon       string
	}{
		{"entities_once", "<title> \n &amp;amp; \t </title>", value("&amp;"), "/favicon.ico"},
		{"unicode_limit", "<title> \t" + strings.Repeat("界🙂", 128) + " </title>", value(strings.Repeat("界🙂", 127) + "界"), "/favicon.ico"},
		{"missing_title", "<p>content</p>", nil, "/favicon.ico"},
		{"empty_title", "<title> \n\t </title><title>second</title>", nil, "/favicon.ico"},
		{"first_title", "<title>first</title><title>second</title>", value("first"), "/favicon.ico"},
		{"icon_priority", `<link rel="apple-touch-icon-precomposed" href="/pre"><link rel="apple-touch-icon" href="/apple"><link rel="shortcut icon" href="/shortcut"><link rel="icon" href="/icon">`, nil, "/icon"},
		{"shortcut_priority", `<link rel="apple-touch-icon-precomposed" href="/pre"><link rel="apple-touch-icon" href="/apple"><link rel="shortcut icon" href="/shortcut">`, nil, "/shortcut"},
		{"apple_priority", `<link rel="apple-touch-icon-precomposed" href="/pre"><link rel="apple-touch-icon" href="/apple">`, nil, "/apple"},
		{"precomposed", `<link rel="apple-touch-icon-precomposed" href="/pre">`, nil, "/pre"},
		{"tokens_and_case", "<link rel='noticon' href='/wrong'><link rel='AlTeRnAtE\tICON' href='/token'>", nil, "/token"},
		{"shortcut_tokens", "<link rel='apple-touch-icon' href='/apple'><link rel='ICON\nSHORTCUT' href='/shortcut'>", nil, "/shortcut"},
		{"first_nonempty", `<link rel="icon"><link rel="icon" href=" "><link rel="icon" href=" /first "><link rel="icon" href="/second">`, nil, "/first"},
		{"same_shortcut_first", `<link rel="shortcut icon" href="/first"><link rel="shortcut icon" href="/second">`, nil, "/first"},
		{"same_apple_first", `<link rel="apple-touch-icon" href="/first"><link rel="apple-touch-icon" href="/second">`, nil, "/first"},
		{"same_pre_first", `<link rel="apple-touch-icon-precomposed" href="/first"><link rel="apple-touch-icon-precomposed" href="/second">`, nil, "/first"},
		{"href_entities_once", `<link rel="icon" href="/icon?a=&amp;amp;">`, nil, "/icon?a=&amp;"},
		{"ignore_active_content", `<base href="/wrong/"><meta http-equiv="refresh" content="0; url=/refresh"><script src="/script"></script><img src="/image"><link rel="stylesheet" href="/style"><link rel="icon" href="relative">`, nil, "/dir/relative"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var paths []string
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				paths = append(paths, r.Method+" "+r.URL.RequestURI())
				mu.Unlock()
				if r.URL.Path == "/dir/page" {
					response(w, 200, "Text/HTML; charset=utf-8", tc.html)
					return
				}
				response(w, 200, "image/png", "icon")
			}))
			defer s.Close()
			target := s.URL + "/dir/page"
			got := newCrawler(t).Crawl(t.Context(), target)
			want := success(target, tc.title)
			want["favicon"] = value("data:image/png;base64,aWNvbg==")
			want["favicon_url"] = value(s.URL + tc.icon)
			wantMetadata(t, got, want)
			mu.Lock()
			defer mu.Unlock()
			if !reflect.DeepEqual(paths, []string{"GET /dir/page", "GET " + tc.icon}) {
				t.Errorf("requests=%v", paths)
			}
		})
	}
	t.Run("final_page_and_icon_urls", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/start":
				w.Header().Set("Location", "/final/page")
				w.WriteHeader(302)
			case "/final/page":
				response(w, 200, "text/html", `<title>redirected</title><base href="/wrong/"><link rel="icon" href="pic">`)
			case "/final/pic":
				w.Header().Set("Location", "/assets/icon")
				w.WriteHeader(302)
			case "/assets/icon":
				response(w, 200, "image/png", "icon")
			default:
				t.Errorf("unexpected path: %s", r.URL.Path)
				w.WriteHeader(404)
			}
		}))
		defer s.Close()
		target := s.URL + "/start?original=1"
		got := newCrawler(t).Crawl(t.Context(), target)
		want := success(target, value("redirected"))
		want["favicon"] = value("data:image/png;base64,aWNvbg==")
		want["favicon_url"] = value(s.URL + "/assets/icon")
		wantMetadata(t, got, want)
	})
}
