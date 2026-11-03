package s3

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func wireInventoryStore(t *testing.T, body string, calls *atomic.Int32) *Store {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("list-type") != "2" {
			versionReply(w, r, "")
			return
		}
		if calls != nil {
			calls.Add(1)
		}
		if r.URL.Query().Get("prefix") != "test/" || r.URL.Query().Get("max-keys") == "" || r.URL.Query().Has("delimiter") {
			t.Error("list request omitted exact prefix/page bound or grouped objects")
		}
		w.Header().Set("Content-Type", "application/xml")
		// A chunked response must be bounded too, not just Content-Length.
		w.(http.Flusher).Flush()
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	s, err := New(t.Context(), testConfig(srv.URL), Deps{RuntimeNow: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func inventoryEntry(key string) string {
	return "<Contents><Key>" + key + "</Key><LastModified>2026-11-03T00:00:00Z</LastModified><ETag>&quot;opaque&quot;</ETag><Size>3</Size></Contents>"
}
func inventoryDocument(contents, tail string) string {
	return "<ListBucketResult>" + contents + tail + "</ListBucketResult>"
}

func inventoryWire(t *testing.T) {
	a, b := "test/"+string(objectKey(t, 'o')), "test/"+string(objectKey(t, 'u'))
	one := inventoryEntry(a)
	for _, tc := range []struct {
		name, content, tail, token string
		limit                      int
		wantError                  bool
	}{
		{"empty exhausted", "", "<IsTruncated>false</IsTruncated>", "", 1, false},
		{"opaque token", one, "<IsTruncated>true</IsTruncated><NextContinuationToken>opaque+/=token</NextContinuationToken>", "input+/=token", 1, false},
		{"missing truncation state", one, "", "", 1, true},
		{"truncated without token", one, "<IsTruncated>true</IsTruncated>", "", 1, true},
		{"empty truncated page", "", "<IsTruncated>true</IsTruncated><NextContinuationToken>next</NextContinuationToken>", "", 1, true},
		{"repeated token", one, "<IsTruncated>true</IsTruncated><NextContinuationToken>same</NextContinuationToken>", "same", 1, true},
		{"token after exhaustion", one, "<IsTruncated>false</IsTruncated><NextContinuationToken>next</NextContinuationToken>", "", 1, true},
		{"large output token", one, "<IsTruncated>true</IsTruncated><NextContinuationToken>" + strings.Repeat("x", 8193) + "</NextContinuationToken>", "", 1, true},
		{"outside prefix", inventoryEntry("neighbor/" + string(objectKey(t, 'o'))), "<IsTruncated>false</IsTruncated>", "", 1, true},
		{"invalid layout", inventoryEntry(a + "/data"), "<IsTruncated>false</IsTruncated>", "", 1, true},
		{"missing key", strings.Replace(one, "<Key>"+a+"</Key>", "", 1), "<IsTruncated>false</IsTruncated>", "", 1, true},
		{"duplicate", one + one, "<IsTruncated>false</IsTruncated>", "", 2, true},
		{"descending", inventoryEntry(b) + one, "<IsTruncated>false</IsTruncated>", "", 2, true},
		{"over page", one + inventoryEntry(b), "<IsTruncated>false</IsTruncated>", "", 1, true},
		{"negative size", strings.Replace(one, "<Size>3</Size>", "<Size>-1</Size>", 1), "<IsTruncated>false</IsTruncated>", "", 1, true},
		{"missing size", strings.Replace(one, "<Size>3</Size>", "", 1), "<IsTruncated>false</IsTruncated>", "", 1, true},
		{"zero size", strings.Replace(one, "<Size>3</Size>", "<Size>0</Size>", 1), "<IsTruncated>false</IsTruncated>", "", 1, true},
		{"oversize", strings.Replace(one, "<Size>3</Size>", "<Size>104857601</Size>", 1), "<IsTruncated>false</IsTruncated>", "", 1, true},
		{"invalid etag", strings.Replace(one, "&quot;opaque&quot;", "*", 1), "<IsTruncated>false</IsTruncated>", "", 1, true},
		{"missing time", strings.Replace(one, "<LastModified>2026-11-03T00:00:00Z</LastModified>", "", 1), "<IsTruncated>false</IsTruncated>", "", 1, true},
		{"unexpected directories", one + "<CommonPrefixes><Prefix>test/directory/</Prefix></CommonPrefixes>", "<IsTruncated>false</IsTruncated>", "", 1, true},
		{"bounded XML before decoding", one + "<Unexpected>" + strings.Repeat("x", 96<<10) + "</Unexpected>", "<IsTruncated>false</IsTruncated>", "", 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := wireInventoryStore(t, inventoryDocument(tc.content, tc.tail), nil)
			page, err := s.List(t.Context(), tc.token, tc.limit)
			if (err != nil) != tc.wantError {
				t.Fatalf("page=%+v error=%v wantError=%t", page, err, tc.wantError)
			}
			if tc.name == "opaque token" && page.Next != "opaque+/=token" {
				t.Fatalf("token interpreted: %q", page.Next)
			}
		})
	}
	for _, tc := range []struct {
		limit int
		token string
	}{{0, ""}, {1001, ""}, {1, strings.Repeat("x", 8193)}} {
		t.Run(fmt.Sprintf("invalid-input-%d-%d", tc.limit, len(tc.token)), func(t *testing.T) {
			var calls atomic.Int32
			s := wireInventoryStore(t, inventoryDocument("", "<IsTruncated>false</IsTruncated>"), &calls)
			if _, err := s.List(t.Context(), tc.token, tc.limit); err == nil || calls.Load() != 0 {
				t.Fatalf("bad inventory input reached provider: %v calls%d", err, calls.Load())
			}
		})
	}
}
