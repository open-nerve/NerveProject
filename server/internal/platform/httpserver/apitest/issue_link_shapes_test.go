package apitest

import "testing"

func TestIssueLinkMetadataError(t *testing.T) {
	c := Load(t)
	for _, tc := range []struct {
		body  string
		valid bool
	}{
		{`{}`, true},
		{`{"error":null}`, true},
		{`{"error":"blocked"}`, true},
		{`{"error":"failed"}`, true},
		{`{"error":"timeout"}`, false},
		{`{"error":"unreachable"}`, false},
		{`{"error":""}`, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			err := c.validateSchema("IssueLinkMetadata", []byte(tc.body))
			if (err == nil) != tc.valid {
				t.Errorf("schema accepted=%v want %v: %v", err == nil, tc.valid, err)
			}
		})
	}
}
