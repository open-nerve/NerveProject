package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// This is deliberately a real-duration test: a scaled-down timeout cannot prove
// that the shipped 30s socket deadline is replaced for this request alone.
func TestRawStorageDefaultSlowUpload(t *testing.T) {
	cfg := defaultAssetConfig(t, unreachableDB)
	cfg.Auth.JWT.PrivateKeyFile = writeFile(t, testKeyPEM)
	budget, err := cfg.Files.TransferBudget(5 << 20)
	if err != nil || budget != 55*time.Second || cfg.Server.ReadTimeout != 30*time.Second || cfg.Server.RequestTimeout != 15*time.Second || cfg.Server.MaxBodyBytes != 1<<20 {
		t.Fatalf("not the shipped budget: files=%s error=%v server=%+v", budget, err, cfg.Server)
	}
	base := startApp(t, cfg, migrations.FS())
	store := assetClientStore(t, cfg)
	pattern := []byte("m5-local-bytes!\n")
	data := bytes.Repeat(pattern, (5<<20)/len(pattern)+1)[:5<<20]
	key := assetKey(t, 'u')
	put, err := store.SignUpload(t.Context(), domain.UploadGrant{Key: key, Size: int64(len(data)), ContentType: "application/octet-stream", ExpiresAt: time.Now().Add(5 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 70*time.Second)
	defer cancel()
	stream := &pacedAssetBody{ctx: ctx, data: data, started: make(chan time.Time, 1)}
	req, err := http.NewRequestWithContext(ctx, put.Method, base+put.URL, stream)
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len(data))
	for key, value := range put.Headers {
		req.Header.Set(key, value)
	}
	transferClient := &http.Client{Timeout: 70 * time.Second}
	res, err := transferClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	response, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil || res.StatusCode != http.StatusNoContent {
		t.Fatalf("default slow upload status=%d body=%q err=%v", res.StatusCode, response, err)
	}
	var started time.Time
	select {
	case started = <-stream.started:
	case <-ctx.Done():
		t.Fatal("HTTP never read the upload stream")
	}
	elapsed := time.Since(started)
	if elapsed <= 30*time.Second || elapsed >= budget {
		t.Fatalf("actual body duration %s must exceed 30s and stay below %s", elapsed, budget)
	}
	info, err := store.Inspect(t.Context(), key)
	wantETag := fmt.Sprintf("\"%x\"", sha256.Sum256(data))
	if err != nil || info.Size != int64(len(data)) || info.ETag != wantETag {
		t.Fatalf("stored slow upload info=%+v err=%v", info, err)
	}
	sealed, err := store.Seal(t.Context(), key, assetKey(t, 'o'), info.ETag)
	if err != nil {
		t.Fatal(err)
	}
	get, err := store.SignRead(t.Context(), domain.ReadGrant{Key: sealed.Key, Method: "GET", Name: "slow.bin", ContentType: "application/octet-stream", ExpiresAt: time.Now().Add(5 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if got := assetRequest(t, base, get, "", http.StatusOK); !bytes.Equal([]byte(got), data) {
		t.Fatal("sealed download differs from the streamed bytes")
	}
	t.Logf("default upload: bytes=%d body_duration=%s file_budget=%s global_read=%s API=%s ETag=%s", len(data), elapsed, budget, cfg.Server.ReadTimeout, cfg.Server.RequestTimeout, wantETag)
}

// Pace by bytes against monotonic time: 5 MiB at 160 KiB/s takes about 32s.
// Context cancellation interrupts waits; no detached stream goroutine is needed.
type pacedAssetBody struct {
	ctx     context.Context
	data    []byte
	offset  int
	start   time.Time
	started chan time.Time
}

func (b *pacedAssetBody) Read(p []byte) (int, error) {
	if b.offset == len(b.data) {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	if b.start.IsZero() {
		b.start = time.Now()
		b.started <- b.start
	}
	if err := b.ctx.Err(); err != nil {
		return 0, err
	}
	if wait := time.Until(b.start.Add(time.Duration(b.offset) * time.Second / (160 << 10))); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-b.ctx.Done():
			return 0, b.ctx.Err()
		case <-timer.C:
		}
	}
	n := copy(p[:min(len(p), 16<<10)], b.data[b.offset:])
	b.offset += n
	return n, nil
}
