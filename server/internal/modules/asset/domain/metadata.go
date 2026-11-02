package domain

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"
	"uuid"
)

const (
	MaxMetadataBytes     = 16 << 10
	MaxCandidates        = 3
	CandidateLease       = 30 * time.Second
	CandidateSettleGrace = 30 * time.Second
)

type Origin string

const (
	OriginUpload Origin = "upload"
	OriginCopy   Origin = "copy"
)

type ObjectPhase string

const (
	ObjectActive     ObjectPhase = "active"
	ObjectReclaiming ObjectPhase = "reclaiming"
)

// ObjectRecord is retained until an external deletion is confirmed.
type ObjectRecord struct {
	Key         ObjectKey   `json:"key"`
	Phase       ObjectPhase `json:"phase"`
	SafeAfter   time.Time   `json:"safe_after"`
	Size        int64       `json:"size"`
	ContentType string      `json:"content_type"`
	ETag        string      `json:"etag"`
	SHA256      string      `json:"sha256"`
}
type Candidate struct {
	ObjectRecord
	LeaseUntil time.Time `json:"lease_until"`
}
type StorageMetadata struct {
	SchemaVersion int           `json:"schema_version"`
	Origin        Origin        `json:"origin"`
	Staging       *ObjectRecord `json:"staging"`
	Final         *ObjectRecord `json:"final"`
	Candidates    []Candidate   `json:"candidates"`
}

func (m StorageMetadata) Validate(id uuid.UUID) error {
	if id == uuid.Nil() || m.SchemaVersion != 1 || (m.Origin != OriginUpload && m.Origin != OriginCopy) ||
		m.Candidates == nil || len(m.Candidates) > MaxCandidates || (m.Origin == OriginCopy && m.Staging != nil) {
		return ErrInvalidMetadata
	}
	seen := make(map[ObjectKey]bool, MaxCandidates+2)
	check := func(record *ObjectRecord, kind byte, proved bool) bool {
		if record == nil {
			return true
		}
		if !record.valid(id, kind, proved) || seen[record.Key] {
			return false
		}
		seen[record.Key] = true
		return true
	}
	if !check(m.Staging, 'u', false) || !check(m.Final, 'o', true) {
		return ErrInvalidMetadata
	}
	for _, c := range m.Candidates {
		if c.LeaseUntil.IsZero() || c.SafeAfter.Before(c.LeaseUntil.Add(CandidateSettleGrace)) || !check(&c.ObjectRecord, 'o', false) {
			return ErrInvalidMetadata
		}
	}
	encoded, err := json.Marshal(m)
	if err != nil || len(encoded) > MaxMetadataBytes {
		return ErrInvalidMetadata
	}
	return nil
}

func (r ObjectRecord) valid(id uuid.UUID, kind byte, proved bool) bool {
	if r.Key.Kind() != kind || r.Key.AssetID() != id || (r.Phase != ObjectActive && r.Phase != ObjectReclaiming) ||
		r.SafeAfter.IsZero() || r.Size < 1 || r.Size > MaxObjectSize || !ValidMediaType(r.ContentType) || len(r.ETag) > 1024 {
		return false
	}
	if r.SHA256 != "" && (len(r.SHA256) != 64 || strings.TrimLeft(r.SHA256, "0123456789abcdef") != "") {
		return false
	}
	return !proved || (r.ETag != "" && r.SHA256 != "")
}

// DecodeStorageMetadata refuses unknown shapes rather than silently dropping recovery keys.
func DecodeStorageMetadata(id uuid.UUID, raw []byte) (StorageMetadata, error) {
	var out StorageMetadata
	if len(raw) > MaxMetadataBytes {
		return out, ErrInvalidMetadata
	}
	var shape map[string]json.RawMessage
	if json.Unmarshal(raw, &shape) != nil || len(shape) != 5 {
		return out, ErrInvalidMetadata
	}
	for _, key := range []string{"schema_version", "origin", "staging", "final", "candidates"} {
		if _, ok := shape[key]; !ok {
			return out, ErrInvalidMetadata
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&out) != nil {
		return StorageMetadata{}, ErrInvalidMetadata
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF || out.Validate(id) != nil {
		return StorageMetadata{}, ErrInvalidMetadata
	}
	return out, nil
}
