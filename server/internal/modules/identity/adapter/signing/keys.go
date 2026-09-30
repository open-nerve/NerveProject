// Package signing holds the instance's Ed25519 key (M2 design 3.7): it signs
// the access tokens and, through a key derived from it for each purpose,
// tags the refresh tokens (M2 design 3.4) and the workspace invitations
// (M3 design 3.8). The key never leaves this package and is never logged.
package signing

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

// Keys are the signing key and its public half. Each purpose's MAC key is
// derived from its seed (MAC).
type Keys struct {
	private ed25519.PrivateKey
	public  ed25519.PublicKey
}

// ParseKeys reads a PKCS#8 PEM Ed25519 private key, the format of
// `openssl genpkey -algorithm ed25519`. Errors never quote the input.
func ParseKeys(pemData []byte) (*Keys, error) {
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, errors.New("not a PEM \"PRIVATE KEY\" block (PKCS#8)")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS#8: %w", err)
	}
	private, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("the key is %T, want an Ed25519 key", key)
	}
	return newKeys(private), nil
}

// EphemeralKeys generates a key for this process only: dev and test without
// auth.jwt.private_key_file (M2 design 3.7).
func EphemeralKeys() *Keys {
	_, private, _ := ed25519.GenerateKey(nil) // crypto/rand; never fails
	return newKeys(private)
}

func newKeys(private ed25519.PrivateKey) *Keys {
	return &Keys{private: private, public: private.Public().(ed25519.PublicKey)}
}
