package local

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/localprotocol"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

const (
	maxQueryBytes = 4096
	uploadPurpose = "upload"
	readPurpose   = "read"
)

// This field order and the zero fields are the version-one wire format.
// Re-encoding after decoding rejects duplicates, unknown fields and aliases.
type payload struct {
	Version     int              `json:"v"`
	Purpose     string           `json:"purpose"`
	Method      string           `json:"method"`
	Key         domain.ObjectKey `json:"key"`
	Size        int64            `json:"size"`
	ContentType string           `json:"content_type"`
	Name        string           `json:"name"`
	Inline      bool             `json:"inline"`
	ExpiresAt   int64            `json:"expires_at"`
}

func signToken(p payload, mac app.MAC) (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", domain.ErrInvalidSignature
	}
	tag := mac.Tag(data)
	query := "payload=" + base64.RawURLEncoding.EncodeToString(data) + "&tag=" + base64.RawURLEncoding.EncodeToString(tag[:])
	if len(query) > maxQueryBytes {
		return "", domain.ErrInvalidSignature
	}
	return localprotocol.Path(p.Key) + "?" + query, nil
}

func verifyToken(key, rawQuery, method, purpose string, mac app.MAC) (payload, error) {
	var zero payload
	if len(rawQuery) > maxQueryBytes {
		return zero, domain.ErrInvalidSignature
	}
	parts := strings.Split(rawQuery, "&")
	if len(parts) != 2 {
		return zero, domain.ErrInvalidSignature
	}
	var encodedPayload, encodedTag string
	for _, part := range parts {
		name, value, ok := strings.Cut(part, "=")
		if !ok || value == "" {
			return zero, domain.ErrInvalidSignature
		}
		switch {
		case name == "payload" && encodedPayload == "":
			encodedPayload = value
		case name == "tag" && encodedTag == "":
			encodedTag = value
		default:
			return zero, domain.ErrInvalidSignature
		}
	}
	data, ok := decodeCanonical(encodedPayload)
	if !ok {
		return zero, domain.ErrInvalidSignature
	}
	tag, ok := decodeCanonical(encodedTag)
	if !ok || len(tag) != 16 || !mac.Verify(data, [16]byte(tag)) {
		return zero, domain.ErrInvalidSignature
	}
	var p payload
	if json.Unmarshal(data, &p) != nil {
		return zero, domain.ErrInvalidSignature
	}
	canonical, err := json.Marshal(p)
	if err != nil || !bytes.Equal(canonical, data) || p.Version != 1 || p.Purpose != purpose || p.Method != method || string(p.Key) != key {
		return zero, domain.ErrInvalidSignature
	}
	return p, nil
}

func decodeCanonical(value string) ([]byte, bool) {
	data, err := base64.RawURLEncoding.Strict().DecodeString(value)
	return data, value != "" && err == nil && base64.RawURLEncoding.EncodeToString(data) == value
}
