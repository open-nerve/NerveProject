package app_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	assets3 "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/s3"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
)

func runS3StorageContract(t *testing.T) {
	b := s3test.New(t).NewBucket(t)
	s, err := assets3.New(t.Context(), assets3.Config{Endpoint: b.Endpoint, Region: b.Region, Bucket: b.Name, Prefix: "contract/", UsePathStyle: true, AccessKeyID: b.AccessKeyID, SecretAccessKey: b.SecretAccessKey, SizeLimit: 5 << 20, OperationTimeout: 10 * time.Second}, assets3.Deps{RuntimeNow: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	transport := &http.Transport{Proxy: nil}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	put := func(ctx context.Context, g domain.UploadGrant, body io.Reader) error {
		signed, err := s.SignUpload(ctx, g)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, signed.Method, signed.URL, body)
		if err != nil {
			return err
		}
		req.ContentLength = g.Size
		for k, v := range signed.Headers {
			req.Header.Set(k, v)
		}
		response, err := client.Do(req)
		if err != nil {
			return err
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode != 200 && response.StatusCode != 204 {
			return fmt.Errorf("S3 raw upload status %d", response.StatusCode)
		}
		return nil
	}
	now := time.Now()
	checkRead := func(info domain.ObjectInfo) {
		for _, method := range []string{"GET", "HEAD"} {
			signed, err := s.SignRead(t.Context(), domain.ReadGrant{Key: info.Key, Method: method, Name: "hello.txt", ContentType: info.ContentType, ExpiresAt: now.Add(time.Minute)})
			if err != nil {
				t.Fatal(err)
			}
			req, err := http.NewRequestWithContext(t.Context(), method, signed.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			want := "hello"
			if method == "HEAD" {
				want = ""
			}
			if err != nil || closeErr != nil || response.StatusCode != 200 || string(data) != want || response.ContentLength != 5 || response.Header.Get("ETag") != info.ETag {
				t.Fatalf("S3 HTTP %s = %d %q %v / %v", method, response.StatusCode, data, err, closeErr)
			}
		}
	}
	runStorageContract(t, s, put, checkRead, now)
}
