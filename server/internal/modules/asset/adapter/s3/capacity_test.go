package s3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
)

// Digests were independently computed with Python hashlib over repeated ASCII
// 'a'. Neither upload nor verification allocates a whole object in this process.
func TestS3Capacity1B5MiB100MiB(t *testing.T) {
	service := s3test.New(t)
	bucket := service.NewBucket(t)
	cfg := testConfig(bucket.Endpoint)
	cfg.Region, cfg.Bucket, cfg.Prefix = bucket.Region, bucket.Name, "capacity/"
	cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken = bucket.AccessKeyID, bucket.SecretAccessKey, ""
	cfg.SizeLimit, cfg.OperationTimeout = 100<<20, 10*time.Second
	store, err := New(t.Context(), cfg, Deps{RuntimeNow: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	transport := newTransport()
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport}
	budgets := config.FilesConfig{SizeLimit: 100 << 20, MinTransferBytesPerSecond: 64 << 10, TransferGrace: 10 * time.Second, MaxTransferDuration: 30 * time.Minute}
	t.Logf("environment GOOS=%s GOARCH=%s Go=%s logicalCPUs=%d provider=SeaweedFS4.52 loopback tmpfs2GiB; memory is this Go process, not provider RSS", runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.NumCPU())
	for _, tc := range []struct {
		size  int64
		sha   string
		pause time.Duration
	}{
		{1, "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb", 0},
		{5 << 20, "a29968fad2e782aa9f2040a35f05adb97ed8979eb1f572c8c8ea78637e275f3c", 16 * time.Second},
		{100 << 20, "cee41e98d0a6ad65cc0ec77a2ba50bf26d64dc9007f7f1c7d7df68b8b71291a6", 0},
	} {
		t.Run(fmt.Sprint(tc.size), func(t *testing.T) {
			budget, err := budgets.TransferBudget(tc.size)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), min(budget, 2*time.Minute))
			defer cancel()
			key, target := freshKey(t, 'u'), freshKey(t, 'o')
			grant, err := store.SignUpload(ctx, domain.UploadGrant{Key: key, Size: tc.size, ContentType: "text/plain", ExpiresAt: time.Now().Add(time.Minute)})
			if err != nil {
				t.Fatal(err)
			}
			snapshot := watchCapacityHeap(t)
			start := time.Now()
			body := &capacityBody{ctx: ctx, left: tc.size, pause: tc.pause}
			req, err := http.NewRequestWithContext(ctx, grant.Method, grant.URL, body)
			if err != nil {
				t.Fatal(err)
			}
			req.ContentLength = tc.size
			for k, v := range grant.Headers {
				req.Header.Set(k, v)
			}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			_, readErr := io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
			closeErr := resp.Body.Close()
			if resp.StatusCode != 200 || readErr != nil || closeErr != nil {
				t.Fatalf("PUT status=%d read=%v close=%v", resp.StatusCode, readErr, closeErr)
			}
			uploaded := time.Since(start)
			if tc.pause > 0 && (uploaded < tc.pause || !body.paused) {
				t.Fatal("slow PUT did not cross the real 15-second API boundary")
			}
			info, err := store.Inspect(ctx, key)
			if err != nil || info.Size != tc.size {
				t.Fatalf("actual staging size=%d error=%v", info.Size, err)
			}
			capacityDigest(t, store, info, tc.sha)
			final, err := store.Seal(ctx, key, target, info.ETag)
			if err != nil || final.Size != tc.size {
				t.Fatalf("actual copy size=%d error=%v", final.Size, err)
			}
			capacityDigest(t, store, final, tc.sha)
			total := time.Since(start)
			mem := snapshot()
			t.Logf("size=%d upload=%s roundtrip=%s uploadMiB/s=%.2f configuredTransferBudget=%s heapBaseline=%d peakHeap=%d peakHeapDelta=%d peakSys=%d totalAllocated=%d", tc.size, uploaded, total, float64(tc.size)/(1<<20)/uploaded.Seconds(), budget, mem.baseline, mem.peak, mem.peak-mem.baseline, mem.sys, mem.allocated)
			if tc.size == 100<<20 && mem.peak-mem.baseline >= 50<<20 {
				t.Fatalf("whole-file buffering: 100MiB transfer added %d heap bytes, want below50MiB", mem.peak-mem.baseline)
			}
			for _, object := range []domain.ObjectKey{key, target} {
				if err := store.Delete(ctx, object); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func capacityDigest(t *testing.T, store *Store, info domain.ObjectInfo, want string) {
	t.Helper()
	r, err := store.Open(t.Context(), info.Key, info.ETag)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	n, readErr := io.Copy(h, r)
	closeErr := r.Close()
	if readErr != nil || closeErr != nil || n != info.Size || hex.EncodeToString(h.Sum(nil)) != want {
		t.Fatalf("stream digest differs: bytes=%d read=%v close=%v hash=%x", n, readErr, closeErr, h.Sum(nil))
	}
}

type capacityBody struct {
	ctx          context.Context
	left         int64
	pause        time.Duration
	sent, paused bool
}

func (r *capacityBody) Read(p []byte) (int, error) {
	if r.left == 0 {
		return 0, io.EOF
	}
	if r.sent && !r.paused && r.pause > 0 {
		timer := time.NewTimer(r.pause)
		defer timer.Stop()
		select {
		case <-timer.C:
			r.paused = true
		case <-r.ctx.Done():
			return 0, r.ctx.Err()
		}
	}
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n := min(int64(len(p)), r.left)
	for i := range p[:n] {
		p[i] = 'a'
	}
	r.left -= n
	r.sent = true
	return int(n), nil
}

type capacityMemory struct{ baseline, peak, sys, allocated uint64 }

func watchCapacityHeap(t *testing.T) func() capacityMemory {
	t.Helper()
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	var peak, sys atomic.Uint64
	peak.Store(before.HeapAlloc)
	sys.Store(before.Sys)
	sample := func() {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		for old := peak.Load(); m.HeapAlloc > old; old = peak.Load() {
			if peak.CompareAndSwap(old, m.HeapAlloc) {
				break
			}
		}
		for old := sys.Load(); m.Sys > old; old = sys.Load() {
			if sys.CompareAndSwap(old, m.Sys) {
				break
			}
		}
	}
	stop, done := make(chan struct{}), make(chan struct{})
	var once sync.Once
	go func() {
		defer close(done)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				sample()
			}
		}
	}()
	finish := func() { once.Do(func() { sample(); close(stop); <-done }) }
	t.Cleanup(finish)
	return func() capacityMemory {
		finish()
		var after runtime.MemStats
		runtime.ReadMemStats(&after)
		return capacityMemory{baseline: before.HeapAlloc, peak: peak.Load(), sys: sys.Load(), allocated: after.TotalAlloc - before.TotalAlloc}
	}
}
