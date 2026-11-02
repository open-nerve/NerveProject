package local

import (
	"context"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func (s *Store) SignUpload(ctx context.Context, g domain.UploadGrant) (domain.SignedRequest, error) {
	if err := ctx.Err(); err != nil {
		return domain.SignedRequest{}, err
	}
	now := s.deps.RuntimeNow()
	if err := domain.ValidateUpload(g, s.cfg.SizeLimit, now); err != nil {
		return domain.SignedRequest{}, err
	}
	// Validate the requested lifetime before rounding; rounding must never admit
	// an overlong grant or produce a token that has already expired.
	g.ExpiresAt = g.ExpiresAt.UTC().Truncate(time.Second)
	if !g.ExpiresAt.After(now) {
		return domain.SignedRequest{}, domain.ErrInvalidUpload
	}
	url, err := signToken(payload{Version: 1, Purpose: uploadPurpose, Method: "PUT", Key: g.Key, Size: g.Size, ContentType: g.ContentType, ExpiresAt: g.ExpiresAt.Unix()}, s.deps.UploadMAC)
	if err != nil {
		return domain.SignedRequest{}, domain.ErrInvalidUpload
	}
	return domain.SignedRequest{Method: "PUT", URL: url, Headers: map[string]string{"Content-Type": g.ContentType}, ExpiresAt: g.ExpiresAt}, nil
}

func (s *Store) SignRead(ctx context.Context, g domain.ReadGrant) (domain.SignedRequest, error) {
	if err := ctx.Err(); err != nil {
		return domain.SignedRequest{}, err
	}
	now := s.deps.RuntimeNow()
	if err := domain.ValidateRead(g, now); err != nil {
		return domain.SignedRequest{}, err
	}
	g.ExpiresAt = g.ExpiresAt.UTC().Truncate(time.Second)
	if !g.ExpiresAt.After(now) {
		return domain.SignedRequest{}, domain.ErrInvalidRead
	}
	url, err := signToken(payload{Version: 1, Purpose: readPurpose, Method: g.Method, Key: g.Key, ContentType: g.ContentType, Name: g.Name, Inline: g.Inline, ExpiresAt: g.ExpiresAt.Unix()}, s.deps.ReadMAC)
	if err != nil {
		return domain.SignedRequest{}, domain.ErrInvalidRead
	}
	return domain.SignedRequest{Method: g.Method, URL: url, ExpiresAt: g.ExpiresAt}, nil
}

func (s *Store) VerifyUpload(key, rawQuery, method string) (domain.UploadGrant, error) {
	p, err := verifyToken(key, rawQuery, method, uploadPurpose, s.deps.UploadMAC)
	if err != nil || p.Method != "PUT" || p.Name != "" || p.Inline {
		return domain.UploadGrant{}, domain.ErrInvalidSignature
	}
	g := domain.UploadGrant{Key: p.Key, Size: p.Size, ContentType: p.ContentType, ExpiresAt: time.Unix(p.ExpiresAt, 0).UTC()}
	// Authentication validates the format's hard bound. The HTTP consumer and
	// Put enforce today's upload limit separately, including grants issued earlier.
	if domain.ValidateUpload(g, domain.MaxObjectSize, s.deps.RuntimeNow()) != nil {
		return domain.UploadGrant{}, domain.ErrInvalidSignature
	}
	return g, nil
}

func (s *Store) VerifyRead(key, rawQuery, method string) (domain.ReadGrant, error) {
	p, err := verifyToken(key, rawQuery, method, readPurpose, s.deps.ReadMAC)
	if err != nil || p.Size != 0 {
		return domain.ReadGrant{}, domain.ErrInvalidSignature
	}
	g := domain.ReadGrant{Key: p.Key, Method: p.Method, Name: p.Name, ContentType: p.ContentType, Inline: p.Inline, ExpiresAt: time.Unix(p.ExpiresAt, 0).UTC()}
	if domain.ValidateRead(g, s.deps.RuntimeNow()) != nil {
		return domain.ReadGrant{}, domain.ErrInvalidSignature
	}
	return g, nil
}
