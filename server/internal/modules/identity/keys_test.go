package identity_test

import (
	"bytes"
	"encoding/hex"
	"log/slog"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
)

// keyPEM is what `openssl genpkey -algorithm ed25519` writes: the key of
// the signing package's tests, for tests only.
const keyPEM = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIGqen6oN2FFQjS+yPQPHLVBIW0B2O9faCmNwftOWxqyE
-----END PRIVATE KEY-----
`

// invitationMessage is "workspace-invitation" and the invitation id
// 0199a2b4-0000-7000-8000-000000000001, the message of the signing
// package's known answer.
var invitationMessage = append([]byte("workspace-invitation"), 0x01, 0x99, 0xa2, 0xb4, 0, 0, 0x70, 0, 0x80, 0, 0, 0, 0, 0, 0, 0x01)

// A purpose's MAC is the one derived for that purpose from the key loaded:
// the known answer of the signing package's test for the key of the file,
// the same for the same file, another for another key.
func TestKeysMACIsThePurposes(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	tagOf := func(pem []byte) [16]byte {
		t.Helper()
		keys, err := identity.LoadKeys(pem, logger)
		if err != nil {
			t.Fatal(err)
		}
		mac, err := keys.MAC("workspace-invitation")
		if err != nil {
			t.Fatal(err)
		}
		return mac.Tag(invitationMessage)
	}
	tag := tagOf([]byte(keyPEM))
	if hex.EncodeToString(tag[:]) != "92fab228187e6c034c4fa240218ce894" {
		t.Errorf("the workspace-invitation tag = %x, want the known answer", tag)
	}
	if tagOf([]byte(keyPEM)) != tag {
		t.Error("the same key file gives another tag")
	}
	if ephemeral := tagOf(nil); ephemeral == tag || tagOf(nil) == ephemeral {
		t.Error("an ephemeral key gives the file's tag, or two ephemeral keys the same")
	}
}

// Without a file the key is ephemeral, and LoadKeys warns that links stop
// verifying at restart (M3 design 3.8).
func TestLoadKeysWithoutAFileWarns(t *testing.T) {
	var logs bytes.Buffer
	if _, err := identity.LoadKeys(nil, slog.New(slog.NewTextHandler(&logs, nil))); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs.String(), "access tokens and invitation links stop verifying at restart") {
		t.Errorf("logs: %s; want the ephemeral key's warning", logs.String())
	}
}

// The refresh tokens' MAC is identity's own: no other module is given its
// key, so none can make a refresh token's tag.
func TestKeysMACRefusesTheRefreshTokensPurpose(t *testing.T) {
	keys, err := identity.LoadKeys([]byte(keyPEM), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if mac, err := keys.MAC("refresh-token"); err == nil || mac != nil {
		t.Errorf("MAC(refresh-token) = %v, %v; want an error", mac, err)
	}
}
