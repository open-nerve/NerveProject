package s3

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
)

func awaitHeld(t *testing.T, h *s3test.HeldRequest) {
	t.Helper()
	select {
	case <-h.Reached:
	case err := <-h.Result:
		t.Fatalf("request did not reach hold: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("request hold timeout")
	}
}
func awaitStored(t *testing.T, h *s3test.HeldRequest) {
	t.Helper()
	select {
	case err := <-h.Result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("provider result timeout")
	}
}

// The caller has already observed cancellation while the independent provider
// request is still allowed to finish. No signed URL is modified after signing.
func TestS3LateObjects(t *testing.T) {
	service := s3test.New(t)
	t.Run("PUT begins before expiry but lands after cancellation and expiry", func(t *testing.T) {
		b := service.NewBucket(t)
		proxy := s3test.NewFaultProxy(t, b)
		through := *b
		through.Endpoint = proxy.URL
		s := realStore(t, &through)
		key := freshKey(t, 'u')
		signed, err := s.SignUpload(t.Context(), domain.UploadGrant{Key: key, Size: 3, ContentType: "text/plain", ExpiresAt: time.Now().Add(3 * time.Second)})
		if err != nil {
			t.Fatal(err)
		}
		held := proxy.HoldNext(t, "PUT", "/"+b.Name+"/"+s.cfg.Prefix+string(key), s3test.BeforeLastUploadByte)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		result := make(chan error, 1)
		go func() {
			req, err := http.NewRequestWithContext(ctx, "PUT", signed.URL, strings.NewReader("abc"))
			if err == nil {
				for k, v := range signed.Headers {
					req.Header.Set(k, v)
				}
				var resp *http.Response
				resp, err = http.DefaultClient.Do(req)
				if resp != nil {
					_ = resp.Body.Close()
				}
			}
			result <- err
		}()
		awaitHeld(t, held)
		cancel()
		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatalf("client not canceled: %v", err)
		}
		waitUntil(t, signed.ExpiresAt.Add(time.Second))
		held.Release()
		awaitStored(t, held)
		info, err := s.Inspect(t.Context(), key)
		if err != nil || info.Size != 3 {
			t.Fatalf("late PUT did not materialize: %+v %v", info, err)
		}
		// Inventory must discover the exact abandoned key.
		page, err := s.List(t.Context(), "", 1)
		if err != nil || len(page.Objects) != 1 || page.Objects[0].Key != key {
			t.Fatalf("late PUT invisible: %+v %v", page, err)
		}
	})
	t.Run("COPY finishes after client cancellation", func(t *testing.T) {
		b := service.NewBucket(t)
		proxy := s3test.NewFaultProxy(t, b)
		through := *b
		through.Endpoint = proxy.URL
		s := realStore(t, &through)
		source, target := freshKey(t, 'u'), freshKey(t, 'o')
		_, err := b.Client().PutObject(t.Context(), &awss3.PutObjectInput{Bucket: &b.Name, Key: aws.String(s.cfg.Prefix + string(source)), Body: strings.NewReader("abc"), ContentType: aws.String("text/plain")})
		if err != nil {
			t.Fatal(err)
		}
		info, err := s.Inspect(t.Context(), source)
		if err != nil {
			t.Fatal(err)
		}
		held := proxy.HoldNext(t, "PUT", "/"+b.Name+"/"+s.cfg.Prefix+string(target), s3test.BeforeRequest)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		result := make(chan error, 1)
		go func() { _, err := s.Seal(ctx, source, target, info.ETag); result <- err }()
		awaitHeld(t, held)
		cancel()
		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatalf("copy not canceled: %v", err)
		}
		if err := s.Delete(t.Context(), target); err != nil {
			t.Fatal(err)
		}
		held.Release()
		awaitStored(t, held)
		info, err = s.Inspect(t.Context(), target)
		if err != nil {
			t.Fatal("late COPY absent", err)
		}
		reader, err := s.Open(t.Context(), target, info.ETag)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil || string(body) != "abc" {
			t.Fatalf("late COPY bytes = %q %v", body, err)
		}
	})
}
