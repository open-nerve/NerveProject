package riveradapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
)

const maxSweepOutput = 16 << 10

func decodeOutput(metadata []byte, stage string) (app.SweepResult, bool, error) {
	var result app.SweepResult
	if len(metadata) == 0 {
		return result, false, nil
	}
	var envelope struct {
		Output json.RawMessage `json:"output"`
	}
	if err := json.Unmarshal(metadata, &envelope); err != nil {
		return result, false, err
	}
	if bytes.Equal(bytes.TrimSpace(metadata), []byte("null")) {
		return result, false, errors.New("asset metadata is null")
	}
	if len(envelope.Output) == 0 {
		return result, false, nil
	}
	if len(envelope.Output) > maxSweepOutput {
		return result, false, errors.New("asset sweep output exceeds 16 KiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(envelope.Output))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return result, false, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return result, false, errors.New("asset output has trailing data")
	}
	if err := validOutput(result, stage); err != nil {
		return result, false, err
	}
	return result, true, nil
}
func validOutput(r app.SweepResult, stage string) error {
	c := r.Cursor
	if c.Version != 1 || c.Stage != stage || c.Cutoff.IsZero() || c.Token != "" || r.Scanned < 0 || r.Removed < 0 || (stage == "rows" && c.RuntimeCutoff.IsZero()) || (stage == "scope" && !c.RuntimeCutoff.IsZero()) {
		return errors.New("invalid asset sweep output")
	}
	return nil
}
func finishSweep(ctx context.Context, result app.SweepResult, workErr error, stage string) error {
	if err := validOutput(result, stage); err != nil {
		return errors.Join(workErr, err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return errors.Join(workErr, err)
	}
	if len(raw) > maxSweepOutput {
		return errors.Join(workErr, errors.New("asset sweep output exceeds 16 KiB"))
	}
	if err := river.RecordOutput(ctx, json.RawMessage(raw)); err != nil {
		return errors.Join(workErr, err)
	}
	if workErr != nil {
		return workErr
	}
	if !result.Complete {
		return river.JobSnooze(0)
	}
	return nil
}
