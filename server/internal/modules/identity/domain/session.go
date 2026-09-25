package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"strings"
	"unicode/utf8"
	"uuid"
)

// RefreshTokenPrefix starts every refresh token (M2 design 3.4).
const RefreshTokenPrefix = "nrv_rt_"

// The layout of a refresh token's 68 bytes (M2 design 3.4): session id,
// generation (big endian), secret, and the MAC tag of the first 52 bytes.
const (
	refreshTokenLen = 16 + 4 + 32 + 16
	macMessageLen   = 16 + 4 + 32
)

// RefreshToken is the content of a refresh token. The server stores only
// SecretHash of the current generation; Tag proves that the server issued a
// generation without storing it.
type RefreshToken struct {
	SessionID  uuid.UUID
	Generation uint32
	Secret     [32]byte
	Tag        [16]byte
}

// MACMessage is what the tag authenticates: session id ‖ generation ‖ secret.
func (t RefreshToken) MACMessage() []byte {
	return t.bytes()[:macMessageLen]
}

// SecretHash is the SHA-256 of the secret, what auth_sessions.token_hash holds.
func (t RefreshToken) SecretHash() []byte {
	h := sha256.Sum256(t.Secret[:])
	return h[:]
}

// String is the token the client holds: nrv_rt_ and the 68 bytes in
// unpadded base64url, 91 characters.
func (t RefreshToken) String() string {
	return RefreshTokenPrefix + base64.RawURLEncoding.EncodeToString(t.bytes())
}

func (t RefreshToken) bytes() []byte {
	b := make([]byte, 0, refreshTokenLen)
	b = append(b, t.SessionID[:]...)
	b = binary.BigEndian.AppendUint32(b, t.Generation)
	b = append(b, t.Secret[:]...)
	return append(b, t.Tag[:]...)
}

// MaxUserAgentLength bounds auth_sessions.user_agent, in characters.
const MaxUserAgentLength = 512

// SanitizeUserAgent is the User-Agent a session records: valid UTF-8, no
// NUL (Postgres text cannot hold it), at most 512 characters.
func SanitizeUserAgent(ua string) string {
	ua = strings.ReplaceAll(strings.ToValidUTF8(ua, string(utf8.RuneError)), "\x00", "")
	if utf8.RuneCountInString(ua) <= MaxUserAgentLength {
		return ua
	}
	return string([]rune(ua)[:MaxUserAgentLength])
}
