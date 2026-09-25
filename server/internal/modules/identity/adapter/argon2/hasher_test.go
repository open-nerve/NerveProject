package argon2adapter

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// testParams are config.test.yaml's: m = 64 KiB, t = 1.
var testParams = Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, MaxConcurrent: 2, MaxWait: 50 * time.Millisecond}

func TestHashIsAnArgon2idPHCString(t *testing.T) {
	phc, err := New(testParams, slog.New(slog.DiscardHandler)).Hash(context.Background(), "Tr0ub4dor&3")
	if err != nil {
		t.Fatal(err)
	}
	var version int
	var m, iterations uint32
	var p uint8
	var salt64, key64 string
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		t.Fatalf("hash %q is not $argon2id$v$params$salt$key", phc)
	}
	if _, err := fmt.Sscanf(parts[2]+" "+parts[3], "v=%d m=%d,t=%d,p=%d", &version, &m, &iterations, &p); err != nil {
		t.Fatalf("hash %q: %v", phc, err)
	}
	salt64, key64 = parts[4], parts[5]
	salt, errSalt := base64.RawStdEncoding.DecodeString(salt64)
	key, errKey := base64.RawStdEncoding.DecodeString(key64)
	if version != argon2.Version || m != 64 || iterations != 1 || p != 1 || errSalt != nil || errKey != nil || len(salt) != 16 || len(key) != 32 {
		t.Errorf("hash %q: v=%d m=%d t=%d p=%d, salt %d bytes, key %d bytes", phc, version, m, iterations, p, len(salt), len(key))
	}
	// What login (M2/P2) will do: the same derivation gives the same key.
	if again := argon2.IDKey([]byte("Tr0ub4dor&3"), salt, iterations, m, p, 32); subtle.ConstantTimeCompare(again, key) != 1 {
		t.Error("re-deriving the key from the PHC parameters gives another key")
	}
	if len(phc) > 128 {
		t.Errorf("hash is %d characters; users.password is varchar(128)", len(phc))
	}
}

func TestHashSaltsEveryHash(t *testing.T) {
	h := New(testParams, slog.New(slog.DiscardHandler))
	a, _ := h.Hash(context.Background(), "Tr0ub4dor&3")
	b, _ := h.Hash(context.Background(), "Tr0ub4dor&3")
	if a == b {
		t.Error("two hashes of one password are equal")
	}
}

// With every slot taken, a caller waits max_wait and gets 503 server_busy
// with Retry-After: 1; the queue length goes to the log (M2 design 3.8, 8.4).
func TestHashIsBusyWhenNoSlotFreesUp(t *testing.T) {
	var logs bytes.Buffer
	h := New(testParams, slog.New(slog.NewJSONHandler(&logs, nil)))
	h.slots <- struct{}{}
	h.slots <- struct{}{}

	start := time.Now()
	_, err := h.Hash(context.Background(), "Tr0ub4dor&3")

	var se *shared.Error
	if !errors.As(err, &se) || se.ProblemStatus() != 503 || se.Code != shared.CodeServerBusy || se.RetryAfter() != time.Second {
		t.Errorf("Hash() = %v, want 503 server_busy with Retry-After 1s", err)
	}
	if waited := time.Since(start); waited < testParams.MaxWait {
		t.Errorf("gave up after %v, want at least max_wait %v", waited, testParams.MaxWait)
	}
	if !strings.Contains(logs.String(), `"msg":"password hashing is saturated","queued":1`) {
		t.Errorf("logs = %s, want the saturation with the queue length", logs.String())
	}
}

func TestHashWaitsForAFreedSlot(t *testing.T) {
	h := New(Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, MaxConcurrent: 1, MaxWait: 5 * time.Second}, slog.New(slog.DiscardHandler))
	h.slots <- struct{}{}
	time.AfterFunc(20*time.Millisecond, func() { <-h.slots })

	if _, err := h.Hash(context.Background(), "Tr0ub4dor&3"); err != nil {
		t.Errorf("Hash() = %v, want a hash once the slot frees up", err)
	}
	if len(h.slots) != 0 {
		t.Errorf("%d slots still taken, want the hash to release its slot", len(h.slots))
	}
}

func TestHashStopsWaitingWhenTheRequestEnds(t *testing.T) {
	h := New(Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, MaxConcurrent: 1, MaxWait: 5 * time.Second}, slog.New(slog.DiscardHandler))
	h.slots <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)

	if _, err := h.Hash(ctx, "Tr0ub4dor&3"); !errors.Is(err, context.Canceled) {
		t.Errorf("Hash() = %v, want context.Canceled", err)
	}
}

// BenchmarkHashDefaultParams measures one hash with the default parameters
// (m = 19456 KiB, t = 2, p = 1): go test -bench . -run ^$ ./internal/modules/identity/adapter/argon2
func BenchmarkHashDefaultParams(b *testing.B) {
	h := New(Params{MemoryKiB: 19456, Iterations: 2, Parallelism: 1, MaxConcurrent: 1, MaxWait: time.Second}, slog.New(slog.DiscardHandler))
	for b.Loop() {
		if _, err := h.Hash(context.Background(), "Tr0ub4dor&3"); err != nil {
			b.Fatal(err)
		}
	}
}
