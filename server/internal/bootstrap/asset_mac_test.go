package bootstrap

import (
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"net/url"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/local"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
)

func TestAssetMACPurposeAndRestart(t *testing.T) {
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	load := func() *identity.Keys {
		t.Helper()
		keys, err := identity.LoadKeys(pemBytes, slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return keys
	}
	upload, read, err := assetMACs(load())
	if err != nil {
		t.Fatal(err)
	}
	message := []byte("asset-purpose-contract")
	for _, tc := range []struct {
		purpose string
		mac     assetapp.MAC
	}{{"asset-upload", upload}, {"asset-read", read}} {
		key, err := hkdf.Key(sha256.New, private.Seed(), nil, "nerve "+tc.purpose+" mac v1", 32)
		if err != nil {
			t.Fatal(err)
		}
		h := hmac.New(sha256.New, key)
		_, _ = h.Write(message)
		want := [16]byte(h.Sum(nil)[:16])
		if got := tc.mac.Tag(message); got != want || !tc.mac.Verify(message, want) {
			t.Fatalf("%s does not use its specified derivation", tc.purpose)
		}
	}
	if read.Verify(message, upload.Tag(message)) || upload.Verify(message, read.Tag(message)) {
		t.Fatal("asset purposes share authentication")
	}
	now := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	newStore := func(keys *identity.Keys) *local.Store {
		t.Helper()
		up, rd, err := assetMACs(keys)
		if err != nil {
			t.Fatal(err)
		}
		store, err := local.New(local.Config{Root: t.TempDir(), SizeLimit: 5242880}, local.Deps{UploadMAC: up, ReadMAC: rd, RuntimeNow: func() time.Time { return now }})
		if err != nil {
			t.Fatal(err)
		}
		return store
	}
	key, err := domain.NewObjectKey('u', uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	signed, err := newStore(load()).SignUpload(t.Context(), domain.UploadGrant{Key: key, Size: 3, ContentType: "image/png", ExpiresAt: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(signed.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newStore(load()).VerifyUpload(string(key), u.RawQuery, "PUT"); err != nil {
		t.Fatalf("same PEM after restart: %v", err)
	}
	other, err := identity.LoadKeys(nil, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := newStore(other).VerifyUpload(string(key), u.RawQuery, "PUT"); err == nil {
		t.Fatal("new key verifies an old grant")
	}
}
