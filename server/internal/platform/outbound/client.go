package outbound

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Options configures a client's immutable address policy and user agent.
type Options struct {
	AllowCIDRs []netip.Prefix
	Version    string
}

// Limits bounds one complete GET, including all redirects and decoded bytes.
type Limits struct {
	ConnectTimeout, Timeout time.Duration
	MaxBytes                int64
}

// Response contains the final HTTP response, including non-success statuses.
type Response struct {
	FinalURL    string
	StatusCode  int
	ContentType string
	Body        []byte
}

type resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

type dependencies struct {
	resolver       resolver
	dial           func(context.Context, string, string) (net.Conn, error)
	interfaceAddrs func() ([]net.Addr, error)
	roots          *x509.CertPool
}

// Client performs credential-free GET requests to policy-approved addresses.
type Client struct {
	policy    policy
	deps      dependencies
	userAgent string
}

// New snapshots local interface addresses; failure to read them is fatal.
func New(options Options) (*Client, error) {
	return newClient(options, dependencies{
		resolver:       net.DefaultResolver,
		dial:           (&net.Dialer{}).DialContext,
		interfaceAddrs: net.InterfaceAddrs,
	})
}

func newClient(options Options, deps dependencies) (*Client, error) {
	addresses, err := deps.interfaceAddrs()
	if err != nil {
		return nil, fmt.Errorf("read outbound interfaces: %w", err)
	}
	local := make([]netip.Addr, 0, len(addresses))
	for _, address := range addresses {
		prefix, err := netip.ParsePrefix(address.String())
		if err != nil {
			return nil, fmt.Errorf("parse outbound interface: %w", err)
		}
		local = append(local, prefix.Addr())
	}
	p, err := newPolicy(options.AllowCIDRs, local)
	if err != nil {
		return nil, err
	}
	return &Client{policy: p, deps: deps, userAgent: "Nerve/" + options.Version + " (+link preview)"}, nil
}

// Get follows at most five redirects with a single deadline and byte budget.
func (c *Client) Get(parent context.Context, rawURL string, limits Limits) (Response, error) {
	if limits.ConnectTimeout <= 0 || limits.Timeout <= 0 || limits.MaxBytes <= 0 || limits.MaxBytes == math.MaxInt64 {
		return Response{}, errors.New("outbound limits must be positive and MaxBytes must allow a sentinel byte")
	}
	ctx, cancel := context.WithTimeout(parent, limits.Timeout)
	defer cancel()
	for redirects := 0; ; redirects++ {
		if err := ctx.Err(); err != nil {
			return Response{}, classify(err)
		}
		target, err := parseTarget(rawURL)
		if err != nil {
			return Response{}, err
		}
		transport, closeTransport := c.transport(ctx, limits.ConnectTimeout)
		response, err := c.request(ctx, transport, target)
		if err != nil {
			closeTransport()
			return Response{}, classify(err)
		}
		location := response.Header.Get("Location")
		if isRedirect(response.StatusCode) && location != "" {
			closeTransport()
			_ = response.Body.Close()
			if redirects == 5 {
				return Response{}, &redirectLimitError{}
			}
			next, err := target.Parse(location)
			if err != nil {
				return Response{}, fmt.Errorf("%w: %w", ErrBlocked, err)
			}
			rawURL = next.String()
			continue
		}
		body, err := io.ReadAll(io.LimitReader(response.Body, limits.MaxBytes+1))
		closeTransport()
		_ = response.Body.Close()
		if err != nil {
			return Response{}, classify(err)
		}
		if int64(len(body)) > limits.MaxBytes {
			return Response{}, ErrTooLarge
		}
		return Response{FinalURL: target.String(), StatusCode: response.StatusCode, ContentType: response.Header.Get("Content-Type"), Body: body}, nil
	}
}

func (c *Client) request(ctx context.Context, transport *http.Transport, target *url.URL) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", c.userAgent)
	return transport.RoundTrip(request)
}

func parseTarget(raw string) (*url.URL, error) {
	target, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBlocked, err)
	}
	target.Scheme = strings.ToLower(target.Scheme)
	host := target.Hostname()
	if (target.Scheme != "http" && target.Scheme != "https") || host == "" || target.User != nil || target.Opaque != "" || strings.Contains(host, "%") {
		return nil, ErrBlocked
	}
	if strings.HasPrefix(target.Host, "[") || strings.Contains(host, ":") {
		if _, err := netip.ParseAddr(host); err != nil {
			return nil, ErrBlocked
		}
	}
	if port := target.Port(); port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return nil, ErrBlocked
		}
	} else if strings.HasSuffix(target.Host, ":") {
		return nil, ErrBlocked
	}
	return target, nil
}

func (c *Client) resolve(ctx context.Context, host string) ([]netip.Addr, error) {
	address, err := netip.ParseAddr(host)
	addresses := []netip.Addr{address}
	if err != nil {
		addresses, err = c.deps.resolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, classify(err)
		}
	}
	if len(addresses) == 0 {
		return nil, ErrUnreachable
	}
	// Validate every candidate before any connection can be attempted.
	for _, address := range addresses {
		if err := c.policy.check(address); err != nil {
			return nil, err
		}
	}
	return addresses, nil
}

func isRedirect(status int) bool {
	return status == 301 || status == 302 || status == 303 || status == 307 || status == 308
}
