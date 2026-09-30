package signing

import (
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
)

// PurposeRefreshToken is the purpose of the refresh tokens' MAC (M2 design
// 3.4): its key's HKDF info is "nerve refresh-token mac v1".
const PurposeRefreshToken = "refresh-token"

// MAC tags messages for one purpose: the first 16 bytes of HMAC-SHA256 under
// K = HKDF-SHA256(the signing key's seed, info "nerve <purpose> mac v1")
// (M2 design 3.4, M3 design 3.8). A key derived for one purpose is no other
// purpose's and not the signing key, so a tag made for one purpose verifies
// for no other. It implements identity's app.RefreshTokenMAC and, through
// identity.Keys, the workspace module's InvitationMAC.
//
// The MAC leaves this package, yet no fmt verb or log handler prints its
// key: Format prints the type alone, and where fmt reaches the MAC without
// Format, through another value's unexported field, it prints the MAC's
// fields at most, and the key's is a pointer, which it prints as an address.
type MAC struct {
	key *[32]byte
}

// MAC returns the MAC of purpose, its key derived from the signing key's
// seed.
func (k *Keys) MAC(purpose string) (*MAC, error) {
	key, err := hkdf.Key(sha256.New, k.private.Seed(), nil, "nerve "+purpose+" mac v1", 32)
	if err != nil {
		return nil, fmt.Errorf("derive the %s MAC key: %w", purpose, err)
	}
	return &MAC{key: (*[32]byte)(key)}, nil
}

// Format prints the MAC as its type alone, whatever the verb.
func (m *MAC) Format(f fmt.State, _ rune) {
	_, _ = io.WriteString(f, "signing.MAC(redacted)")
}

// Tag returns the tag of message.
func (m *MAC) Tag(message []byte) [16]byte {
	h := hmac.New(sha256.New, m.key[:])
	h.Write(message)
	return [16]byte(h.Sum(nil)[:16])
}

// Verify reports whether tag is the tag of message. It compares in constant
// time: the time of a comparison would tell a forger how much of a tag is
// right (M2 design 3.9).
func (m *MAC) Verify(message []byte, tag [16]byte) bool {
	want := m.Tag(message)
	return hmac.Equal(want[:], tag[:])
}
