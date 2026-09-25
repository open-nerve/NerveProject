package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
	"uuid"
)

func sampleToken() RefreshToken {
	t := RefreshToken{SessionID: uuid.MustParse("0199a2b4-7c3e-7d2a-9f10-2b3c4d5e6f70"), Generation: 0x01020304}
	for i := range t.Secret {
		t.Secret[i] = byte(0x40 + i)
	}
	for i := range t.Tag {
		t.Tag[i] = byte(0xa0 + i)
	}
	return t
}

// The 68 bytes: session id 0–15, generation 16–19 big endian, secret 20–51,
// tag 52–67 (M2 design 3.4).
func TestRefreshTokenLayout(t *testing.T) {
	tok := sampleToken()
	s := tok.String()

	// The secret-scanning pattern of M2 design 8.6.
	if !regexp.MustCompile(`^nrv_rt_[A-Za-z0-9_-]{91}$`).MatchString(s) {
		t.Fatalf("token %q does not match nrv_rt_[A-Za-z0-9_-]{91}", s)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(s, RefreshTokenPrefix))
	if err != nil || len(raw) != 68 {
		t.Fatalf("decoded %d bytes, %v; want 68", len(raw), err)
	}
	if !bytes.Equal(raw[0:16], tok.SessionID[:]) || !bytes.Equal(raw[16:20], []byte{1, 2, 3, 4}) ||
		!bytes.Equal(raw[20:52], tok.Secret[:]) || !bytes.Equal(raw[52:68], tok.Tag[:]) {
		t.Errorf("layout = % x", raw)
	}
	if !bytes.Equal(tok.MACMessage(), raw[:52]) {
		t.Errorf("MACMessage() = % x, want the first 52 bytes", tok.MACMessage())
	}
}

func TestRefreshTokenSecretHash(t *testing.T) {
	tok := sampleToken()
	want := sha256.Sum256(tok.Secret[:])
	if got := tok.SecretHash(); !bytes.Equal(got, want[:]) || len(got) != 32 {
		t.Errorf("SecretHash() = % x, want the SHA-256 of the secret", got)
	}
}

func TestSanitizeUserAgent(t *testing.T) {
	long := strings.Repeat("é", 600)
	tests := []struct{ in, want string }{
		{"Mozilla/5.0", "Mozilla/5.0"},
		{"agent\x00/1", "agent/1"},
		{"bad \xff byte", "bad " + string(utf8.RuneError) + " byte"},
		{long, strings.Repeat("é", 512)},
	}
	for _, tt := range tests {
		got := SanitizeUserAgent(tt.in)
		if got != tt.want || !utf8.ValidString(got) {
			t.Errorf("SanitizeUserAgent(%.20q) = %.20q (%d runes), want %.20q", tt.in, got, utf8.RuneCountInString(got), tt.want)
		}
	}
}
