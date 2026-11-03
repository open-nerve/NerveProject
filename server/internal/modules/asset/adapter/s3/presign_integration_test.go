package s3

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
)

func realStore(t *testing.T, bucket *s3test.Bucket) *Store {
	t.Helper()
	cfg := testConfig(bucket.Endpoint)
	cfg.Region, cfg.Bucket, cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken = bucket.Region, bucket.Name, bucket.AccessKeyID, bucket.SecretAccessKey, ""
	cfg.Prefix = "owned/instance/"
	cfg.OperationTimeout = 10 * time.Second
	s, err := New(t.Context(), cfg, Deps{RuntimeNow: time.Now})
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

func transfer(t *testing.T, signed domain.SignedRequest, body string) (*http.Response, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var reader io.Reader
	if signed.Method == "PUT" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, signed.Method, signed.URL, reader)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range signed.Headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	closeErr := resp.Body.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("transfer body: %v / %v", err, closeErr)
	}
	return resp, data
}

func TestS3SignedRequests(t *testing.T) {
	service := s3test.New(t)
	bucket := service.NewBucket(t)
	s := realStore(t, bucket)
	t.Run("canonical endpoint authenticates browser authority", func(t *testing.T) { canonicalEndpointTransfer(t, bucket) })
	upload, err := s.SignUpload(t.Context(), domain.UploadGrant{Key: objectKey(t, 'u'), Size: 3, ContentType: "text/plain", ExpiresAt: time.Now().Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(upload.URL)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme+"://"+u.Host != s.Origin() || u.Path != "/"+bucket.Name+"/owned/instance/"+string(objectKey(t, 'u')) {
		t.Fatal("signed URL escaped expected origin or prefix")
	}
	if upload.Method != "PUT" || !reflect.DeepEqual(upload.Headers, map[string]string{"Content-Type": "text/plain"}) {
		t.Fatalf("browser headers = %v", upload.Headers)
	}
	if u.Query().Get("X-Amz-SignedHeaders") != "content-length;content-type;host" {
		t.Errorf("size/type not signed: %s", u.Query().Get("X-Amz-SignedHeaders"))
	}
	if !signatureExpiry(t, upload.URL).Equal(upload.ExpiresAt) {
		t.Fatal("actual upload expiry differs from DTO")
	}
	t.Run("length and type are authenticated", func(t *testing.T) {
		for _, tc := range []struct {
			name, body, media string
			want              int
		}{{"wrong size", "abcd", "text/plain", 403}, {"wrong type", "abc", "image/png", 403}, {"correct", "abc", "text/plain", 200}} {
			t.Run(tc.name, func(t *testing.T) {
				copy := upload
				copy.Headers = map[string]string{"Content-Type": tc.media}
				resp, _ := transfer(t, copy, tc.body)
				if resp.StatusCode != tc.want {
					t.Fatalf("%s status=%d, want%d", tc.name, resp.StatusCode, tc.want)
				}
			})
		}
		head, err := bucket.Client().HeadObject(t.Context(), &awss3.HeadObjectInput{Bucket: aws.String(bucket.Name), Key: aws.String(s.cfg.Prefix + string(objectKey(t, 'u')))})
		if err != nil {
			t.Fatal(err)
		}
		if aws.ToInt64(head.ContentLength) != 3 || aws.ToString(head.ContentType) != "text/plain" {
			t.Fatalf("wrong actual upload metadata: %+v", head)
		}
	})
	t.Run("PUT method key and expiry tampering", func(t *testing.T) {
		for _, field := range []string{"method", "key", "expiry"} {
			t.Run(field, func(t *testing.T) {
				copy := upload
				u, _ := url.Parse(copy.URL)
				switch field {
				case "method":
					copy.Method = "GET"
				case "key":
					u.Path += "-other"
				case "expiry":
					q := u.Query()
					q.Set("X-Amz-Expires", "900")
					u.RawQuery = q.Encode()
				}
				copy.URL = u.String()
				resp, _ := transfer(t, copy, "abc")
				if resp.StatusCode != 403 {
					t.Fatalf("tampered PUT %s accepted: %d", field, resp.StatusCode)
				}
			})
		}
	})
	// A fixture object isolates read signing from T4's conditional sealing.
	_, err = bucket.Client().PutObject(t.Context(), &awss3.PutObjectInput{Bucket: aws.String(bucket.Name), Key: aws.String(s.cfg.Prefix + string(objectKey(t, 'o'))), Body: strings.NewReader("abc"), ContentType: aws.String("application/octet-stream")})
	if err != nil {
		t.Fatal(err)
	}
	t.Run("GET HEAD and disposition overrides", func(t *testing.T) {
		for _, method := range []string{"GET", "HEAD"} {
			for _, inline := range []bool{false, true} {
				grant := domain.ReadGrant{Key: objectKey(t, 'o'), Method: method, Name: "报告 \"final\".png", ContentType: "image/png", Inline: inline, ExpiresAt: time.Now().Add(time.Minute)}
				signed, err := s.SignRead(t.Context(), grant)
				if err != nil {
					t.Fatal(err)
				}
				if signed.Method != method || len(signed.Headers) != 0 || !signatureExpiry(t, signed.URL).Equal(signed.ExpiresAt) {
					t.Fatal("read method, public headers or expiry changed")
				}
				resp, data := transfer(t, signed, "")
				if resp.StatusCode != 200 || resp.ContentLength != 3 || resp.Header.Get("Content-Type") != "image/png" || resp.Header.Get("Cache-Control") != "private, no-store" {
					t.Fatalf("read response %d %v", resp.StatusCode, resp.Header)
				}
				kind, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
				want := "attachment"
				if inline {
					want = "inline"
				}
				if err != nil || kind != want || params["filename"] != grant.Name {
					t.Fatalf("disposition %q: %v", resp.Header.Get("Content-Disposition"), err)
				}
				if method == "GET" && sha256.Sum256(data) != sha256.Sum256([]byte("abc")) {
					t.Fatal("actual read bytes changed")
				}
				if method == "HEAD" && len(data) != 0 {
					t.Fatal("HEAD returned bytes")
				}
			}
		}
	})
	t.Run("read tampering and anonymous access", func(t *testing.T) {
		grant := domain.ReadGrant{Key: objectKey(t, 'o'), Method: "GET", Name: "report.txt", ContentType: "text/plain", ExpiresAt: time.Now().Add(time.Minute)}
		signed, err := s.SignRead(t.Context(), grant)
		if err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"method", "key", "query", "expiry", "anonymous"} {
			t.Run(field, func(t *testing.T) {
				copy := signed
				u, _ := url.Parse(copy.URL)
				q := u.Query()
				switch field {
				case "method":
					copy.Method = "HEAD"
				case "key":
					u.Path += "-other"
				case "query":
					q.Set("response-content-type", "image/png")
				case "expiry":
					q.Set("X-Amz-Expires", "3900")
				case "anonymous":
					q = url.Values{}
				}
				u.RawQuery = q.Encode()
				copy.URL = u.String()
				resp, _ := transfer(t, copy, "")
				if resp.StatusCode != 403 {
					t.Fatalf("tampered read %s accepted: %d", field, resp.StatusCode)
				}
			})
		}
		grant.Name = "bad\r\nX-Injected: yes"
		if _, err := s.SignRead(t.Context(), grant); !errors.Is(err, domain.ErrInvalidRead) {
			t.Fatalf("header injection accepted: %v", err)
		}
	})
	t.Run("late GET stays uncacheable and expired URL is refused", func(t *testing.T) {
		signed, err := s.SignRead(t.Context(), domain.ReadGrant{Key: objectKey(t, 'o'), Method: "GET", Name: "late.txt", ContentType: "text/plain", ExpiresAt: time.Now().Add(5 * time.Second)})
		if err != nil {
			t.Fatal(err)
		}
		waitUntil(t, signed.ExpiresAt.Add(-2*time.Second))
		resp, data := transfer(t, signed, "")
		if resp.StatusCode != 200 || string(data) != "abc" || resp.Header.Get("Cache-Control") != "private, no-store" {
			t.Fatalf("late GET cache or bytes: %d %v", resp.StatusCode, resp.Header)
		}
		waitUntil(t, signed.ExpiresAt.Add(time.Second))
		resp, _ = transfer(t, signed, "")
		if resp.StatusCode != 403 {
			t.Fatalf("expired GET accepted: %d", resp.StatusCode)
		}
	})
}

func waitUntil(t *testing.T, at time.Time) {
	t.Helper()
	timer := time.NewTimer(time.Until(at))
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
}
