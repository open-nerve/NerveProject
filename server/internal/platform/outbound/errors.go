package outbound

import (
	"context"
	"errors"
	"fmt"
	"net"
)

// ErrBlocked identifies a URL or address refused by the outbound policy.
var ErrBlocked = errors.New("outbound address blocked")

// ErrTimeout identifies expiration or cancellation of an outbound request.
var ErrTimeout = errors.New("outbound request timed out")

// ErrUnreachable identifies other failures to complete an outbound request.
var ErrUnreachable = errors.New("outbound request unreachable")

// ErrTooLarge identifies a decoded response exceeding its byte limit.
var ErrTooLarge = errors.New("outbound response too large")

type redirectLimitError struct{}

func (*redirectLimitError) Error() string { return "outbound redirect limit exceeded" }
func (*redirectLimitError) Unwrap() error { return ErrUnreachable }

// Reason returns only a controlled diagnostic, never text from the error chain.
func Reason(err error) string {
	var limit *redirectLimitError
	if errors.As(err, &limit) {
		return "redirect_limit"
	}
	return ""
}

func classify(err error) error {
	var network net.Error
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || (errors.As(err, &network) && network.Timeout()) {
		return fmt.Errorf("%w: %w", ErrTimeout, err)
	}
	return fmt.Errorf("%w: %w", ErrUnreachable, err)
}
