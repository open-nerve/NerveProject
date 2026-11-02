// Package storagetest holds fixtures used only by asset storage tests.
package storagetest

import (
	"crypto/hmac"
	"crypto/sha256"
)

// MAC uses explicit test keys. Bootstrap tests separately exercise real HKDF.
type MAC string

func (m MAC) Tag(message []byte) [16]byte {
	h := hmac.New(sha256.New, []byte(m))
	_, _ = h.Write(message)
	return [16]byte(h.Sum(nil)[:16])
}

func (m MAC) Verify(message []byte, tag [16]byte) bool {
	want := m.Tag(message)
	return hmac.Equal(want[:], tag[:])
}
