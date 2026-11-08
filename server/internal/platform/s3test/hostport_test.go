package s3test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

// errTakenPort is the error New failed with in main CI 42399566014 (attempt 2):
// the host port Docker picked for 8333/tcp was already bound on the runner.
var errTakenPort = errors.New("start container: container start: Error response from daemon: failed to set up container networking: " +
	"driver failed programming external connectivity on endpoint nerve-s3-test-hjxtqy3yheiagvtblsnkngrpjy " +
	"(1195ba7d8970a60653a77d259064b5cbac4bf2e41a4e22aaf72bf70ee23ad4fe): failed to listen on TCP socket: address already in use")

func TestHostPortTaken(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"the daemon's refusal of a taken host port", errTakenPort, true},
		{"the same refusal, wrapped", fmt.Errorf("start: %w", errTakenPort), true},
		{"another refusal of the same step", errors.New("start container: container start: Error response from daemon: " +
			"failed to set up container networking: network 4f1c2e not found"), false},
		{"a host port Docker had already given out", errors.New("start container: container start: Error response from daemon: " +
			"driver failed programming external connectivity on endpoint nerve-s3-test-x (1195ba7d): Bind for 127.0.0.1:8333 failed: " +
			"port is already allocated"), false},
		{"a port in use elsewhere", errors.New("readied hook: listen tcp 127.0.0.1:8333: bind: address already in use"), false},
		{"no error", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := hostPortTaken(tc.err); got != tc.want {
				t.Fatalf("hostPortTaken(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}

// fakeContainer stands in for a created container; only its removal is used.
type fakeContainer struct {
	testcontainers.Container
	removed int
}

func (c *fakeContainer) Terminate(context.Context, ...testcontainers.TerminateOption) error {
	c.removed++
	return nil
}

// scripted is a start answering its calls with a new container each and the
// next of errs, then, past errs, with an error of its own, so that a loop
// that does not stop ends instead of hanging; made lists its containers.
func scripted(errs ...error) (start func() (testcontainers.Container, error), made *[]*fakeContainer) {
	made = &[]*fakeContainer{}
	return func() (testcontainers.Container, error) {
		c := &fakeContainer{}
		*made = append(*made, c)
		if len(*made) > len(errs) {
			return c, errors.New("started past the script")
		}
		return c, errs[len(*made)-1]
	}, made
}

func TestStartPublishedRetriesATakenHostPort(t *testing.T) {
	t.Run("taken twice, then started", func(t *testing.T) {
		start, made := scripted(errTakenPort, errTakenPort, nil)
		c, err := startPublished(start)
		if err != nil || len(*made) != 3 || c != (*made)[2] {
			t.Fatalf("got %v, %v after %d starts; want the third container, started", c, err, len(*made))
		}
		for i, m := range *made {
			want := 1
			if i == 2 {
				want = 0
			}
			if m.removed != want {
				t.Errorf("container %d removed %d times, want %d", i+1, m.removed, want)
			}
		}
	})
	t.Run("another error returns at once", func(t *testing.T) {
		other := errors.New("start container: container start: Error response from daemon: no such image")
		start, made := scripted(other)
		c, err := startPublished(start)
		// The caller removes this container, as for any failed start.
		if !errors.Is(err, other) || len(*made) != 1 || c != (*made)[0] || (*made)[0].removed != 0 {
			t.Fatalf("got %v, %v after %d starts; want the one container, not removed, and its error", c, err, len(*made))
		}
	})
	t.Run("taken on every start", func(t *testing.T) {
		start, made := scripted(errTakenPort, errTakenPort, errTakenPort)
		c, err := startPublished(start)
		if c != nil || !errors.Is(err, errTakenPort) || !strings.Contains(err.Error(), "3 starts") || len(*made) != 3 {
			t.Fatalf("got %v, %v after %d starts; want no container and the cause after 3 starts", c, err, len(*made))
		}
		for i, m := range *made {
			if m.removed != 1 {
				t.Errorf("container %d removed %d times, want 1", i+1, m.removed)
			}
		}
	})
}
