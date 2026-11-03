package s3

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func newTransport() *http.Transport {
	return &http.Transport{
		Proxy: nil, DialContext: (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2: true, MaxIdleConns: 32, MaxIdleConnsPerHost: 16, IdleConnTimeout: 90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second, ExpectContinueTimeout: time.Second,
	}
}

type originTransport struct {
	origin string
	next   http.RoundTripper
}

func (t originTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	origin, err := urlOrigin(req.URL)
	if err != nil || origin != t.origin || (req.Host != "" && !strings.EqualFold(req.Host, req.URL.Host)) {
		return nil, errors.New("S3 request destination is outside configured origin")
	}
	return t.next.RoundTrip(req)
}

func urlOrigin(u *url.URL) (string, error) {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Opaque != "" || u.Fragment != "" {
		return "", errors.New("invalid S3 request origin")
	}
	return u.Scheme + "://" + strings.ToLower(u.Host), nil
}
