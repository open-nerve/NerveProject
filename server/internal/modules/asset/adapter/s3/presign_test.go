package s3

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func objectKey(t *testing.T, kind byte) domain.ObjectKey {
	t.Helper()
	k, err := domain.NewObjectKey(kind, uuid.MustParse("75ff43ce-6ea3-4299-818d-af6275d98207"), uuid.MustParse("f76161f8-0c22-4940-88f7-23c9f30ad06a"))
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func signatureExpiry(t *testing.T, raw string) time.Time {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	at, err := time.Parse("20060102T150405Z", u.Query().Get("X-Amz-Date"))
	if err != nil {
		t.Fatal(err)
	}
	seconds, err := strconv.ParseInt(u.Query().Get("X-Amz-Expires"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return at.Add(time.Duration(seconds) * time.Second)
}

// Duration truncation used at a different clock instant makes a persisted
// deadline one second early. These cases use literal signing instants and
// expected expiries while retaining the official V4 signing algorithm.
func TestS3PresignDeadline(t *testing.T) {
	base := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name         string
		at, deadline time.Time
		limit        time.Duration
		valid        bool
	}{
		{"fractional signing second", base.Add(900 * time.Millisecond), base.Add(2 * time.Second), 15 * time.Minute, true},
		{"crossed signing second", base.Add(1100 * time.Millisecond), base.Add(2 * time.Second), 15 * time.Minute, true},
		{"minimum whole second", base, base.Add(time.Second), 15 * time.Minute, true},
		{"upload upper bound", base, base.Add(15 * time.Minute), 15 * time.Minute, true},
		{"read upper bound", base, base.Add(65 * time.Minute), 65 * time.Minute, true},
		{"already expired", base, base, 15 * time.Minute, false},
		{"queued through deadline", base.Add(2 * time.Second), base.Add(2 * time.Second), 15 * time.Minute, false},
		{"upload too long", base, base.Add(15*time.Minute + time.Second), 15 * time.Minute, false},
		{"read too long", base, base.Add(65*time.Minute + time.Second), 65 * time.Minute, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), "PUT", "https://storage.invalid/bucket/owned/u-key?X-Amz-Expires=999", nil)
			if err != nil {
				t.Fatal(err)
			}
			p := deadlinePresigner{deadline: tc.deadline, limit: tc.limit, invalid: domain.ErrInvalidUpload}
			raw, _, err := p.PresignHTTP(t.Context(), aws.Credentials{AccessKeyID: "fixture", SecretAccessKey: "fixture-secret"}, req, "UNSIGNED-PAYLOAD", "s3", "us-east-1", tc.at)
			if !tc.valid {
				if !errors.Is(err, domain.ErrInvalidUpload) {
					t.Fatalf("invalid deadline accepted: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if actual := signatureExpiry(t, raw); !actual.Equal(tc.deadline) {
				t.Fatalf("signature deadline = %s, want %s", actual, tc.deadline)
			}
			u, _ := url.Parse(raw)
			if u.Query().Get("X-Amz-Signature") == "" {
				t.Fatal("missing official signature")
			}
		})
	}
	t.Run("cancellation before signing", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		req, err := http.NewRequestWithContext(ctx, "GET", "https://storage.invalid/bucket/key", nil)
		if err != nil {
			t.Fatal(err)
		}
		p := deadlinePresigner{deadline: base.Add(time.Minute), limit: 65 * time.Minute, invalid: domain.ErrInvalidRead}
		_, _, err = p.PresignHTTP(ctx, aws.Credentials{}, req, "UNSIGNED-PAYLOAD", "s3", "us-east-1", base)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled signature accepted: %v", err)
		}
	})
	t.Run("grant boundaries before rounding", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { versionReply(w, r, "") }))
		defer srv.Close()
		now := time.Now().UTC()
		s, err := New(t.Context(), testConfig(srv.URL), Deps{RuntimeNow: func() time.Time { return now }})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = s.Close() }()
		for _, ttl := range []time.Duration{15 * time.Minute, 15*time.Minute + time.Nanosecond, 0, time.Nanosecond} {
			g := domain.UploadGrant{Key: objectKey(t, 'u'), Size: 3, ContentType: "text/plain", ExpiresAt: now.Add(ttl)}
			signed, err := s.SignUpload(t.Context(), g)
			if ttl != 15*time.Minute {
				if !errors.Is(err, domain.ErrInvalidUpload) {
					t.Fatalf("upload ttl=%v err=%v", ttl, err)
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			if !signed.ExpiresAt.Equal(g.ExpiresAt.Truncate(time.Second)) || !signatureExpiry(t, signed.URL).Equal(signed.ExpiresAt) {
				t.Fatal("upload absolute expiry changed")
			}
		}
		for _, ttl := range []time.Duration{65 * time.Minute, 65*time.Minute + time.Nanosecond, 0, time.Nanosecond} {
			g := domain.ReadGrant{Key: objectKey(t, 'o'), Method: "HEAD", Name: "read.txt", ContentType: "text/plain", ExpiresAt: now.Add(ttl)}
			signed, err := s.SignRead(t.Context(), g)
			if ttl != 65*time.Minute {
				if !errors.Is(err, domain.ErrInvalidRead) {
					t.Fatalf("read ttl=%v err=%v", ttl, err)
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			if !signed.ExpiresAt.Equal(g.ExpiresAt.Truncate(time.Second)) || !signatureExpiry(t, signed.URL).Equal(signed.ExpiresAt) {
				t.Fatal("read absolute expiry changed")
			}
		}
	})
}
