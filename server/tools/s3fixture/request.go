package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type request struct {
	Action          string `json:"action"`
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	Prefix          string `json:"prefix"`
	Key             string `json:"key,omitempty"`
	Origin          string `json:"origin,omitempty"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

var ownedBucket = regexp.MustCompile(`^nerve-e2e-[a-f0-9]{32}$`)
var ownedPrefix = regexp.MustCompile(`^e2e/[a-f0-9]{32}/$`)
var objectKey = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func decodeRequest(data []byte) (request, error) {
	var r request
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(&r) != nil || d.Decode(new(any)) != io.EOF {
		return r, errRequest
	}
	if !loopbackOrigin(r.Endpoint) || r.Region != "us-east-1" || !ownedBucket.MatchString(r.Bucket) || !ownedPrefix.MatchString(r.Prefix) || !secret(r.AccessKeyID, 256) || !secret(r.SecretAccessKey, 4096) {
		return r, errRequest
	}
	switch r.Action {
	case "create-bucket":
		if r.Key != "" || r.Origin != "" {
			return r, errRequest
		}
	case "cors":
		if r.Key != "" || !loopbackOrigin(r.Origin) {
			return r, errRequest
		}
	case "inspect", "delete":
		if !objectKey.MatchString(r.Key) || r.Origin != "" {
			return r, errRequest
		}
	case "delete-bucket", "head-bucket":
		if r.Key != "" || r.Origin != "" {
			return r, errRequest
		}
	default:
		return r, errRequest
	}
	return r, nil
}

func secret(s string, limit int) bool {
	return len(s) > 0 && len(s) <= limit && !strings.ContainsFunc(s, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) })
}

// No production endpoint can be administered by this helper. The fixture
// owns a random bucket/prefix and an explicitly loopback-bound service.
func loopbackOrigin(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.Path != "" || u.RawPath != "" || u.Opaque != "" || strings.ContainsAny(raw, "?#") {
		return false
	}
	port, err := strconv.Atoi(u.Port())
	return err == nil && port > 0 && port <= 65535 && raw == "http://127.0.0.1:"+u.Port()
}
