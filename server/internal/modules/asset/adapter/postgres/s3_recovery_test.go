package postgresadapter

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	assets3 "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/s3"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
)

func s3RecoveryConfig(b *s3test.Bucket) assets3.Config {
	return assets3.Config{Endpoint: b.Endpoint, Region: b.Region, Bucket: b.Name, Prefix: "recovery/", UsePathStyle: true, AccessKeyID: b.AccessKeyID, SecretAccessKey: b.SecretAccessKey, SizeLimit: 5 << 20, OperationTimeout: 10 * time.Second}
}
func openRecoveryS3(t *testing.T, cfg assets3.Config) *assets3.Store {
	t.Helper()
	s, err := assets3.New(t.Context(), cfg, assets3.Deps{RuntimeNow: time.Now})
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
func s3Key(t *testing.T, kind byte, id uuid.UUID) domain.ObjectKey {
	t.Helper()
	key, err := domain.NewObjectKey(kind, id, uuid.NewV7())
	if err != nil {
		t.Fatal(err)
	}
	return key
}
func putS3Recovery(t *testing.T, b *s3test.Bucket, key domain.ObjectKey, media string) {
	t.Helper()
	_, err := b.Client().PutObject(t.Context(), &awss3.PutObjectInput{Bucket: &b.Name, Key: aws.String("recovery/" + string(key)), Body: strings.NewReader("abc"), ContentType: &media})
	if err != nil {
		t.Fatal(err)
	}
}
func expectS3Recovery(t *testing.T, s *assets3.Store, key domain.ObjectKey, exists bool) {
	t.Helper()
	info, err := s.Inspect(t.Context(), key)
	if !exists {
		if !errors.Is(err, domain.ErrObjectNotFound) {
			t.Fatalf("abandoned bytes remain: %s %v", key, err)
		}
		return
	}
	if err != nil {
		t.Fatal("protected object lost", err)
	}
	reader, err := s.Open(t.Context(), key, info.ETag)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || string(body) != "abc" {
		t.Fatalf("object bytes=%q %v", body, err)
	}
}
func waitS3Held(t *testing.T, h *s3test.HeldRequest) {
	t.Helper()
	select {
	case <-h.Reached:
	case err := <-h.Result:
		t.Fatalf("held request failed: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("hold timeout")
	}
}
func releaseS3Held(t *testing.T, h *s3test.HeldRequest) {
	t.Helper()
	h.Release()
	select {
	case err := <-h.Result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("release timeout")
	}
}

func TestS3Recovery(t *testing.T) {
	service := s3test.New(t)
	t.Run("candidate removed before late COPY and runtime reconciliation", func(t *testing.T) { s3LateCandidate(t, service.NewBucket(t)) })
	t.Run("lost DELETE response retains durable candidate", func(t *testing.T) { s3LostDeleteResponse(t, service.NewBucket(t)) })
	t.Run("River token and partial page cross a fresh process", func(t *testing.T) { s3ProcessRecovery(t, service.NewBucket(t)) })
}

func s3LateCandidate(t *testing.T, b *s3test.Bucket) {
	w := newAssetWorld(t)
	w.now = time.Now().UTC()
	proxy := s3test.NewFaultProxy(t, b)
	cfg := s3RecoveryConfig(b)
	cfg.Endpoint = proxy.URL
	s := openRecoveryS3(t, cfg)
	source := w.asset(domain.ScopeIssue)
	source.Uploaded = true
	source.BoundAt = &w.now
	key := s3Key(t, 'o', source.ID)
	putS3Recovery(t, b, key, source.ContentType)
	info, err := s.Inspect(t.Context(), key)
	if err != nil {
		t.Fatal(err)
	}
	source.Metadata.Final = &domain.ObjectRecord{Key: key, Phase: domain.ObjectActive, Size: 3, ContentType: source.ContentType, ETag: info.ETag, SHA256: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", SafeAfter: w.now.Add(-time.Hour)}
	if _, err := w.store.CreateAsset(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	a := w.asset(domain.ScopeIssue)
	a.CreatedAt = w.now.Add(-8 * 24 * time.Hour)
	target := s3Key(t, 'o', a.ID)
	a.Metadata.Candidates = []domain.Candidate{{ObjectRecord: domain.ObjectRecord{Key: target, Phase: domain.ObjectActive, Size: 3, ContentType: a.ContentType, SafeAfter: w.now.Add(2 * time.Minute)}, LeaseUntil: w.now.Add(time.Minute)}}
	if _, err := w.store.CreateAsset(t.Context(), a); err != nil {
		t.Fatal(err)
	}
	held := proxy.HoldNext(t, "PUT", "/"+b.Name+"/recovery/"+string(target), s3test.BeforeRequest)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	copied := make(chan error, 1)
	go func() { _, err := s.Seal(ctx, key, target, info.ETag); copied <- err }()
	waitS3Held(t, held)
	cancel()
	if err := <-copied; !errors.Is(err, context.Canceled) {
		t.Fatalf("COPY client=%v", err)
	}
	business := w.now.Add(61 * 24 * time.Hour)
	r, err := w.cleanup(s, nil, business).Execute(t.Context(), app.SweepCursor{})
	if err != nil || !r.Complete || r.Removed != 0 {
		t.Fatalf("business clock crossed runtime lease: %+v %v", r, err)
	}
	got, err := w.store.FindAsset(t.Context(), a.ID)
	if err != nil || len(got.Metadata.Candidates) != 1 {
		t.Fatalf("active lease forgotten: %+v %v", got, err)
	}
	// Advance only the injected cleanup clock, not the provider clock. The
	// already-started provider request can still arrive after acknowledgment.
	w.now = w.now.Add(3 * time.Minute)
	r, err = w.cleanup(s, nil, business).Execute(t.Context(), app.SweepCursor{})
	if err != nil || !r.Complete || r.Removed != 1 {
		t.Fatalf("missing candidate not acknowledged: %+v %v", r, err)
	}
	releaseS3Held(t, held)
	info, err = s.Inspect(t.Context(), target)
	if err != nil {
		t.Fatal("late copy absent", err)
	}
	for _, offset := range []time.Duration{0, 24 * time.Hour, 24*time.Hour + time.Microsecond} {
		w.now = info.ModifiedAt.Add(offset)
		r, err = w.cleanup(s, nil, business).Execute(t.Context(), app.SweepCursor{})
		if err != nil || !r.Complete {
			t.Fatalf("reconcile %+v %v", r, err)
		}
		expectS3Recovery(t, s, target, offset <= 24*time.Hour)
		expectS3Recovery(t, s, key, true)
		got, err = w.store.FindAsset(t.Context(), a.ID)
		if err != nil || len(got.Metadata.Candidates) != 0 || got.Metadata.Final != nil || got.Uploaded || got.DeletedAt == nil {
			t.Fatalf("late key re-adopted by DB: %+v %v", got, err)
		}
	}
}

func s3LostDeleteResponse(t *testing.T, b *s3test.Bucket) {
	w := newAssetWorld(t)
	// Preserve a sub-microsecond fraction to exercise PostgreSQL timestamp precision.
	w.now = time.Now().UTC().Truncate(time.Microsecond).Add(789 * time.Nanosecond)
	proxy := s3test.NewFaultProxy(t, b)
	cfg := s3RecoveryConfig(b)
	cfg.Endpoint = proxy.URL
	s := openRecoveryS3(t, cfg)
	a := w.asset(domain.ScopeIssue)
	key := s3Key(t, 'o', a.ID)
	putS3Recovery(t, b, key, a.ContentType)
	a.Metadata.Candidates = []domain.Candidate{{ObjectRecord: domain.ObjectRecord{Key: key, Phase: domain.ObjectActive, Size: 3, ContentType: a.ContentType, SafeAfter: w.now.Add(-time.Minute)}, LeaseUntil: w.now.Add(-2 * time.Minute)}}
	// Recovery must preserve the persisted audit, whose precision is microseconds.
	a, err := w.store.CreateAsset(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	held := proxy.HoldNext(t, "DELETE", "/"+b.Name+"/recovery/"+string(key), s3test.BeforeResponse)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	type outcome struct {
		r   app.SweepResult
		err error
	}
	finished := make(chan outcome, 1)
	go func() {
		r, err := w.cleanup(s, nil, w.now).Execute(ctx, app.SweepCursor{})
		finished <- outcome{r, err}
	}()
	waitS3Held(t, held)
	if w.pool.Stat().AcquiredConns() != 0 {
		t.Fatal("external delete holds DB connection")
	}
	expectS3Recovery(t, s, key, false)
	cancel()
	first := <-finished
	if !errors.Is(first.err, context.Canceled) {
		t.Fatalf("lost response not canceled: %v", first.err)
	}
	got, err := w.store.FindAsset(t.Context(), a.ID)
	if err != nil || len(got.Metadata.Candidates) != 1 || got.Metadata.Candidates[0].Phase != domain.ObjectReclaiming {
		t.Fatalf("unacknowledged key lost: %+v %v", got, err)
	}
	releaseS3Held(t, held)
	_ = s.Close()
	s = openRecoveryS3(t, s3RecoveryConfig(b))
	w.store = New(w.pool)
	r, err := w.cleanup(s, nil, w.now).Execute(t.Context(), first.r.Cursor)
	if err != nil || !r.Complete || r.Removed != 1 {
		t.Fatalf("idempotent recovery: %+v %v", r, err)
	}
	got, err = w.store.FindAsset(t.Context(), a.ID)
	if err != nil || len(got.Metadata.Candidates) != 0 || got.DeletedAt != nil || !got.UpdatedAt.Equal(a.UpdatedAt) {
		t.Fatalf("retry lost audit (want updated_at %v): %+v %v", a.UpdatedAt, got, err)
	}
}
