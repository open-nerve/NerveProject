package local

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

// These keys are test inputs only. Bootstrap separately exercises real identity HKDF.
type testMAC string

func (m testMAC) Tag(message []byte) [16]byte {
	h := hmac.New(sha256.New, []byte(m))
	_, _ = h.Write(message)
	return [16]byte(h.Sum(nil)[:16])
}

func (m testMAC) Verify(message []byte, tag [16]byte) bool {
	want := m.Tag(message)
	return hmac.Equal(want[:], tag[:])
}

func signingStore(t *testing.T, now *time.Time) *Store {
	t.Helper()
	s, err := New(Config{Root: t.TempDir(), SizeLimit: 5242880}, Deps{UploadMAC: testMAC("upload-test-key"), ReadMAC: testMAC("read-test-key"), RuntimeNow: func() time.Time { return *now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s
}

func signingKey(t *testing.T, kind byte) domain.ObjectKey {
	t.Helper()
	key, err := domain.NewObjectKey(kind, uuid.MustParse("28fe241a-34a8-4d94-a769-116a4cd80d15"), uuid.MustParse("a34c291b-e277-43e2-acce-78a2a7ff053e"))
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func signedURL(t *testing.T, request domain.SignedRequest) *url.URL {
	t.Helper()
	u, err := url.Parse(request.URL)
	if err != nil || u.IsAbs() || u.Host != "" || u.Fragment != "" || !strings.HasPrefix(u.Path, "/storage/local/") {
		t.Fatalf("not a same-origin transfer URL: %+v, %v", request, err)
	}
	return u
}

func TestSignedTransferBindsEveryField(t *testing.T) {
	now := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	s := signingStore(t, &now)
	upload := domain.UploadGrant{Key: signingKey(t, 'u'), Size: 123, ContentType: "image/png", ExpiresAt: now.Add(10*time.Minute + 123*time.Millisecond)}
	read := domain.ReadGrant{Key: signingKey(t, 'o'), Method: "GET", Name: "项目 \"封面\".png", ContentType: "image/png", Inline: true, ExpiresAt: now.Add(60 * time.Minute)}
	put, err := s.SignUpload(t.Context(), upload)
	if err != nil {
		t.Fatal(err)
	}
	putURL := signedURL(t, put)
	if put.Method != "PUT" || putURL.Path != "/storage/local/"+string(upload.Key) || !reflect.DeepEqual(put.Headers, map[string]string{"Content-Type": "image/png"}) || !put.ExpiresAt.Equal(upload.ExpiresAt.Truncate(time.Second)) {
		t.Fatalf("upload request = %+v", put)
	}
	gotUpload, err := s.VerifyUpload(string(upload.Key), putURL.RawQuery, "PUT")
	upload.ExpiresAt = upload.ExpiresAt.Truncate(time.Second)
	if err != nil || gotUpload != upload {
		t.Fatalf("verify upload = %+v, %v", gotUpload, err)
	}
	get, err := s.SignRead(t.Context(), read)
	if err != nil {
		t.Fatal(err)
	}
	getURL := signedURL(t, get)
	if get.Method != "GET" || len(get.Headers) != 0 || !get.ExpiresAt.Equal(read.ExpiresAt) {
		t.Fatalf("read request = %+v", get)
	}
	gotRead, err := s.VerifyRead(string(read.Key), getURL.RawQuery, "GET")
	if err != nil || gotRead != read {
		t.Fatalf("verify read = %+v, %v", gotRead, err)
	}
	for _, tc := range []struct {
		name    string
		request domain.SignedRequest
		changes map[string]any
		upload  bool
	}{
		{"upload", put, map[string]any{"v": 2, "purpose": "read", "method": "GET", "key": strings.Replace(string(upload.Key), "053e", "053f", 1), "size": 124, "content_type": "image/gif", "expires_at": upload.ExpiresAt.Unix() + 1}, true},
		{"read", get, map[string]any{"v": 2, "purpose": "upload", "method": "HEAD", "key": strings.Replace(string(read.Key), "053e", "053f", 1), "name": "other.png", "inline": false, "content_type": "image/gif", "expires_at": read.ExpiresAt.Unix() + 1}, false},
	} {
		for field, value := range tc.changes {
			t.Run(tc.name+"/tampered "+field, func(t *testing.T) {
				u := signedURL(t, tc.request)
				q := u.Query()
				data, err := base64.RawURLEncoding.DecodeString(q.Get("payload"))
				if err != nil {
					t.Fatal(err)
				}
				var payload map[string]json.RawMessage
				if err := json.Unmarshal(data, &payload); err != nil {
					t.Fatal(err)
				}
				if _, exists := payload[field]; !exists {
					t.Fatalf("signed payload lacks %s", field)
				}
				encoded, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				// Preserve field order so canonical-JSON checks cannot hide a missing MAC.
				old := `"` + field + `":` + string(payload[field])
				if strings.Count(string(data), old) != 1 {
					t.Fatalf("cannot isolate field %s", field)
				}
				data = []byte(strings.Replace(string(data), old, `"`+field+`":`+string(encoded), 1))
				q.Set("payload", base64.RawURLEncoding.EncodeToString(data))
				key, method := string(read.Key), "GET"
				if tc.upload {
					key, method = string(upload.Key), "PUT"
				}
				if field == "key" {
					key = value.(string)
				}
				if field == "method" {
					method = value.(string)
				}
				if tc.upload {
					_, err = s.VerifyUpload(key, q.Encode(), method)
				} else {
					_, err = s.VerifyRead(key, q.Encode(), method)
				}
				if !errors.Is(err, domain.ErrInvalidSignature) {
					t.Fatalf("tamper = %v", err)
				}
			})
		}
	}
	if _, err := s.VerifyRead(string(read.Key), getURL.RawQuery, "HEAD"); !errors.Is(err, domain.ErrInvalidSignature) {
		t.Fatalf("GET token as HEAD = %v", err)
	}
	if _, err := s.VerifyUpload(string(read.Key), putURL.RawQuery, "PUT"); !errors.Is(err, domain.ErrInvalidSignature) {
		t.Fatalf("different path = %v", err)
	}
	headGrant := read
	headGrant.Method = "HEAD"
	head, err := s.SignRead(t.Context(), headGrant)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.VerifyRead(string(read.Key), signedURL(t, head).RawQuery, "HEAD"); err != nil {
		t.Fatal(err)
	}
	now = upload.ExpiresAt
	if _, err := s.VerifyUpload(string(upload.Key), putURL.RawQuery, "PUT"); !errors.Is(err, domain.ErrInvalidSignature) {
		t.Fatalf("expired exactly = %v", err)
	}
	now = read.ExpiresAt
	if _, err := s.VerifyRead(string(read.Key), getURL.RawQuery, "GET"); !errors.Is(err, domain.ErrInvalidSignature) {
		t.Fatalf("expired read exactly = %v", err)
	}
}

func TestSignedTransferRejectsAmbiguousQuery(t *testing.T) {
	now := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	s := signingStore(t, &now)
	grant := domain.UploadGrant{Key: signingKey(t, 'u'), Size: 3, ContentType: "image/png", ExpiresAt: now.Add(time.Minute)}
	request, err := s.SignUpload(t.Context(), grant)
	if err != nil {
		t.Fatal(err)
	}
	u := signedURL(t, request)
	q := u.Query()
	for _, query := range []string{
		"", u.RawQuery + "&payload=" + q.Get("payload"), u.RawQuery + "&tag=" + q.Get("tag"), u.RawQuery + "&x=y",
		"payload=" + q.Get("payload") + "=&tag=" + q.Get("tag"), "payload=%" + "65" + q.Get("payload")[1:] + "&tag=" + q.Get("tag"),
		"payload=" + q.Get("payload") + "&tag=", "payload=" + q.Get("payload") + "&tag=" + q.Get("tag") + "=", strings.Repeat("x", 4097),
		u.RawQuery + "&", u.RawQuery + ";bad", "payload=" + q.Get("payload") + "&Tag=" + q.Get("tag"),
	} {
		if _, err := s.VerifyUpload(string(grant.Key), query, "PUT"); !errors.Is(err, domain.ErrInvalidSignature) || err.Error() != domain.ErrInvalidSignature.Error() {
			t.Fatalf("ambiguous query = %v", err)
		}
	}
	// A valid MAC is still insufficient for an unsupported purpose/version or a
	// noncanonical payload. Use the real test MAC, so rejection cannot hide behind it.
	data, err := base64.RawURLEncoding.DecodeString(q.Get("payload"))
	if err != nil {
		t.Fatal(err)
	}
	for _, changed := range []string{
		strings.Replace(string(data), `"purpose":"upload"`, `"purpose":"read"`, 1),
		strings.Replace(string(data), `"v":1`, `"v":2`, 1),
		strings.Replace(string(data), `"name":""`, `"name":"unused.png"`, 1),
		strings.Replace(string(data), `"inline":false`, `"inline":true`, 1),
		" " + string(data), strings.TrimSuffix(string(data), "}") + `,"v":1}`, strings.TrimSuffix(string(data), "}") + `,"unknown":true}`,
	} {
		if changed == string(data) {
			t.Fatal("payload mutation did not change the input")
		}
		tag := testMAC("upload-test-key").Tag([]byte(changed))
		query := "payload=" + base64.RawURLEncoding.EncodeToString([]byte(changed)) + "&tag=" + base64.RawURLEncoding.EncodeToString(tag[:])
		if _, err := s.VerifyUpload(string(grant.Key), query, "PUT"); !errors.Is(err, domain.ErrInvalidSignature) {
			t.Fatalf("signed malformed payload %s = %v", changed, err)
		}
	}
	wrongTag := testMAC("read-test-key").Tag(data)
	q.Set("tag", base64.RawURLEncoding.EncodeToString(wrongTag[:]))
	if _, err := s.VerifyUpload(string(grant.Key), q.Encode(), "PUT"); !errors.Is(err, domain.ErrInvalidSignature) {
		t.Fatalf("wrong MAC purpose = %v", err)
	}
	// Keep the payload canonical and correctly tagged: only the transport's
	// size cap may reject it, not malformed syntax or an invalid signature.
	largeType := "application/" + strings.Repeat("a", 3100)
	largeData := []byte(strings.Replace(string(data), `"content_type":"image/png"`, `"content_type":"`+largeType+`"`, 1))
	largeTag := testMAC("upload-test-key").Tag(largeData)
	largeQuery := "payload=" + base64.RawURLEncoding.EncodeToString(largeData) + "&tag=" + base64.RawURLEncoding.EncodeToString(largeTag[:])
	if len(largeQuery) <= 4096 {
		t.Fatal("oversized-query fixture did not exceed its limit")
	}
	if _, err := s.VerifyUpload(string(grant.Key), largeQuery, "PUT"); !errors.Is(err, domain.ErrInvalidSignature) {
		t.Fatalf("oversized canonical query = %v", err)
	}
	grant.ContentType = largeType
	if _, err := s.SignUpload(t.Context(), grant); !errors.Is(err, domain.ErrInvalidUpload) {
		t.Fatalf("signer produced oversized query: %v", err)
	}
}

func TestSignedTransferRejectsInvalidInputs(t *testing.T) {
	now := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	s := signingStore(t, &now)
	grant := domain.UploadGrant{Key: signingKey(t, 'u'), Size: 3, ContentType: "image/png", ExpiresAt: now.Add(time.Minute)}
	for _, change := range []func(*domain.UploadGrant){
		func(g *domain.UploadGrant) { g.Key = signingKey(t, 'o') }, func(g *domain.UploadGrant) { g.Size = 0 },
		func(g *domain.UploadGrant) { g.ExpiresAt = now }, func(g *domain.UploadGrant) { g.ExpiresAt = now.Add(16 * time.Minute) },
		func(g *domain.UploadGrant) { g.ExpiresAt = now.Add(15*time.Minute + time.Nanosecond) },
	} {
		g := grant
		change(&g)
		if _, err := s.SignUpload(t.Context(), g); !errors.Is(err, domain.ErrInvalidUpload) {
			t.Errorf("invalid signing input = %v", err)
		}
	}
	read := domain.ReadGrant{Key: signingKey(t, 'o'), Method: "GET", Name: "test.png", ContentType: "image/png", Inline: true, ExpiresAt: now.Add(time.Minute)}
	for _, change := range []func(*domain.ReadGrant){
		func(g *domain.ReadGrant) { g.ExpiresAt = now.Add(65*time.Minute + time.Nanosecond) },
		func(g *domain.ReadGrant) { g.Key = signingKey(t, 'u') },
		func(g *domain.ReadGrant) { g.Method = "PUT" },
	} {
		g := read
		change(&g)
		if _, err := s.SignRead(t.Context(), g); !errors.Is(err, domain.ErrInvalidRead) {
			t.Errorf("invalid read signing input = %v", err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := s.SignUpload(ctx, grant); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled signer = %v", err)
	}
	if _, err := s.SignRead(ctx, read); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled read signer = %v", err)
	}
	var typedNil *testMAC
	for _, change := range []func(*Deps){
		func(d *Deps) { d.UploadMAC = nil }, func(d *Deps) { d.ReadMAC = nil },
		func(d *Deps) { d.UploadMAC = typedNil }, func(d *Deps) { d.ReadMAC = typedNil }, func(d *Deps) { d.RuntimeNow = nil },
	} {
		deps := Deps{UploadMAC: testMAC("upload"), ReadMAC: testMAC("read"), RuntimeNow: func() time.Time { return now }}
		change(&deps)
		if _, err := New(Config{Root: t.TempDir(), SizeLimit: 5242880}, deps); err == nil {
			t.Fatal("nil dependency admitted")
		}
	}
}
