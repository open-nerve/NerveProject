package httpadapter

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/storagetest"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestLocalHTTPReadContract(t *testing.T) {
	w := newTransferWorld(t)
	source := w.upload(t, "0123456789", "image/png")
	sealed, err := w.store.Seal(t.Context(), source.Key, transferKey(t, 'o'), source.ETag)
	if err != nil {
		t.Fatal(err)
	}
	grant := domain.ReadGrant{Key: sealed.Key, Method: "GET", Name: "项目 \"封面\".png", ContentType: "image/png", Inline: true, ExpiresAt: w.now().Add(time.Minute)}
	signed, err := w.store.SignRead(t.Context(), grant)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name               string
		headers            map[string]string
		status             int
		body, contentRange string
	}{
		{"full", nil, 200, "0123456789", ""},
		{"prefix", map[string]string{"Range": "bytes=0-2"}, 206, "012", "bytes 0-2/10"},
		{"suffix", map[string]string{"Range": "bytes=-3"}, 206, "789", "bytes 7-9/10"},
		{"open end", map[string]string{"Range": "bytes=7-"}, 206, "789", "bytes 7-9/10"},
		{"clipped end", map[string]string{"Range": "bytes=8-99"}, 206, "89", "bytes 8-9/10"},
		{"if range matches", map[string]string{"Range": "bytes=0-2", "If-Range": sealed.ETag}, 206, "012", "bytes 0-2/10"},
		{"if range mismatches", map[string]string{"Range": "bytes=0-2", "If-Range": `"other"`}, 200, "0123456789", ""},
		{"if range date matches", map[string]string{"Range": "bytes=0-2", "If-Range": sealed.ModifiedAt.Format(http.TimeFormat)}, 206, "012", "bytes 0-2/10"},
		{"not modified", map[string]string{"If-None-Match": sealed.ETag}, 304, "", ""},
		{"weak etag", map[string]string{"If-None-Match": "W/" + sealed.ETag}, 304, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, data := request(t, w, "GET", signed.URL, nil, tc.headers)
			if response.StatusCode != tc.status || string(data) != tc.body || response.Header.Get("Content-Range") != tc.contentRange {
				t.Fatalf("read = %d %q range %q", response.StatusCode, data, response.Header.Get("Content-Range"))
			}
			if response.Header.Get("ETag") != sealed.ETag || response.Header.Get("Cache-Control") != "private, max-age=60" || response.Header.Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("headers = %v", response.Header)
			}
			if tc.status != 304 {
				if response.Header.Get("Content-Length") != strconv.Itoa(len(tc.body)) || response.Header.Get("Content-Type") != "image/png" || response.Header.Get("Last-Modified") != sealed.ModifiedAt.Format(http.TimeFormat) {
					t.Fatalf("content headers = %v", response.Header)
				}
				disposition, params, err := mime.ParseMediaType(response.Header.Get("Content-Disposition"))
				if err != nil || disposition != "inline" || params["filename"] != grant.Name {
					t.Fatalf("disposition = %q, %v", response.Header.Get("Content-Disposition"), err)
				}
			}
		})
	}
	for _, r := range []string{"bytes=10-", "bytes=3-1", "bytes=0-1,3-4", "bytes=-0", "bytes=", "bytes=+1-2", "items=0-2", "bytes=99999999999999999999999999999-"} {
		t.Run(r, func(t *testing.T) {
			response, data := request(t, w, "GET", signed.URL, nil, map[string]string{"Range": r})
			if response.StatusCode != 416 || response.Header.Get("Cache-Control") != "no-store" || response.Header.Get("Content-Type") != "application/problem+json" || !bytes.Contains(data, []byte(`"code":"asset.invalid_range"`)) {
				t.Fatalf("range %q = %d %s %v", r, response.StatusCode, data, response.Header)
			}
		})
	}
	headGrant := grant
	headGrant.Method = "HEAD"
	head, err := w.store.SignRead(t.Context(), headGrant)
	if err != nil {
		t.Fatal(err)
	}
	response, data := request(t, w, "HEAD", head.URL, nil, nil)
	if response.StatusCode != 200 || len(data) != 0 || response.ContentLength != 10 || response.Header.Get("ETag") != sealed.ETag {
		t.Fatalf("HEAD = %d %q %v", response.StatusCode, data, response.Header)
	}
	response, _ = request(t, w, "HEAD", signed.URL, nil, nil)
	if response.StatusCode != 403 {
		t.Fatalf("GET signature used as HEAD = %d", response.StatusCode)
	}
	missingGrant := grant
	missingGrant.Key = transferKey(t, 'o')
	missing, err := w.store.SignRead(t.Context(), missingGrant)
	if err != nil {
		t.Fatal(err)
	}
	response, data = request(t, w, "GET", missing.URL, nil, nil)
	if response.StatusCode != 404 || !bytes.Contains(data, []byte(`"code":"asset.object_not_found"`)) {
		t.Fatalf("missing = %d %s", response.StatusCode, data)
	}
	for _, change := range []func(*domain.ReadGrant){
		func(g *domain.ReadGrant) { g.Key = source.Key },
		func(g *domain.ReadGrant) { g.Name = "evil\r\nInjected: yes" },
		func(g *domain.ReadGrant) { g.ContentType = "text/html" },
		func(g *domain.ReadGrant) { g.ContentType = "image/svg+xml" },
	} {
		g := grant
		change(&g)
		response, data = request(t, w, "GET", unsafeReadURL(t, g), nil, nil)
		if response.StatusCode != 403 || response.Header.Get("Injected") != "" || bytes.Contains(data, []byte(g.Name)) {
			t.Fatalf("invalid signed grant = %d %s", response.StatusCode, data)
		}
	}
	attachment := grant
	attachment.Name = `name "quoted".png`
	attachment.Inline = false
	download, err := w.store.SignRead(t.Context(), attachment)
	if err != nil {
		t.Fatal(err)
	}
	response, _ = request(t, w, "GET", download.URL, nil, nil)
	disposition, params, err := mime.ParseMediaType(response.Header.Get("Content-Disposition"))
	if err != nil || disposition != "attachment" || params["filename"] != attachment.Name {
		t.Fatalf("attachment header = %v, %v", response.Header, err)
	}
	t.Run("expired conditional", func(t *testing.T) {
		w.clock.Store(signed.ExpiresAt.Unix())
		response, data = request(t, w, "GET", signed.URL, nil, map[string]string{"If-None-Match": sealed.ETag})
		if response.StatusCode != 403 || response.Header.Get("Cache-Control") != "no-store" || response.Header.Get("ETag") != "" || strings.Contains(string(data), string(sealed.Key)) {
			t.Fatalf("expired conditional = %d %s %v", response.StatusCode, data, response.Header)
		}
	})
}

// Produce a cryptographically valid v1 envelope with an invalid policy value.
// This isolates key/type/name validation from MAC tamper protection.
func unsafeReadURL(t *testing.T, g domain.ReadGrant) string {
	t.Helper()
	p := struct {
		Version     int              `json:"v"`
		Purpose     string           `json:"purpose"`
		Method      string           `json:"method"`
		Key         domain.ObjectKey `json:"key"`
		Size        int64            `json:"size"`
		ContentType string           `json:"content_type"`
		Name        string           `json:"name"`
		Inline      bool             `json:"inline"`
		ExpiresAt   int64            `json:"expires_at"`
	}{1, "read", g.Method, g.Key, 0, g.ContentType, g.Name, g.Inline, g.ExpiresAt.Unix()}
	encoded, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	tag := storagetest.MAC("read-http-test").Tag(encoded)
	return "/static/local/" + string(g.Key) + "?payload=" + base64.RawURLEncoding.EncodeToString(encoded) + "&tag=" + base64.RawURLEncoding.EncodeToString(tag[:])
}
