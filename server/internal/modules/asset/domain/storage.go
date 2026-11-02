// Package domain holds the asset module's values and pure rules.
package domain

import (
	"encoding/hex"
	"errors"
	"mime"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"uuid"
)

// MaxObjectSize bounds a single non-multipart object, independently of configuration.
const MaxObjectSize int64 = 100 << 20

// Transfer grants use real time, not the application's shifted business clock.
const (
	MaxUploadTTL = 15 * time.Minute
	MaxReadTTL   = 65 * time.Minute
)

var (
	ErrInvalidObjectKey = errors.New("invalid asset object key")
	ErrInvalidUpload    = errors.New("invalid asset upload grant")
	ErrInvalidRead      = errors.New("invalid asset read grant")
	ErrInvalidSignature = errors.New("invalid asset transfer signature")
	ErrObjectNotFound   = errors.New("asset object not found")
	ErrObjectChanged    = errors.New("asset object changed")
	ErrObjectExists     = errors.New("asset object already exists")
)

// ObjectKey identifies one staging (u) or immutable sealed (o) object.
// Its two UUIDs are explicit inputs; constructing a key performs no I/O.
type ObjectKey string

func NewObjectKey(kind byte, assetID, objectID uuid.UUID) (ObjectKey, error) {
	if (kind != 'u' && kind != 'o') || assetID == uuid.Nil() || objectID == uuid.Nil() {
		return "", ErrInvalidObjectKey
	}
	return ObjectKey(string(kind) + "-" + hex.EncodeToString(assetID[:]) + "-" + hex.EncodeToString(objectID[:])), nil
}

func ParseObjectKey(raw string) (ObjectKey, error) {
	if _, _, _, ok := keyParts(raw); !ok {
		return "", ErrInvalidObjectKey
	}
	return ObjectKey(raw), nil
}

// Invalid values expose no usable kind or identity, even when constructed by conversion.
func (k ObjectKey) Kind() byte {
	kind, _, _, _ := keyParts(string(k))
	return kind
}

func (k ObjectKey) AssetID() uuid.UUID {
	_, asset, _, _ := keyParts(string(k))
	return asset
}

func (k ObjectKey) ObjectID() uuid.UUID {
	_, _, object, _ := keyParts(string(k))
	return object
}

func keyParts(raw string) (byte, uuid.UUID, uuid.UUID, bool) {
	invalid := func() (byte, uuid.UUID, uuid.UUID, bool) { return 0, uuid.Nil(), uuid.Nil(), false }
	if len(raw) != 67 || (raw[0] != 'u' && raw[0] != 'o') || raw[1] != '-' || raw[34] != '-' {
		return invalid()
	}
	var asset, object uuid.UUID
	for _, part := range []struct {
		raw  string
		into []byte
	}{{raw[2:34], asset[:]}, {raw[35:], object[:]}} {
		if strings.TrimLeft(part.raw, "0123456789abcdef") != "" {
			return invalid()
		}
		if _, err := hex.Decode(part.into, []byte(part.raw)); err != nil {
			return invalid()
		}
	}
	if asset == uuid.Nil() || object == uuid.Nil() {
		return invalid()
	}
	return raw[0], asset, object, true
}

type UploadGrant struct {
	Key         ObjectKey
	Size        int64
	ContentType string
	ExpiresAt   time.Time
}

type ReadGrant struct {
	Key               ObjectKey
	Method            string
	Name, ContentType string
	Inline            bool
	ExpiresAt         time.Time
}

type SignedRequest struct {
	Method, URL string
	Headers     map[string]string
	ExpiresAt   time.Time
}

type ObjectInfo struct {
	Key               ObjectKey
	Size              int64
	ContentType, ETag string
	ModifiedAt        time.Time
}

func ValidateUpload(g UploadGrant, sizeLimit int64, now time.Time) error {
	if g.Key.Kind() != 'u' || sizeLimit < 1 || sizeLimit > MaxObjectSize || g.Size < 1 || g.Size > sizeLimit ||
		!ValidMediaType(g.ContentType) || !validExpiry(g.ExpiresAt, now, MaxUploadTTL) {
		return ErrInvalidUpload
	}
	return nil
}

func ValidateRead(g ReadGrant, now time.Time) error {
	if g.Key.Kind() != 'o' || (g.Method != "GET" && g.Method != "HEAD") ||
		!validName(g.Name) || !ValidMediaType(g.ContentType) || !validExpiry(g.ExpiresAt, now, MaxReadTTL) {
		return ErrInvalidRead
	}
	if g.Inline {
		switch g.ContentType {
		case "image/jpeg", "image/png", "image/webp", "image/gif":
		default:
			return ErrInvalidRead
		}
	}
	return nil
}

func validExpiry(expiresAt, now time.Time, limit time.Duration) bool {
	return expiresAt.After(now) && !expiresAt.After(now.Add(limit))
}

// ValidMediaType accepts a canonical type without parameters, for grants and stored metadata.
func ValidMediaType(value string) bool {
	media, params, err := mime.ParseMediaType(value)
	return err == nil && media == value && len(params) == 0 && strings.Contains(media, "/") && !strings.Contains(media, "*")
}

func validName(name string) bool {
	if name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > 255 || strings.ContainsAny(name, "/\\") {
		return false
	}
	return !strings.ContainsFunc(name, unicode.IsControl)
}
