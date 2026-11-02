package httpadapter

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

const uploadResponseGrace = 5 * time.Second

func (t *Transfer) claim(w http.ResponseWriter) (func(), bool) {
	select {
	case t.slots <- struct{}{}:
		return func() { <-t.slots }, true
	default:
		w.Header().Set("Retry-After", "1")
		transferProblem(w, http.StatusServiceUnavailable, "server_busy")
		return nil, false
	}
}

// transferBudget uses the real monotonic runtime clock: socket deadlines cannot
// follow the application business clock or a clock used to test signed expiry.
type transferBudget struct {
	ctx                 context.Context
	cancel              context.CancelFunc
	controller          *http.ResponseController
	end                 time.Time
	idle                time.Duration
	upload              bool
	mu                  sync.Mutex // A progress refresh must not undo a concurrent cancellation.
	lastRead, lastWrite time.Time
	stop                func() bool
	interrupted         chan struct{}
}

func (t *Transfer) begin(w http.ResponseWriter, r *http.Request, size int64, upload bool) (*transferBudget, error) {
	duration, err := t.deps.Files.TransferBudget(size)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	end := now.Add(duration)
	ctx, cancel := context.WithDeadline(r.Context(), end)
	b := &transferBudget{ctx: ctx, cancel: cancel, controller: http.NewResponseController(w), end: end, idle: t.deps.Files.TransferIdleTimeout, upload: upload, lastRead: now, lastWrite: now, interrupted: make(chan struct{})}
	if err := b.refresh(true); err != nil {
		cancel()
		return nil, err
	}
	if err := b.refresh(false); err != nil {
		cancel()
		return nil, err
	}
	b.stop = context.AfterFunc(ctx, func() {
		defer close(b.interrupted)
		b.mu.Lock()
		defer b.mu.Unlock()
		now := time.Now()
		_ = b.controller.SetReadDeadline(now)
		writeEnd := now
		if b.upload {
			writeEnd = earlier(b.end.Add(uploadResponseGrace), now.Add(uploadResponseGrace))
		}
		_ = b.controller.SetWriteDeadline(writeEnd)
	})
	return b, nil
}

func earlier(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func (b *transferBudget) refresh(read bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.ctx.Err(); err != nil {
		return err
	}
	if read {
		if !b.upload {
			// net/http reads in the background to detect a disconnected GET
			// client. No new request bytes are expected during a healthy download.
			return b.controller.SetReadDeadline(b.end)
		}
		return b.controller.SetReadDeadline(earlier(b.end, b.lastRead.Add(b.idle)))
	}
	if b.upload {
		return b.controller.SetWriteDeadline(b.end.Add(uploadResponseGrace))
	}
	return b.controller.SetWriteDeadline(earlier(b.end, b.lastWrite.Add(b.idle)))
}

func (b *transferBudget) progress(read bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if read {
		b.lastRead = time.Now()
	} else {
		b.lastWrite = time.Now()
	}
}

func (b *transferBudget) close() {
	// On success, stop the hook before canceling: a late callback must not
	// expire the connection while net/http flushes the final response headers.
	if !b.stop() {
		<-b.interrupted
	}
	b.cancel()
}

type budgetReader struct {
	reader io.Reader
	budget *transferBudget
}

func (r budgetReader) Read(p []byte) (int, error) {
	r.budget.mu.Lock()
	expectedEnd := earlier(r.budget.end, r.budget.lastRead.Add(r.budget.idle))
	r.budget.mu.Unlock()
	if err := r.budget.refresh(true); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	if n > 0 {
		r.budget.progress(true)
	}
	if err != nil && err != io.EOF {
		// net/http cancels its context when a socket read times out. Preserve a
		// deadline that actually elapsed; only an earlier cancellation caused
		// by our interrupt hook should replace that socket error.
		var timeout net.Error
		if canceled := r.budget.ctx.Err(); canceled != nil && errors.As(err, &timeout) && timeout.Timeout() && time.Now().Before(expectedEnd) {
			return n, canceled
		}
		return n, err
	}
	if canceled := r.budget.ctx.Err(); canceled != nil {
		return n, canceled
	}
	return n, err
}

type budgetWriter struct {
	http.ResponseWriter
	budget *transferBudget
}

func (w *budgetWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *budgetWriter) WriteHeader(status int) {
	if err := w.budget.refresh(false); err != nil {
		panic(http.ErrAbortHandler)
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *budgetWriter) Write(p []byte) (int, error) {
	if err := w.budget.refresh(false); err != nil {
		return 0, err
	}
	n, err := w.ResponseWriter.Write(p)
	if n > 0 {
		w.budget.progress(false)
	}
	return n, err
}
