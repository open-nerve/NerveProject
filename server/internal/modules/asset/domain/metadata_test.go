package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"uuid"
)

func testMetadata(t *testing.T) (uuid.UUID, StorageMetadata) {
	t.Helper()
	id := uuid.MustParse("3d82e8b9-59bf-4f1b-82a3-bbc71dd52b9d")
	at := time.Date(2026, 11, 3, 1, 0, 0, 0, time.UTC)
	key, _ := NewObjectKey('u', id, uuid.MustParse("66c84c3d-dd62-4938-aa60-c0c507aa8b3a"))
	return id, StorageMetadata{SchemaVersion: 1, Origin: OriginUpload, Staging: &ObjectRecord{Key: key, Phase: ObjectActive, SafeAfter: at, Size: 3, ContentType: "image/png"}, Candidates: []Candidate{}}
}

// Four candidates, foreign keys, and ambiguous slots would lose or misdirect recovery.
func TestStorageMetadataBounds(t *testing.T) {
	id, good := testMetadata(t)
	if err := good.Validate(id); err != nil {
		t.Fatal(err)
	}
	candidate := func() Candidate {
		k, _ := NewObjectKey('o', id, uuid.NewV7())
		lease := good.Staging.SafeAfter
		return Candidate{ObjectRecord: ObjectRecord{Key: k, Phase: ObjectActive, SafeAfter: lease.Add(30 * time.Second), Size: 3, ContentType: "image/png"}, LeaseUntil: lease}
	}
	for range 3 {
		good.Candidates = append(good.Candidates, candidate())
	}
	if err := good.Validate(id); err != nil {
		t.Fatal("three candidates", err)
	}
	good.Candidates = append(good.Candidates, candidate())
	if good.Validate(id) == nil {
		t.Fatal("four candidates accepted")
	}
	for _, tc := range []struct {
		name   string
		change func(*StorageMetadata)
	}{
		{"version", func(m *StorageMetadata) { m.SchemaVersion = 2 }},
		{"origin", func(m *StorageMetadata) { m.Origin = "unknown" }},
		{"copy staging", func(m *StorageMetadata) { m.Origin = OriginCopy }},
		{"foreign key", func(m *StorageMetadata) { m.Staging.Key, _ = NewObjectKey('u', uuid.NewV7(), uuid.NewV7()) }},
		{"wrong key kind", func(m *StorageMetadata) { m.Staging.Key, _ = NewObjectKey('o', id, uuid.NewV7()) }},
		{"zero deadline", func(m *StorageMetadata) { m.Staging.SafeAfter = time.Time{} }},
		{"unknown phase", func(m *StorageMetadata) { m.Staging.Phase = "finished" }},
		{"negative size", func(m *StorageMetadata) { m.Staging.Size = -1 }},
		{"null list", func(m *StorageMetadata) { m.Candidates = nil }},
		{"duplicate", func(m *StorageMetadata) { c := candidate(); m.Candidates = []Candidate{c, c} }},
		{"early reclaim", func(m *StorageMetadata) { c := candidate(); c.SafeAfter = c.LeaseUntil; m.Candidates = []Candidate{c} }},
		{"invalid hash", func(m *StorageMetadata) { m.Staging.SHA256 = "abc" }},
		{"unbounded metadata", func(m *StorageMetadata) { m.Staging.ETag = strings.Repeat("x", 16385) }},
		{"final lacks proof", func(m *StorageMetadata) { c := candidate(); m.Final = &c.ObjectRecord }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, m := testMetadata(t)
			tc.change(&m)
			if m.Validate(id) == nil {
				t.Fatal("invalid metadata accepted")
			}
		})
	}
	c := candidate()
	c.ETag = "opaque-version"
	c.SHA256 = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	m := StorageMetadata{SchemaVersion: 1, Origin: OriginCopy, Final: &c.ObjectRecord, Candidates: []Candidate{}}
	if err := m.Validate(id); err != nil {
		t.Fatal("proved final", err)
	}
	m.Final.Phase = ObjectReclaiming
	if err := m.Validate(id); err != nil {
		t.Fatal("reclaim record must survive", err)
	}
}

func TestStorageMetadataDecoding(t *testing.T) {
	id, m := testMetadata(t)
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeStorageMetadata(id, raw)
	if err != nil || decoded.Staging == nil || decoded.Staging.Key != m.Staging.Key {
		t.Fatalf("roundtrip: %+v %v", decoded, err)
	}
	for _, bad := range []string{"null", "{}", "[]", string(raw) + " {}", strings.Replace(string(raw), `"schema_version":1`, `"schema_version":2`, 1), strings.Replace(string(raw), `"candidates":[]`, `"candidates":null`, 1), strings.Replace(string(raw), `"origin":"upload"`, `"origin":"upload","extra":true`, 1), strings.Replace(string(raw), `"phase":"active"`, `"phase":"active","surprise":true`, 1), strings.Repeat(" ", 16385) + string(raw)} {
		if _, err := DecodeStorageMetadata(id, []byte(bad)); err == nil {
			t.Fatalf("invalid JSON metadata accepted: %.160s", bad)
		}
	}
}
