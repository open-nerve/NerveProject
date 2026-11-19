package domain

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

const listTimestampLayout = "2006-01-02T15:04:05.000000Z"

func canonicalListKey(order ListOrder, raw json.RawMessage) (json.RawMessage, error) {
	if !ListOrderAllowed(order) {
		return nil, shared.InvalidCursor()
	}
	field := strings.TrimPrefix(string(order), "-")
	nullable := field == "start_date" || field == "target_date" || field == "state_name" || field == "label_name" || field == "assignee_name" || field == "cycle_name" || field == "module_name"
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		if nullable {
			return json.RawMessage("null"), nil
		}
		return nil, shared.InvalidCursor()
	}
	switch field {
	case "created_at", "updated_at":
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, shared.InvalidCursor()
		}
		at, ok := listTimestamp(value)
		if !ok {
			return nil, shared.InvalidCursor()
		}
		return json.Marshal(at.Format(listTimestampLayout))
	case "start_date", "target_date":
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, shared.InvalidCursor()
		}
		day, err := shared.ParseDate(value)
		if err != nil {
			return nil, shared.InvalidCursor()
		}
		return json.Marshal(day.String())
	case "state_name", "label_name", "assignee_name", "cycle_name", "module_name":
		var value string
		if err := json.Unmarshal(raw, &value); err != nil || strings.ContainsRune(value, '\x00') {
			return nil, shared.InvalidCursor()
		}
		return json.Marshal(NameOrderKey(value))
	case "priority", "sub_issues_count", "link_count", "attachment_count":
		var value int64
		if err := json.Unmarshal(raw, &value); err != nil || value < 0 || (field == "priority" && value > 4) {
			return nil, shared.InvalidCursor()
		}
		return json.Marshal(value)
	case "sort_order":
		var value float64
		if err := json.Unmarshal(raw, &value); err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, shared.InvalidCursor()
		}
		if value == 0 {
			value = 0
		} // Equal zero positions have one public spelling.
		return json.Marshal(value)
	}
	return nil, shared.InvalidCursor()
}

// listTimestamp reads a cursor's timestamp key, RFC 3339 of a year 1–9999
// where it is written and in UTC, at most microseconds, and answers it in
// UTC. The list's and the intake list's cursors read their time keys so.
func listTimestamp(value string) (time.Time, bool) {
	at, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || at.Year() < 1 || at.Year() > 9999 || at.UTC().Year() < 1 || at.UTC().Year() > 9999 || !listMicrosecondAligned(value) {
		return time.Time{}, false
	}
	return at.UTC(), true
}

// time.Parse truncates fractions beyond nanoseconds. Inspect the original
// fraction as well so even a nonzero tenth decimal cannot silently disappear.
func listMicrosecondAligned(value string) bool {
	start := strings.IndexAny(value, ".,")
	if start < 0 {
		return true
	}
	for i := start + 1; i < len(value) && value[i] >= '0' && value[i] <= '9'; i++ {
		if i-start > 6 && value[i] != '0' {
			return false
		}
	}
	return true
}
