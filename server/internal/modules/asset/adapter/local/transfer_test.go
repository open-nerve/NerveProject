package local

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLocalTransferReaderOwnsFile(t *testing.T) {
	s := newStorageTestStore(t, t.TempDir())
	g := uploadGrant(t, s, 6)
	if err := s.Put(t.Context(), g, strings.NewReader("abcdef")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	info, reader, err := s.OpenTransfer(ctx, g.Key)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	other, err := s.Open(t.Context(), g.Key, info.ETag)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = other.Close() }()
	if offset, err := reader.Seek(-2, io.SeekEnd); err != nil || offset != 4 {
		t.Fatalf("Seek = %d, %v", offset, err)
	}
	data, err := io.ReadAll(reader)
	if err != nil || string(data) != "ef" {
		t.Fatalf("tail = %q, %v", data, err)
	}
	cancel()
	if _, err := reader.Seek(0, io.SeekStart); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Seek = %v", err)
	}
	data, err = io.ReadAll(other)
	if err != nil || string(data) != "abcdef" {
		t.Fatalf("other reader = %q, %v", data, err)
	}
}
