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
		// Bound the wire envelope before SDK parsing; the normalized object
		// metadata itself is limited to 16 KiB by objectInfo.
		MaxResponseHeaderBytes: 32 << 10,
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
	response, err := t.next.RoundTrip(req)
	if err != nil {
		return response, err
	}
	// HEAD advertises the object's byte size, but has no response body.
	if req.Method == http.MethodHead {
		return response, nil
	}
	if object, _ := req.Context().Value(objectBodyKey{}).(bool); object && response.StatusCode >= 200 && response.StatusCode < 300 {
		return response, nil
	}
	limit, ok := req.Context().Value(metadataBudgetKey{}).(int64)
	if !ok {
		limit = protocolMetadataLimit
	}
	if response.ContentLength > limit {
		_ = response.Body.Close()
		return nil, errors.New("S3 metadata response exceeds limit")
	}
	response.Body = &metadataBody{ReadCloser: response.Body, left: limit}
	return response, nil
}

func urlOrigin(u *url.URL) (string, error) {
	if u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Opaque != "" || u.Fragment != "" {
		return "", errors.New("invalid S3 request origin")
	}
	return u.Scheme + "://" + strings.ToLower(u.Host), nil
}
