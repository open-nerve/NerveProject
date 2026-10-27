package outbound

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"
)

func (c *Client) transport(requestContext context.Context, timeout time.Duration) (*http.Transport, func()) {
	var mutex sync.Mutex
	var connections []net.Conn
	closed := false
	connect := func(canonicalAddress string, secure bool) (net.Conn, error) {
		// Transport supplies the same IDNA hostname it uses for the HTTP Host.
		host, port, err := net.SplitHostPort(canonicalAddress)
		if err != nil {
			return nil, err
		}
		addresses, err := c.resolve(requestContext, host)
		if err != nil {
			return nil, err
		}
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
				ServerName: host,
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
		DialContext:       func(_ context.Context, _, address string) (net.Conn, error) { return connect(address, false) },
		DialTLSContext:    func(_ context.Context, _, address string) (net.Conn, error) { return connect(address, true) },
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
