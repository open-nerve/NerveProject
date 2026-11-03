// s3fixture is a bounded, one-shot helper for owned loopback test services.
// It is built separately and is never imported by the nerve binary.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

const envelopeLimit = 1 << 20

var errRequest = errors.New("invalid S3 fixture request")
var errOperation = errors.New("S3 fixture operation failed")

func main() {
	if err := run(context.Background(), os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(parent context.Context, input io.Reader, output io.Writer) error {
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	stopIO := context.AfterFunc(ctx, func() {
		for _, stream := range []any{input, output} {
			if closer, ok := stream.(io.Closer); ok {
				_ = closer.Close()
			}
		}
	})
	defer stopIO()
	data, err := io.ReadAll(io.LimitReader(input, envelopeLimit+1))
	if err != nil || len(data) > envelopeLimit {
		return errRequest
	}
	r, err := decodeRequest(data)
	if err != nil {
		return errRequest
	}
	out, err := execute(ctx, r)
	if err != nil {
		return errOperation
	}
	encoded, err := json.Marshal(out)
	if err != nil || len(encoded)+1 > envelopeLimit {
		return errOperation
	}
	_, err = output.Write(append(encoded, '\n'))
	if err != nil {
		return errOperation
	}
	return nil
}
