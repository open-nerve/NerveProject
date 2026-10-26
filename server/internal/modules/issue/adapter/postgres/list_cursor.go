package postgresadapter

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// sortOrderNumericKey maps finite IEEE binary64 to a signed, exact ordering
// integer. Both zeros map to zero; negative magnitudes reverse their bits.
func sortOrderNumericKey(value float64) (int64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("non-finite list sort order")
	}
	bits := math.Float64bits(value)
	key := int64(bits & 0x7fffffffffffffff)
	if bits>>63 != 0 {
		key = -key
	}
	return key, nil
}

func cursorParameters(p *gen.FlatIssuesParams, c domain.ListCursor, order domain.ListOrder) error {
	if c.Order != order {
		return fmt.Errorf("list cursor ordering mismatch")
	}
	// Reuse domain validation instead of trusting programmatically supplied keys.
	encoded, err := domain.EncodeListCursor(c)
	if err != nil {
		return err
	}
	c, err = domain.DecodeListCursor(encoded, order)
	if err != nil {
		return err
	}
	p.HasCursor, p.CursorNull, p.CursorID = true, string(c.Key) == "null", &c.ID
	if p.CursorNull {
		return nil
	}
	var numeric int64
	switch p.OrderKey {
	case "state_name", "label_name", "assignee_name":
		var value string
		if err := json.Unmarshal(c.Key, &value); err != nil {
			return err
		}
		p.CursorText = &value
		return nil
	case "created_at", "updated_at", "start_date", "target_date":
		var value string
		if err := json.Unmarshal(c.Key, &value); err != nil {
			return err
		}
		layout := time.RFC3339Nano
		if strings.HasSuffix(p.OrderKey, "_date") {
			layout = time.DateOnly
		}
		at, err := time.Parse(layout, value)
		if err != nil {
			return err
		}
		numeric = at.UnixMicro()
		if layout == time.DateOnly {
			numeric = at.Unix() / 86400
		}
	case "sort_order":
		var value float64
		if err := json.Unmarshal(c.Key, &value); err != nil {
			return err
		}
		numeric, err = sortOrderNumericKey(value)
		if err != nil {
			return err
		}
	default:
		if err := json.Unmarshal(c.Key, &numeric); err != nil {
			return err
		}
	}
	p.CursorNum = &numeric
	return nil
}

func listPosition(r gen.FlatIssuesRow, order domain.ListOrder) (domain.ListCursor, error) {
	var key any
	switch strings.TrimPrefix(string(order), "-") {
	case "sort_order":
		key = r.SortOrder
	case "created_at":
		key = r.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000Z")
	case "updated_at":
		key = r.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000000Z")
	case "start_date":
		if r.StartDate != nil {
			key = r.StartDate.Format(time.DateOnly)
		}
	case "target_date":
		if r.TargetDate != nil {
			key = r.TargetDate.Format(time.DateOnly)
		}
	case "sub_issues_count":
		key = r.SubIssuesCount
	case "priority":
		switch r.Priority {
		case "urgent":
			key = int64(4)
		case "high":
			key = int64(3)
		case "medium":
			key = int64(2)
		case "low":
			key = int64(1)
		default:
			key = int64(0)
		}
	default:
		if !r.NameNull {
			key = r.NameKey
		}
	}
	raw, err := json.Marshal(key)
	if err != nil {
		return domain.ListCursor{}, err
	}
	position := domain.ListCursor{Order: order, Key: raw, ID: r.ID}
	// Validate and normalize the public key (including negative zero).
	encoded, err := domain.EncodeListCursor(position)
	if err != nil {
		return domain.ListCursor{}, err
	}
	return domain.DecodeListCursor(encoded, order)
}
