package identity_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/signing"
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

// Neither the keys nor a MAC of theirs shows a key when printed or logged:
// the signing key's seed, which begins its private key, and the MAC's key
// are in no output (outputsOf) of identity's Keys and MAC, or of signing's
// Keys, MAC and AccessTokens, each as a value and a pointer, alone and held
// in another value's field, exported and unexported, that value printed as
// a value and a pointer. A field of type any holds a value as a field of
// its type would: fmt looks through the interface to it, calls its methods
// only when the field is exported, and so does encoding/json. The one form
// left out is the one signing.Keys and signing.MAC name: a value of theirs,
// not a pointer, in an unexported field, which no code makes. And each
// Format prints its type alone under every verb, for a value as for a
// pointer.
func TestKeysAndTheirMACsShowNoKey(t *testing.T) {
	keys, err := identity.LoadKeys([]byte(keyPEM), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	mac, err := keys.MAC("workspace-invitation")
	if err != nil {
		t.Fatal(err)
	}
	signingKeys, err := signing.ParseKeys([]byte(keyPEM))
	if err != nil {
		t.Fatal(err)
	}
	signingMAC, err := signingKeys.MAC("workspace-invitation")
	if err != nil {
		t.Fatal(err)
	}
	tokens := signing.NewAccessTokens(signingKeys)
	seed := seedOf(t, keyPEM)
	macKey, err := hkdf.Key(sha256.New, seed, nil, "nerve workspace-invitation mac v1", 32)
	if err != nil {
		t.Fatal(err)
	}
	secrets := map[string][]byte{"the signing key's seed": seed, "the MAC's key": macKey}
	type exported struct{ Held any }
	type unexported struct{ held any }
	for name, value := range map[string]any{"identity.Keys": *keys, "*identity.Keys": keys, "signing.Keys": *signingKeys,
		"*signing.Keys": signingKeys, "signing.MAC": *signingMAC, "*signing.MAC": signingMAC} {
		want := strings.TrimPrefix(name, "*") + "(redacted)"
		for _, verb := range verbs {
			if got := fmt.Sprintf(verb, value); got != want {
				t.Errorf("%s, %s: %q, want %q", name, verb, got, want)
			}
		}
	}
	forms := map[string]any{}
	for name, value := range map[string]any{
		"identity.Keys": *keys, "*identity.Keys": keys, "identity.MAC": mac,
		"signing.Keys": *signingKeys, "*signing.Keys": signingKeys, "signing.MAC": *signingMAC, "*signing.MAC": signingMAC,
		"signing.AccessTokens": *tokens, "*signing.AccessTokens": tokens,
	} {
		forms[name] = value
		forms[name+" in an exported field"] = exported{value}
		forms[name+" in an exported field, the holder by pointer"] = &exported{value}
		if name == "signing.Keys" || name == "signing.MAC" {
			continue
		}
		forms[name+" in an unexported field"] = unexported{value}
		forms[name+" in an unexported field, the holder by pointer"] = &unexported{value}
	}
	for name, value := range forms {
		for how, out := range outputsOf(value) {
			for secret, b := range secrets {
				if shows(out, b) {
					t.Errorf("%s, %s: %q shows %s", name, how, out, secret)
				}
			}
		}
	}
}

// verbs are the fmt verbs a value is printed with: every one a byte slice
// has a form of its own under.
var verbs = []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%X", "%d"}

// outputsOf is value as each of verbs prints it, as each log handler
// writes it, as encoding/json encodes it, and in an error that %w wraps
// (vet refuses %w of a value that is not an error).
func outputsOf(value any) map[string]string {
	outputs := map[string]string{}
	for _, verb := range verbs {
		outputs[verb] = fmt.Sprintf(verb, value)
	}
	for handler, newHandler := range map[string]func(io.Writer) slog.Handler{
		"the text log": func(w io.Writer) slog.Handler { return slog.NewTextHandler(w, nil) },
		"the JSON log": func(w io.Writer) slog.Handler { return slog.NewJSONHandler(w, nil) },
	} {
		var logs bytes.Buffer
		slog.New(newHandler(&logs)).Info("printed", "value", value)
		outputs[handler] = logs.String()
	}
	encoded, err := json.Marshal(value)
	outputs["encoding/json"] = fmt.Sprint(string(encoded), " ", err)
	outputs["%w"] = fmt.Errorf("wrapped: %w", fmt.Errorf("holding %s", value)).Error()
	return outputs
}

// shows finds a secret wherever fmt, a log handler or encoding/json writes
// it, also when it begins a longer slice: a value holding the seed, and one
// holding the seed and a byte more, show it in every output.
func TestShowsFindsASecretInEveryOutput(t *testing.T) {
	seed := seedOf(t, keyPEM)
	for name, value := range map[string]any{
		"the seed":              struct{ Key []byte }{seed},
		"the seed and one more": struct{ Key []byte }{append(bytes.Clone(seed), 0xff)},
	} {
		for how, out := range outputsOf(value) {
			if !shows(out, seed) {
				t.Errorf("%s, %s: shows(%q) = false", name, how, out)
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
// bytes, alone, in a field, or at the start of a longer slice: as they are,
// in hex (%x, %X), as numbers (%v, %d), as Go syntax (%#v), quoted (%q) or
// in base64 (JSON), whose whole groups of three bytes are the same whatever
// follows them.
func shows(out string, b []byte) bool {
	for _, r := range []string{
		string(b),
		hex.EncodeToString(b),
		strings.ToUpper(hex.EncodeToString(b)),
		strings.Trim(fmt.Sprintf("%d", b), "[]"),
		strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf("%#v", b), "[]byte"), "}"),
		strings.Trim(fmt.Sprintf("%q", b), `"`),
		base64.StdEncoding.EncodeToString(b[:len(b)/3*3]),
	} {
		if strings.Contains(out, r) {
			return true
		}
	}
	return false
}
