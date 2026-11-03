package s3

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
)

func TestS3Inventory(t *testing.T) {
	t.Run("real pages and resume", realInventory)
	t.Run("wire validation", inventoryWire)
	t.Run("temporary contract", temporaryInventory)
}

func realInventory(t *testing.T) {
	b := s3test.New(t).NewBucket(t)
	s := realStore(t, b)
	keys := []domain.ObjectKey{freshKey(t, 'u'), freshKey(t, 'o'), freshKey(t, 'o')}
	slices.Sort(keys)
	put := func(key string) {
		t.Helper()
		_, err := b.Client().PutObject(t.Context(), &awss3.PutObjectInput{Bucket: &b.Name, Key: &key, Body: strings.NewReader("abc"), ContentType: aws.String("text/plain")})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, key := range keys {
		put(s.cfg.Prefix + string(key))
	}
	neighbor := "owned/instance-other/" + string(freshKey(t, 'u'))
	put(neighbor)
	first, err := s.List(t.Context(), "", 1)
	if err != nil || len(first.Objects) != 1 || first.Objects[0].Key != keys[0] || first.Next == "" {
		t.Fatalf("first bounded prefix page = %+v %v", first, err)
	}
	// Discard the client and delete the preceding page: the provider cursor
	// must continue even though this is not a snapshot.
	if err := s.Delete(t.Context(), keys[0]); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s = realStore(t, b)
	token := first.Next
	for i := 1; i < len(keys); i++ {
		page, err := s.List(t.Context(), token, 1)
		if err != nil || len(page.Objects) != 1 || page.Objects[0].Key != keys[i] {
			t.Fatalf("resumed page%d = %+v %v", i, page, err)
		}
		object := page.Objects[0]
		head, err := s.Inspect(t.Context(), object.Key)
		if err != nil || object.Size != 3 || object.ETag != head.ETag || !object.ModifiedAt.Equal(head.ModifiedAt) {
			t.Fatalf("inventory metadata differs: %+v %+v %v", object, head, err)
		}
		if (page.Next == "") != (i == len(keys)-1) || page.Next == token {
			t.Fatalf("cursor did not advance/exhaust: %q -> %q", token, page.Next)
		}
		token = page.Next
	}
	out, err := b.Client().GetObject(t.Context(), &awss3.GetObjectInput{Bucket: &b.Name, Key: &neighbor})
	if err != nil {
		t.Fatal("neighbor was touched", err)
	}
	_ = out.Body.Close()
	_, err = b.Client().DeleteBucket(t.Context(), &awss3.DeleteBucketInput{Bucket: &b.Name})
	if err == nil {
		t.Fatal("fixture unexpectedly deleted a nonempty bucket")
	}
}

func temporaryInventory(t *testing.T) {
	s := wireInventoryStore(t, "", nil)
	for _, tc := range []struct {
		name  string
		at    time.Time
		limit int
		token string
		bad   bool
	}{
		{"empty", time.Now(), 1, "", false},
		{"max", time.Now(), 1000, "", false},
		{"zero cutoff", time.Time{}, 1, "", true},
		{"zero limit", time.Now(), 0, "", true},
		{"large limit", time.Now(), 1001, "", true},
		{"unexpected token", time.Now(), 1, "opaque", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next, removed, err := s.SweepTemporary(t.Context(), tc.at, tc.limit, tc.token)
			if (err != nil) != tc.bad || next != "" || removed != 0 {
				t.Fatalf("temporary = %q %d %v", next, removed, err)
			}
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := s.SweepTemporary(ctx, time.Now(), 1, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation = %v", err)
	}
	if _, err := s.List(ctx, "", 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("list cancellation = %v", err)
	}
	_ = s.Close()
	if _, _, err := s.SweepTemporary(t.Context(), time.Now(), 1, ""); err == nil {
		t.Fatal("closed temporary cleaner accepted")
	}
}
