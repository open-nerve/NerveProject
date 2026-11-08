// Package s3test owns isolated S3 protocol services for tests only. Production
// imports are forbidden by archtest. No existing container is ever modified.
package s3test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const image = "chrislusf/seaweedfs@sha256:57877e32bb90073b37dc109b438ba66e84fd5ed9ced5f5c04f56444ef243ce59"

type Service struct {
	container                testcontainers.Container
	endpoint, access, secret string
	client                   *s3.Client
	transport                *http.Transport
	closeOnce                sync.Once
	closeErr                 error
}

func New(t testing.TB) *Service {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: needs Docker")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	s, err := startService(ctx, ready)
	if err != nil {
		t.Fatalf("s3test start: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("s3test cleanup: %v", err)
		}
	})
	return s
}

func startService(ctx context.Context, check func(context.Context, *Service) error) (*Service, error) {
	s := &Service{access: rand.Text(), secret: rand.Text() + rand.Text()}
	auth, err := json.Marshal(map[string]any{"identities": []any{map[string]any{"name": "nerve-test", "credentials": []any{map[string]string{"accessKey": s.access, "secretKey": s.secret}}, "actions": []string{"Admin", "Read", "Write", "List", "Tagging"}}}})
	if err != nil {
		return nil, err
	}
	// Each start builds its request anew: a start reads the file's reader.
	s.container, err = startPublished(func() (testcontainers.Container, error) {
		return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: testcontainers.ContainerRequest{
			// Docker copies this synthetic secret as root. The image drops to
			// seaweed, so transfer ownership before invoking its normal entrypoint.
			Entrypoint: []string{"/bin/sh", "-c", `chown seaweed:seaweed /etc/nerve-s3.json && exec /entrypoint.sh "$@"`, "nerve-s3-test"},
			Image:      image, Name: "nerve-s3-test-" + strings.ToLower(rand.Text()), ExposedPorts: []string{"8333/tcp"},
			Labels: map[string]string{"nerve.test-owner": "s3test"}, Tmpfs: map[string]string{"/data": "rw,size=2g"},
			Files: []testcontainers.ContainerFile{{Reader: strings.NewReader(string(auth)), ContainerFilePath: "/etc/nerve-s3.json", FileMode: 0600}},
			Cmd:   []string{"server", "-dir=/data", "-ip=127.0.0.1", "-ip.bind=0.0.0.0", "-s3", "-s3.config=/etc/nerve-s3.json", "-s3.iam=false", "-s3.port.iceberg=0", "-s3.port.lance=0", "-volume.max=32", "-master.volumeSizeLimitMB=512", "-master.telemetry=false"},
			HostConfigModifier: func(h *container.HostConfig) {
				h.PortBindings = network.PortMap{network.MustParsePort("8333/tcp"): {{HostIP: netip.MustParseAddr("127.0.0.1")}}}
			},
			WaitingFor: wait.ForHTTP("/").WithPort("8333/tcp").WithStatusCodeMatcher(func(code int) bool { return code == 400 || code == 403 }).WithStartupTimeout(90 * time.Second),
		}, Started: true})
	})
	if err != nil {
		return nil, errors.Join(err, s.Close())
	}
	port, err := s.container.MappedPort(ctx, "8333/tcp")
	if err != nil {
		return nil, errors.Join(err, s.Close())
	}
	s.endpoint = fmt.Sprintf("http://127.0.0.1:%d", port.Num())
	s.transport = &http.Transport{Proxy: nil, MaxIdleConns: 16, IdleConnTimeout: 30 * time.Second}
	s.client = s3.NewFromConfig(aws.Config{Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider(s.access, s.secret, ""),
		HTTPClient: &http.Client{Transport: s.transport, Timeout: 10 * time.Second}, RetryMaxAttempts: 1}, func(o *s3.Options) { o.BaseEndpoint = aws.String(s.endpoint); o.UsePathStyle = true })
	if err := check(ctx, s); err != nil {
		return nil, errors.Join(err, s.Close())
	}
	return s, nil
}

func (s *Service) Close() error {
	s.closeOnce.Do(func() {
		if s.transport != nil {
			s.transport.CloseIdleConnections()
		}
		if s.container != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			// The owned test data is discarded; allow two seconds for graceful
			// exit inside the thirty-second removal budget.
			s.closeErr = s.container.Terminate(ctx, testcontainers.StopTimeout(2*time.Second))
		}
	})
	return s.closeErr
}
