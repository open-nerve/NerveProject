package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// Server is the HTTP server of a nerve process.
type Server struct {
	srv             *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

// NewServer serves h behind the platform middleware chain
// (request ID -> recover -> access log).
func NewServer(cfg config.ServerConfig, h http.Handler, logger *slog.Logger) *Server {
	return &Server{
		srv: &http.Server{
			Addr:              cfg.Addr,
			Handler:           middleware(h, logger),
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
		},
		logger:          logger,
		shutdownTimeout: cfg.ShutdownTimeout,
	}
}

// ListenAndServe listens on server.addr and then behaves like Serve.
func (s *Server) ListenAndServe(ctx context.Context) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", s.srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.srv.Addr, err)
	}
	return s.Serve(ctx, ln)
}

// Serve serves HTTP on ln until ctx is done. It then stops accepting
// connections and gives in-flight requests up to server.shutdown_timeout to
// finish; connections still open after that are closed and an error is
// returned. A clean shutdown returns nil.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	s.logger.InfoContext(ctx, "http server listening", slog.String("addr", ln.Addr().String()))
	served := make(chan error, 1)
	go func() { served <- s.srv.Serve(ln) }()

	select {
	case err := <-served:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	s.logger.InfoContext(ctx, "http server shutting down", slog.Duration("timeout", s.shutdownTimeout))
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.shutdownTimeout)
	defer cancel()
	if err := s.srv.Shutdown(shutdownCtx); err != nil {
		closeErr := s.srv.Close()
		<-served
		return fmt.Errorf("http server shutdown: %w", errors.Join(err, closeErr))
	}
	<-served // http.ErrServerClosed once Shutdown has begun
	s.logger.InfoContext(ctx, "http server stopped")
	return nil
}
