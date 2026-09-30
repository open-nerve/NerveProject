package domain_test

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// The message is the label and the id's bytes, and a token the prefix and
// the tag in base64url: the known answer of the signing package's test
// (spec P3, appendix), computed by hand from RFC 5869 and RFC 4648.
func TestTheTokenOfTheKnownAnswer(t *testing.T) {
	id := uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	want := append([]byte("workspace-invitation"), 0x01, 0x99, 0xa2, 0xb4, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0, 0x01)
	if got := domain.InvitationMessage(id); !bytes.Equal(got, want) {
		t.Errorf("InvitationMessage() = %q, want %q", got, want)
	}
	tag, err := hex.DecodeString("92fab228187e6c034c4fa240218ce894")
	if err != nil {
		t.Fatal(err)
	}
	if got := domain.FormatToken([16]byte(tag)); got != "nrv_inv_kvqyKBh-bANMT6JAIYzolA" {
		t.Errorf("FormatToken() = %q, want nrv_inv_kvqyKBh-bANMT6JAIYzolA", got)
	}
}

// Two ids, two messages: the message holds the whole id.
func TestInvitationMessagesDifferByEveryByteOfTheID(t *testing.T) {
	id := uuid.NewV7()
	for i := range id {
		other := id
		other[i] ^= 0x80
		if bytes.Equal(domain.InvitationMessage(id), domain.InvitationMessage(other)) {
			t.Errorf("byte %d of the id is not in the message", i)
		}
	}
}

// randomTag is 16 random bytes.
func randomTag(t *testing.T) [16]byte {
	t.Helper()
	var tag [16]byte
	if _, err := rand.Read(tag[:]); err != nil {
		t.Fatal(err)
	}
	return tag
}

// A token is 30 characters, parses back to its tag, and is written in
// base64url: no "+", "/" or "=".
func TestATokenParsesBackToItsTag(t *testing.T) {
	for range 1000 {
		tag := randomTag(t)
		token := domain.FormatToken(tag)
		if got, ok := domain.ParseToken(token); !ok || got != tag || len(token) != 30 || strings.ContainsAny(token, "+/=") {
			t.Fatalf("FormatToken(%x) = %q, which parses to %x, %v", tag, token, got, ok)
		}
	}
}

// Changing any one bit of a token gives no token of the same tag: it does
// not parse, or it parses to another tag, which the MAC then refuses. The
// last character carries four bits of padding: a decoder that ignored them
// would take four other characters for it (M3 design 8.1).
func TestEveryBitOfATokenCounts(t *testing.T) {
	for range 64 {
		tag := randomTag(t)
		token := domain.FormatToken(tag)
		for i := range len(token) {
			for bit := range 8 {
				changed := []byte(token)
				changed[i] ^= 1 << bit
				if got, ok := domain.ParseToken(string(changed)); ok && got == tag {
					t.Fatalf("%q with bit %d of character %d changed, %q, parses to the same tag", token, bit, i, changed)
				}
			}
		}
	}
}

// Anything but "nrv_inv_" and 22 base64url characters of 16 bytes is no
// token. Go's base64 decoder skips "\r" and "\n", even strict: the length
// of the text refuses one added, and the length of the bytes two in place
// of two characters.
func TestParseTokenRefuses(t *testing.T) {
	valid := domain.FormatToken(randomTag(t))
	encoded := strings.TrimPrefix(valid, "nrv_inv_")
	for name, token := range map[string]string{
		"empty":                              "",
		"the prefix alone":                   "nrv_inv_",
		"without the prefix":                 encoded,
		"another prefix":                     "nrv_pat_" + encoded,
		"an upper-case prefix":               "NRV_INV_" + encoded,
		"one character short":                valid[:len(valid)-1],
		"one character more":                 valid + "A",
		"padded":                             valid[:len(valid)-2] + "==",
		"standard base64's +":                valid[:12] + "+" + valid[13:],
		"standard base64's /":                valid[:12] + "/" + valid[13:],
		"a new line in place of a character": valid[:12] + "\n" + valid[13:],
		"a new line inserted":                valid[:12] + "\n" + valid[12:],
		"a carriage return appended":         valid + "\r",
		"CRLF in place of two characters":    valid[:12] + "\r\n" + valid[14:],
		"a space inside":                     valid[:12] + " " + valid[13:],
		"padding bits set":                   "nrv_inv_AAAAAAAAAAAAAAAAAAAAAB",
		"a space before":                     " " + valid,
		"fifteen bytes":                      "nrv_inv_" + base64.RawURLEncoding.EncodeToString(make([]byte, 15)),
		"seventeen bytes":                    "nrv_inv_" + base64.RawURLEncoding.EncodeToString(make([]byte, 17)),
		"the token twice":                    valid + valid,
		"a multi-byte character last":        valid[:len(valid)-1] + "é",
	} {
		if tag, ok, panicked := parseToken(token); ok || panicked != nil {
			t.Errorf("%s: ParseToken(%q) = %x, %v, panic %v; want no token", name, token, tag, ok, panicked)
		}
	}
}

// parseToken is domain.ParseToken, with a panic returned instead of ending
// the test binary.
func parseToken(token string) (tag [16]byte, ok bool, panicked any) {
	defer func() { panicked = recover() }()
	tag, ok = domain.ParseToken(token)
	return tag, ok, nil
}
