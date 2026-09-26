package argon2adapter

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"slices"
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
	// Any argon2id implementation reads the string: the same derivation
	// gives the same key.
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

func TestVerify(t *testing.T) {
	h := New(testParams, slog.New(slog.DiscardHandler))
	hash, err := h.Hash(context.Background(), "Tr0ub4dor&3")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, password string
		ok             bool
	}{
		{"the password", "Tr0ub4dor&3", true},
		{"another password", "Tr0ub4dor&4", false},
		{"the password with a space", "Tr0ub4dor&3 ", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		ok, rehash, err := h.Verify(context.Background(), tt.password, hash)
		if ok != tt.ok || rehash || err != nil {
			t.Errorf("%s: Verify() = %v, rehash %v, %v; want %v, no rehash", tt.name, ok, rehash, err, tt.ok)
		}
	}
}

// A hash with other parameters still verifies, and asks for a new hash
// with the current ones (M2 design 3.8).
func TestVerifyAsksForARehashWhenTheParametersChanged(t *testing.T) {
	old, err := New(testParams, slog.New(slog.DiscardHandler)).Hash(context.Background(), "Tr0ub4dor&3")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []Params{
		{MemoryKiB: 128, Iterations: 1, Parallelism: 1, MaxConcurrent: 1, MaxWait: time.Second},
		{MemoryKiB: 64, Iterations: 2, Parallelism: 1, MaxConcurrent: 1, MaxWait: time.Second},
		{MemoryKiB: 64, Iterations: 1, Parallelism: 2, MaxConcurrent: 1, MaxWait: time.Second},
	} {
		ok, rehash, err := New(p, slog.New(slog.DiscardHandler)).Verify(context.Background(), "Tr0ub4dor&3", old)
		if !ok || !rehash || err != nil {
			t.Errorf("Verify() with m=%d t=%d p=%d = %v, rehash %v, %v; want true, rehash", p.MemoryKiB, p.Iterations, p.Parallelism, ok, rehash, err)
		}
	}
}

func TestVerifyRejectsAnotherFormat(t *testing.T) {
	h := New(testParams, slog.New(slog.DiscardHandler))
	good, err := h.Hash(context.Background(), "Tr0ub4dor&3")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(good, "$")
	with := func(i int, s string) string {
		p := slices.Clone(parts)
		p[i] = s
		return strings.Join(p, "$")
	}
	tests := []struct{ name, hash string }{
		{"empty", ""},
		{"bcrypt", "$2b$10$abcdefghijklmnopqrstuuABCDEFGHIJKLMNOPQRSTUVWXYZ01234"},
		{"argon2i", with(1, "argon2i")},
		{"another version", with(2, "v=16")},
		{"no parameters", with(3, "")},
		{"parameters in another order", with(3, "t=1,m=64,p=1")},
		{"an extra parameter", with(3, "m=64,t=1,p=1,k=2")},
		{"zero iterations", with(3, "m=64,t=0,p=1")},
		{"zero lanes", with(3, "m=64,t=1,p=0")},
		{"lanes beyond a byte", with(3, "m=64,t=1,p=256")},
		{"memory beyond 32 bits", with(3, "m=4294967360,t=1,p=1")},
		{"a signed parameter", with(3, "m=+64,t=1,p=1")},
		{"a short salt", with(4, parts[4][:10])},
		{"a salt with padding", with(4, parts[4]+"==")},
		// The same 16 bytes, with an unused bit of the last character set.
		{"a salt in another spelling", with(4, parts[4][:21]+string(rune(parts[4][21]+1)))},
		{"a short key", with(5, parts[5][:20])},
		{"an empty key", with(5, "")},
		// The same 32 bytes, with an unused bit of the last character set.
		{"a key in another spelling", with(5, parts[5][:42]+string(rune(parts[5][42]+1)))},
		{"a trailing part", good + "$x"},
		{"a leading part", "x" + good},
	}
	for _, tt := range tests {
		ok, rehash, err := h.Verify(context.Background(), "Tr0ub4dor&3", tt.hash)
		if ok || rehash || !errors.Is(err, errNotOurHash) {
			t.Errorf("%s: Verify() = %v, rehash %v, %v; want the format error", tt.name, ok, rehash, err)
		}
		if tt.hash != "" && strings.Contains(err.Error(), tt.hash) {
			t.Errorf("%s: the error %q quotes the hash", tt.name, err)
		}
	}
}

// Verify takes a slot like Hash: a login waits and gets 503 the same way,
// whether its address exists or not.
func TestVerifyIsBusyWhenNoSlotFreesUp(t *testing.T) {
	h := New(testParams, slog.New(slog.DiscardHandler))
	hash, err := h.Hash(context.Background(), "Tr0ub4dor&3")
	if err != nil {
		t.Fatal(err)
	}
	h.slots <- struct{}{}
	h.slots <- struct{}{}

	_, _, err = h.Verify(context.Background(), "Tr0ub4dor&3", hash)

	var se *shared.Error
	if !errors.As(err, &se) || se.Code != shared.CodeServerBusy {
		t.Errorf("Verify() = %v, want 503 server_busy", err)
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
