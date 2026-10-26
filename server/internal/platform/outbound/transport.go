package outbound

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sync"
	"time"
)

func (c *Client) transport(requestContext context.Context, target *url.URL, addresses []netip.Addr, timeout time.Duration) (*http.Transport, func()) {
	port := target.Port()
	if port == "" {
		port = "80"
		if target.Scheme == "https" {
			port = "443"
		}
	}
	var mutex sync.Mutex
	var connections []net.Conn
	closed := false
	connect := func(secure bool) (net.Conn, error) {
		// The same connect budget bounds all candidate dials and the TLS handshake.
		ctx, cancel := context.WithTimeout(requestContext, timeout)
		defer cancel()
		var last error
		for _, address := range addresses {
			conn, err := c.deps.dial(ctx, "tcp", net.JoinHostPort(address.String(), port))
			if err != nil {
				last = err
				continue
			}
			mutex.Lock()
			if closed {
				mutex.Unlock()
				_ = conn.Close()
				return nil, context.Canceled
			}
			connections = append(connections, conn)
			mutex.Unlock()
			if !secure {
				return conn, nil
			}
			tlsConn := tls.Client(conn, &tls.Config{
				MinVersion: tls.VersionTLS12,
				ServerName: target.Hostname(),
				RootCAs:    c.deps.roots,
			})
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				_ = conn.Close()
				last = err
				continue
			}
			return tlsConn, nil
		}
		return nil, last
	}
	transport := &http.Transport{
		Proxy:             nil,
		DisableKeepAlives: true,
		DialContext:       func(_ context.Context, _, _ string) (net.Conn, error) { return connect(false) },
		DialTLSContext:    func(_ context.Context, _, _ string) (net.Conn, error) { return connect(true) },
	}
	closeTransport := func() {
		mutex.Lock()
		closed = true
		for _, conn := range connections {
			_ = conn.Close()
		}
		connections = nil
		mutex.Unlock()
		transport.CloseIdleConnections()
	}
	return transport, closeTransport
}
