package bootstrap

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"uuid"
)

func TestPurgeCursorOutputCodec(t *testing.T) {
	valid := purgeResult{Cursor: purgeCursor{Version: 1, Module: "issue", Table: "issues", After: uuid.New(), Cutoff: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}, Scanned: 2, Deleted: 1, Blocked: 1}
	raw, err := encodePurgeOutput(valid)
	if err != nil {
		t.Fatal(err)
	}
	got, found, err := decodePurgeOutput(append(append([]byte(`{"other_river_key":true,"output":`), raw...), '}'))
	if err != nil || !found || got != valid {
		t.Fatalf("roundtrip=%+v %t %v", got, found, err)
	}
	for _, metadata := range [][]byte{nil, []byte(`{}`)} {
		_, found, err := decodePurgeOutput(metadata)
		if err != nil || found {
			t.Fatalf("initial=%v %t %v", metadata, found, err)
		}
	}
	bad := valid
	bad.Cursor.Version = 2
	version, _ := json.Marshal(bad)
	bad = valid
	bad.Scanned = 0
	count, _ := json.Marshal(bad)
	bad = valid
	bad.Cursor.Cutoff = time.Time{}
	cutoff, _ := json.Marshal(bad)
	for _, raw := range []string{"null", "[]", "{", string(version), string(count), string(cutoff), strings.TrimSuffix(string(raw), "}") + `,"unknown":1}`, `"` + strings.Repeat("x", maxPurgeOutput) + `"`} {
		if _, _, err := decodePurgeOutput([]byte(`{"output":` + raw + `}`)); err == nil {
			t.Fatalf("accepted malformed output, bytes=%d", len(raw))
		}
	}
	valid.Cursor.Table = strings.Repeat("x", maxPurgeOutput)
	if _, err := encodePurgeOutput(valid); err == nil {
		t.Fatal("accepted oversized output")
	}
}
