package s3test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/containerd/errdefs"
)

func TestS3ServiceLifecycle(t *testing.T) {
	t.Run("independent writable buckets and owned cleanup", func(t *testing.T) {
		service := New(t)
		inspect, err := service.container.Inspect(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		for _, bindings := range inspect.HostConfig.PortBindings {
			for _, binding := range bindings {
				if binding.HostIP.String() != "127.0.0.1" {
					t.Fatalf("test port exposed beyond loopback: %v", binding)
				}
			}
		}
		first, second := service.NewBucket(t), service.NewBucket(t)
		if first.Name == second.Name {
			t.Fatal("test worlds share a bucket")
		}
		client := first.Client()
		_, err = client.PutObject(t.Context(), &s3.PutObjectInput{Bucket: aws.String(first.Name), Key: aws.String("proof"), Body: strings.NewReader("real bytes")})
		if err != nil {
			t.Fatal(err)
		}
		out, err := client.GetObject(t.Context(), &s3.GetObjectInput{Bucket: aws.String(first.Name), Key: aws.String("proof")})
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(out.Body)
		_ = out.Body.Close()
		if err != nil || string(body) != "real bytes" {
			t.Fatalf("object = %q: %v", body, err)
		}
		_, err = second.Client().HeadObject(t.Context(), &s3.HeadObjectInput{Bucket: aws.String(second.Name), Key: aws.String("proof")})
		if err == nil {
			t.Fatal("bucket data crossed worlds")
		}
		req, err := http.NewRequestWithContext(t.Context(), "GET", first.Endpoint+"/"+first.Name+"/proof", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != 403 {
			t.Fatalf("anonymous status = %d", resp.StatusCode)
		}
		if err := service.Close(); err != nil {
			t.Fatal(err)
		}
		if err := service.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := service.container.State(t.Context()); !errdefs.IsNotFound(err) {
			t.Fatalf("own container removal not confirmed: %v", err)
		}
	})
	t.Run("failed readiness also removes its own container", func(t *testing.T) {
		if testing.Short() {
			t.Skip("integration test: needs Docker")
		}
		ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
		defer cancel()
		var observed *Service
		want := errors.New("controlled readiness failure")
		_, err := startService(ctx, func(_ context.Context, s *Service) error { observed = s; return want })
		if !errors.Is(err, want) || observed == nil {
			t.Fatalf("failure path not reached: %v", err)
		}
		if _, err := observed.container.State(t.Context()); !errdefs.IsNotFound(err) {
			t.Fatalf("failed startup container removal not confirmed: %v", err)
		}
	})
}
