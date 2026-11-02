package httpadapter

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

func TestLocalTransferDeadlines(t *testing.T) {
	t.Run("progress extends both inherited socket deadlines", func(t *testing.T) {
		w := newBudgetWorld(t, budgetTestFiles(), 200*time.Millisecond, 200*time.Millisecond)
		g, signed := w.grant(t, 512<<10)
		conn := w.beginUpload(t, signed, g.Size)
		chunk := strings.Repeat("a", 64<<10)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		start := time.Now()
		for range 8 {
			<-ticker.C
			if _, err := io.WriteString(conn, chunk); err != nil {
				t.Fatal(err)
			}
		}
		response, body := wireResponse(t, conn)
		if response.StatusCode != 204 || len(body) != 0 || time.Since(start) < 600*time.Millisecond {
			t.Fatalf("progress response = %d %q after %s", response.StatusCode, body, time.Since(start))
		}
		if response.Header.Get(httpserver.HeaderRequestID) == "" || response.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("platform headers = %v", response.Header)
		}
		info, err := w.store.Inspect(t.Context(), g.Key)
		if err != nil || info.Size != g.Size {
			t.Fatalf("completed object = %+v, %v", info, err)
		}
		w.nextUploadWorks(t)
	})
	t.Run("idle read ends the socket and releases the slot", func(t *testing.T) {
		files := budgetTestFiles()
		files.TransferIdleTimeout = 200 * time.Millisecond
		w := newBudgetWorld(t, files, 5*time.Second, 5*time.Second)
		g, signed := w.grant(t, 1024)
		conn := w.beginUpload(t, signed, g.Size)
		if _, err := conn.Write([]byte("a")); err != nil {
			t.Fatal(err)
		}
		select {
		case <-w.finished:
		case <-time.After(time.Second):
			t.Fatal("idle upload retained its socket")
		}
		response, _ := wireResponse(t, conn)
		if response.StatusCode != 408 {
			t.Fatalf("idle status = %d", response.StatusCode)
		}
		if _, err := w.store.Inspect(t.Context(), g.Key); err != domain.ErrObjectNotFound {
			t.Fatalf("idle upload published: %v", err)
		}
		w.nextUploadWorks(t)
	})
	t.Run("progress cannot extend the total budget", func(t *testing.T) {
		files := budgetTestFiles()
		files.TransferGrace = 50 * time.Millisecond
		files.MaxTransferDuration = 1050 * time.Millisecond
		w := newBudgetWorld(t, files, 5*time.Second, 5*time.Second)
		g, signed := w.grant(t, 1024)
		conn := w.beginUpload(t, signed, g.Size)
		stop, writerDone := make(chan struct{}), make(chan struct{})
		go func() {
			defer close(writerDone)
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					if _, err := conn.Write([]byte("a")); err != nil {
						return
					}
				}
			}
		}()
		defer func() { close(stop); <-writerDone }()
		select {
		case <-w.finished:
		case <-time.After(2 * time.Second):
			t.Fatal("progress bypassed the total transfer budget")
		}
		response, _ := wireResponse(t, conn)
		if response.StatusCode != 408 {
			t.Fatalf("total status = %d", response.StatusCode)
		}
		w.nextUploadWorks(t)
	})
	t.Run("parent cancellation interrupts a blocked body", func(t *testing.T) {
		files := budgetTestFiles()
		files.TransferIdleTimeout = 2 * time.Second
		w := newBudgetWorld(t, files, 5*time.Second, 5*time.Second)
		g, signed := w.grant(t, 1024)
		conn := w.beginUpload(t, signed, g.Size)
		(<-w.cancellations)()
		select {
		case <-w.finished:
		case <-time.After(time.Second):
			t.Fatal("cancellation left the body read blocked")
		}
		response, _ := wireResponse(t, conn)
		if response.StatusCode != 499 {
			t.Fatalf("cancel status = %d", response.StatusCode)
		}
		w.nextUploadWorks(t)
	})
	t.Run("busy and all early exits release their slots", func(t *testing.T) {
		w := newBudgetWorld(t, budgetTestFiles(), 5*time.Second, 5*time.Second)
		g, first := w.grant(t, 3)
		conn := w.beginUpload(t, first, g.Size)
		_, second := w.grant(t, 3)
		if status := w.put(t, second, "new", false); status != 503 {
			t.Fatalf("busy = %d", status)
		}
		if _, err := conn.Write([]byte("old")); err != nil {
			t.Fatal(err)
		}
		if response, _ := wireResponse(t, conn); response.StatusCode != 204 {
			t.Fatalf("first upload = %d", response.StatusCode)
		}
		if status := w.put(t, first, "old", false); status != 409 {
			t.Fatalf("conflict = %d", status)
		}
		if status := w.put(t, second, "new", true); status != 500 {
			t.Fatalf("unsupported controller = %d", status)
		}
		if _, err := w.store.Inspect(t.Context(), transferKeyFromRequest(t, second)); err != domain.ErrObjectNotFound {
			t.Fatalf("unsupported transfer published: %v", err)
		}
		tmp := filepath.Join(w.root, "tmp")
		if err := os.Remove(tmp); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tmp, []byte("cannot create temporary directory"), 0600); err != nil {
			t.Fatal(err)
		}
		if status := w.put(t, second, "new", false); status != 500 {
			t.Fatalf("storage failure = %d", status)
		}
		if err := os.Remove(tmp); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(tmp, 0700); err != nil {
			t.Fatal(err)
		}
		if status := w.put(t, second, "new", false); status != 204 {
			t.Fatalf("retry after faults = %d", status)
		}
		w.emptyTemporaryDirectory(t)
	})
	t.Run("stalled download ends its write and releases the slot", func(t *testing.T) {
		files := budgetTestFiles()
		files.TransferIdleTimeout = 200 * time.Millisecond
		w := newBudgetWorld(t, files, 5*time.Second, 5*time.Second)
		data := strings.Repeat("a", 8<<20)
		grant, _ := w.grant(t, int64(len(data)))
		if err := w.store.Put(t.Context(), grant, strings.NewReader(data)); err != nil {
			t.Fatal(err)
		}
		info, err := w.store.Inspect(t.Context(), grant.Key)
		if err != nil {
			t.Fatal(err)
		}
		sealed, err := w.store.Seal(t.Context(), grant.Key, transferKey(t, 'o'), info.ETag)
		if err != nil {
			t.Fatal(err)
		}
		signed, err := w.store.SignRead(t.Context(), domain.ReadGrant{Key: sealed.Key, Method: "GET", Name: "large.bin", ContentType: "application/octet-stream", ExpiresAt: time.Now().Add(time.Minute)})
		if err != nil {
			t.Fatal(err)
		}
		connection, err := net.DialTimeout("tcp", w.address, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		conn := connection.(*net.TCPConn)
		defer func() { _ = conn.Close() }()
		if err := conn.SetReadBuffer(1024); err != nil {
			t.Fatal(err)
		}
		if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", signed.URL, w.address); err != nil {
			t.Fatal(err)
		}
		response, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode != 200 || response.ContentLength != int64(len(data)) {
			t.Fatalf("download did not start: %d, length %d", response.StatusCode, response.ContentLength)
		}
		// Do not drain the receive buffer. The real TCP write must time out well
		// before the five-second global deadline and the three-second total.
		select {
		case <-w.finished:
		case <-time.After(1500 * time.Millisecond):
			t.Fatal("stalled client retained the write/socket")
		}
		if err := conn.SetReadBuffer(4 << 20); err != nil {
			t.Fatal(err)
		}
		partial, err := io.ReadAll(response.Body)
		if !errors.Is(err, io.ErrUnexpectedEOF) || len(partial) >= len(data) || string(partial) != data[:len(partial)] {
			t.Fatalf("stalled response was not truncated: %d bytes, %v", len(partial), err)
		}
		w.nextUploadWorks(t)
	})
	t.Run("active download is not canceled by an idle request side", func(t *testing.T) {
		w := newBudgetWorld(t, budgetTestFiles(), 200*time.Millisecond, 200*time.Millisecond)
		data := strings.Repeat("a", 8<<20)
		grant, _ := w.grant(t, int64(len(data)))
		if err := w.store.Put(t.Context(), grant, strings.NewReader(data)); err != nil {
			t.Fatal(err)
		}
		info, err := w.store.Inspect(t.Context(), grant.Key)
		if err != nil {
			t.Fatal(err)
		}
		sealed, err := w.store.Seal(t.Context(), grant.Key, transferKey(t, 'o'), info.ETag)
		if err != nil {
			t.Fatal(err)
		}
		signed, err := w.store.SignRead(t.Context(), domain.ReadGrant{Key: sealed.Key, Method: "GET", Name: "paced.bin", ContentType: "application/octet-stream", ExpiresAt: time.Now().Add(time.Minute)})
		if err != nil {
			t.Fatal(err)
		}
		connection, err := net.DialTimeout("tcp", w.address, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		conn := connection.(*net.TCPConn)
		defer func() { _ = conn.Close() }()
		if err := conn.SetReadBuffer(64 << 10); err != nil {
			t.Fatal(err)
		}
		if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", signed.URL, w.address); err != nil {
			t.Fatal(err)
		}
		response, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode != 200 {
			t.Fatalf("download status = %d", response.StatusCode)
		}
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		chunk := make([]byte, 256<<10)
		for range 32 {
			<-ticker.C
			n, err := io.ReadFull(response.Body, chunk)
			if err != nil || n != len(chunk) || string(chunk) != data[:len(chunk)] {
				t.Fatalf("active download interrupted: %d bytes, %v", n, err)
			}
		}
		if n, err := response.Body.Read(chunk[:1]); n != 0 || err != io.EOF {
			t.Fatalf("download end = %d, %v", n, err)
		}
		w.nextUploadWorks(t)
	})
}
