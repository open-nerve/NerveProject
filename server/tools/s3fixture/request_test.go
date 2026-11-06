package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

func testRequest(endpoint string) request {
	return request{Action: "inspect", Endpoint: endpoint, Region: "us-east-1", Bucket: "nerve-e2e-" + strings.Repeat("a", 32), Prefix: "e2e/" + strings.Repeat("b", 32) + "/", Key: "proof", AccessKeyID: "generated-access", SecretAccessKey: "generated-secret"}
}

func TestInputDeadlineClosesPipe(t *testing.T) {
	reader, writer := io.Pipe()
	defer func() { _ = writer.Close(); _ = reader.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- run(ctx, reader, io.Discard) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("unfinished input succeeded")
		}
	case <-time.After(time.Second):
		_ = writer.Close()
		<-done
		t.Fatal("input read ignored the command deadline")
	}
}

func TestInvalidRequests(t *testing.T) {
	valid := testRequest("http://127.0.0.1:8333")
	for name, mutate := range map[string]func(*request){
		"unknown action":        func(r *request) { r.Action = "presign" },
		"external endpoint":     func(r *request) { r.Endpoint = "https://bucket.example.com" },
		"path":                  func(r *request) { r.Endpoint += "/api" },
		"credentials in URL":    func(r *request) { r.Endpoint = "http://secret@127.0.0.1:8333" },
		"invalid port":          func(r *request) { r.Endpoint = "http://127.0.0.1:65536" },
		"existing bucket":       func(r *request) { r.Bucket = "production" },
		"root prefix":           func(r *request) { r.Prefix = "" },
		"outside key":           func(r *request) { r.Key = "../private" },
		"missing secret":        func(r *request) { r.SecretAccessKey = "" },
		"oversized secret":      func(r *request) { r.SecretAccessKey = strings.Repeat("s", 4097) },
		"wildcard CORS":         func(r *request) { r.Action = "cors"; r.Key = ""; r.Origin = "*" },
		"missing CORS":          func(r *request) { r.Action = "cors"; r.Key = "" },
		"unneeded origin":       func(r *request) { r.Origin = "http://127.0.0.1:8334" },
		"key on bucket removal": func(r *request) { r.Action = "delete-bucket" },
		"origin on bucket check": func(r *request) {
			r.Action = "head-bucket"
			r.Key = ""
			r.Origin = "http://127.0.0.1:8334"
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := valid
			mutate(&r)
			data, _ := json.Marshal(r)
			var out bytes.Buffer
			if err := run(context.Background(), bytes.NewReader(data), &out); err == nil {
				t.Fatal("invalid request accepted")
			}
			if out.Len() != 0 {
				t.Fatal("invalid request produced output")
			}
		})
	}
	for _, input := range []string{"null", "{} {}", `{"unknown":true}`, strings.Repeat(" ", (1<<20)+1)} {
		var out bytes.Buffer
		if err := run(context.Background(), strings.NewReader(input), &out); err == nil {
			t.Fatal("invalid JSON envelope accepted")
		}
	}
}
