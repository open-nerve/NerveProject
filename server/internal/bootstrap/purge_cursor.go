package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const maxPurgeOutput = 16 * 1024

func decodePurgeOutput(metadata []byte) (purgeResult, bool, error) {
	var result purgeResult
	if len(metadata) == 0 {
		return result, false, nil
	}
	var envelope struct {
		Output json.RawMessage `json:"output"`
	}
	if err := json.Unmarshal(metadata, &envelope); err != nil {
		return result, false, fmt.Errorf("purge job metadata: %w", err)
	}
	if bytes.Equal(bytes.TrimSpace(metadata), []byte("null")) {
		return result, false, errors.New("purge job metadata is null")
	}
	if len(envelope.Output) == 0 {
		return result, false, nil
	}
	if len(envelope.Output) > maxPurgeOutput {
		return result, false, errors.New("purge output exceeds 16 KiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(envelope.Output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return result, false, fmt.Errorf("purge output: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return result, false, errors.New("purge output has trailing data")
	}
	if err := validPurgeOutput(result); err != nil {
		return result, false, err
	}
	return result, true, nil
}
func encodePurgeOutput(result purgeResult) (json.RawMessage, error) {
	if err := validPurgeOutput(result); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if len(raw) > maxPurgeOutput {
		return nil, errors.New("purge output exceeds 16 KiB")
	}
	return json.RawMessage(raw), nil
}
func validPurgeOutput(result purgeResult) error {
	c := result.Cursor
	if c.Version != 1 || c.Module == "" || c.Cutoff.IsZero() || result.Scanned < 0 || result.Deleted < 0 || result.Blocked < 0 || result.Deleted > result.Scanned || result.Blocked > result.Scanned-result.Deleted {
		return errors.New("invalid purge output version, cutoff or counts")
	}
	return nil
}
