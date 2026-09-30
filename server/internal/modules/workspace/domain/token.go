package domain

import (
	"encoding/base64"
	"strings"
	"uuid"
)

// An invitation's token (M3 design 3.8, 8.1) is "nrv_inv_" and 22 base64url
// characters without padding: the 16-byte tag of the invitation MAC on
// InvitationMessage. The server never stores it; it computes it again from
// the invitation's id.
const (
	tokenPrefix = "nrv_inv_"
	// tokenLabel begins every message the invitation MAC tags.
	tokenLabel = "workspace-invitation"
)

// tokenEncoding is base64url without padding, strict: a last character with
// any of its four low bits set is refused, so that one tag has one token.
var tokenEncoding = base64.RawURLEncoding.Strict()

// InvitationMessage is the message an invitation's token tags:
// "workspace-invitation", then the id's 16 bytes.
func InvitationMessage(id uuid.UUID) []byte {
	return append([]byte(tokenLabel), id[:]...)
}

// FormatToken is the token that carries tag.
func FormatToken(tag [16]byte) string {
	return tokenPrefix + tokenEncoding.EncodeToString(tag[:])
}

// ParseToken returns the tag token carries. ok is false for anything but
// "nrv_inv_" and the 22 characters FormatToken writes for some tag.
func ParseToken(token string) (tag [16]byte, ok bool) {
	encoded, found := strings.CutPrefix(token, tokenPrefix)
	if !found || len(encoded) != 22 {
		return tag, false
	}
	b, err := tokenEncoding.DecodeString(encoded)
	if err != nil || len(b) != len(tag) {
		return tag, false
	}
	return [16]byte(b), true
}
