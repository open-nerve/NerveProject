package identity_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
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
// the same for the same file, another for another key or another purpose.
func TestKeysMACIsThePurposes(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	tagOf := func(pem []byte, purpose string) [16]byte {
		t.Helper()
		keys, err := identity.LoadKeys(pem, logger)
		if err != nil {
			t.Fatal(err)
		}
		mac, err := keys.MAC(purpose)
		if err != nil {
			t.Fatal(err)
		}
		return mac.Tag(invitationMessage)
	}
	tag := tagOf([]byte(keyPEM), "workspace-invitation")
	if hex.EncodeToString(tag[:]) != "92fab228187e6c034c4fa240218ce894" {
		t.Errorf("the workspace-invitation tag = %x, want the known answer", tag)
	}
	if tagOf([]byte(keyPEM), "workspace-invitation") != tag {
		t.Error("the same key file gives another tag")
	}
	if tagOf([]byte(keyPEM), "another-purpose") == tag {
		t.Error("another purpose gives the workspace-invitation tag of the same message")
	}
	if ephemeral := tagOf(nil, "workspace-invitation"); ephemeral == tag || tagOf(nil, "workspace-invitation") == ephemeral {
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

// Neither the keys nor a MAC of theirs shows a key when printed or logged,
// alone or held in an unexported field the way a module holds its MAC: the
// signing key's seed and the MAC's key are in no fmt verb's output and no
// log handler's.
func TestKeysAndTheirMACsShowNoKey(t *testing.T) {
	keys, err := identity.LoadKeys([]byte(keyPEM), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	mac, err := keys.MAC("workspace-invitation")
	if err != nil {
		t.Fatal(err)
	}
	seed := seedOf(t, keyPEM)
	macKey, err := hkdf.Key(sha256.New, seed, nil, "nerve workspace-invitation mac v1", 32)
	if err != nil {
		t.Fatal(err)
	}
	secrets := map[string][]byte{"the signing key's seed": seed, "the MAC's key": macKey}
	holder := struct {
		mac  identity.MAC
		keys *identity.Keys
	}{mac, keys}
	handlers := map[string]func(io.Writer) slog.Handler{
		"the text log": func(w io.Writer) slog.Handler { return slog.NewTextHandler(w, nil) },
		"the JSON log": func(w io.Writer) slog.Handler { return slog.NewJSONHandler(w, nil) },
	}
	for name, value := range map[string]any{"the MAC": mac, "the keys": keys, "a struct holding them": holder} {
		outputs := map[string]string{}
		for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%d"} {
			outputs[verb] = fmt.Sprintf(verb, value)
		}
		for handler, newHandler := range handlers {
			var logs bytes.Buffer
			slog.New(newHandler(&logs)).Info("printed", "value", value)
			outputs[handler] = logs.String()
		}
		for how, out := range outputs {
			for secret, b := range secrets {
				if shows(out, b) {
					t.Errorf("%s, %s: %q shows %s", name, how, out, secret)
				}
			}
		}
	}
}

// seedOf is the seed of the Ed25519 key of the PKCS#8 PEM keyPEM.
func seedOf(t *testing.T, keyPEM string) []byte {
	t.Helper()
	block, _ := pem.Decode([]byte(keyPEM))
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return key.(ed25519.PrivateKey).Seed()
}

// shows reports whether out holds b the way fmt or a log handler writes
// bytes, alone or in a field: as they are, in hex, as numbers (%v, %d), as
// Go syntax (%#v), quoted (%q) or in base64 (JSON).
func shows(out string, b []byte) bool {
	for _, r := range []string{
		string(b),
		hex.EncodeToString(b),
		strings.ToUpper(hex.EncodeToString(b)),
		strings.Trim(fmt.Sprintf("%d", b), "[]"),
		strings.TrimPrefix(fmt.Sprintf("%#v", b), "[]byte"),
		strings.Trim(fmt.Sprintf("%q", b), `"`),
		base64.StdEncoding.EncodeToString(b),
	} {
		if strings.Contains(out, r) {
			return true
		}
	}
	return false
}
