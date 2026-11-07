package s3test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
)

// startAttempts bounds the starts of one container whose host port is taken
// (hostPortTaken).
const startAttempts = 3

// startPublished calls start, which creates and starts a container that
// publishes a port on a host port Docker picks, until it succeeds or fails
// otherwise than by hostPortTaken, at most startAttempts times in all. It
// removes each container refused that way before the next start, and after
// the last answers no container and the last refusal. Any other error
// returns at once, with its container, which the caller removes as before.
func startPublished(start func() (testcontainers.Container, error)) (testcontainers.Container, error) {
	for attempt := 1; ; attempt++ {
		c, err := start()
		if err == nil || !hostPortTaken(err) {
			return c, err
		}
		if c != nil {
			// The start may have used up the caller's context: the removal
			// owns a fresh bounded one, as Close does.
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			removeErr := c.Terminate(ctx)
			cancel()
			if removeErr != nil {
				return nil, errors.Join(err, fmt.Errorf("remove container refused its host port: %w", removeErr))
			}
		}
		if attempt == startAttempts {
			return nil, fmt.Errorf("host port taken on all %d starts: %w", startAttempts, err)
		}
	}
}

// hostPortTaken reports whether err is Docker refusing to start a container
// because the host port it picked to publish a container port on is bound
// already. Docker picks that port from the ephemeral range without asking
// the host, where the many local connections of parallel test packages hold
// ports too. The daemon's cause reaches the client only as message text, so
// the text is matched: its port-publishing step, and the bind's errno.
func hostPortTaken(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	return strings.Contains(text, "programming external connectivity") && strings.Contains(text, "address already in use")
}
