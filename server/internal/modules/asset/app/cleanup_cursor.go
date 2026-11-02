package app

import (
	"encoding/json"
	"uuid"
)

// Limit the encoded opaque token before doing work; all remaining fields are
// bounded values, leaving ample room in River's 16 KiB output envelope.
func validSweepToken(token string) bool {
	if len(token) > 8<<10 {
		return false
	}
	raw, err := json.Marshal(token)
	return err == nil && len(raw) <= 8<<10
}

func (c SweepCursor) ValidCleanup() bool {
	if c.Version != 1 || c.Cutoff.IsZero() || c.RuntimeCutoff.IsZero() || !validSweepToken(c.Token) {
		return false
	}
	switch c.Stage {
	case "rows":
		return c.Token == "" && c.ObjectAfter == ""
	case "objects":
		return c.After == uuid.Nil() && (c.ObjectAfter == "" || c.ObjectAfter.Kind() != 0)
	case "tmp":
		return c.After == uuid.Nil() && c.ObjectAfter == ""
	case "done":
		return c.After == uuid.Nil() && c.Token == "" && c.ObjectAfter == ""
	default:
		return false
	}
}

func (c SweepCursor) ValidScope() bool {
	return c.Version == 1 && c.Stage == "scope" && !c.Cutoff.IsZero() && c.RuntimeCutoff.IsZero() && c.Token == "" && c.ObjectAfter == ""
}
