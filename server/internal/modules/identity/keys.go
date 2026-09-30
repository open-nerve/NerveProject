package identity

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/signing"
)

// Keys are the instance's signing key, loaded once, before any module is
// built (M3 design 6.6 step 1). They never give the key out: identity.New
// signs the access tokens and tags the refresh tokens with it, and MAC
// gives another module a MAC of its own purpose.
type Keys struct {
	signing *signing.Keys
}

// LoadKeys loads the signing key from pemData, the content of
// auth.jwt.private_key_file. nil is no file: the key is then ephemeral, for
// dev and test only (M2 design 3.7), and logger warns. A key that cannot be
// parsed is an error that never quotes the key.
func LoadKeys(pemData []byte, logger *slog.Logger) (*Keys, error) {
	if pemData == nil {
		logger.Warn("auth.jwt.private_key_file is not set: signing with an ephemeral key; " +
			"access tokens and invitation links stop verifying at restart (dev and test only)")
		return &Keys{signing: signing.EphemeralKeys()}, nil
	}
	keys, err := signing.ParseKeys(pemData)
	if err != nil {
		return nil, fmt.Errorf("auth.jwt.private_key_file: %w", err)
	}
	return &Keys{signing: keys}, nil
}

// MAC authenticates the messages of one purpose: Tag gives the first 16
// bytes of HMAC-SHA256 under the purpose's key, and Verify compares a tag
// in constant time.
type MAC interface {
	Tag(message []byte) [16]byte
	Verify(message []byte, tag [16]byte) bool
}

// MAC returns the MAC of purpose, whose key is derived from the signing key
// for that purpose alone, with HKDF info "nerve <purpose> mac v1" (M3
// design 3.8, 11.1): a tag of one purpose verifies for no other. purpose
// names another module's use, e.g. "workspace-invitation"; the refresh
// tokens' purpose is identity's own and no other module's.
func (k *Keys) MAC(purpose string) (MAC, error) {
	if purpose == signing.PurposeRefreshToken {
		return nil, errors.New("the refresh-token MAC is identity's own")
	}
	m, err := k.signing.MAC(purpose)
	if err != nil {
		return nil, err
	}
	return m, nil
}
