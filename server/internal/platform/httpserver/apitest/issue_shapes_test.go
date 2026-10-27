package apitest

import (
	"bytes"
	"encoding/json"
	"maps"

	"slices"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestIssueDetailShape(t *testing.T) {
	c := Load(t)
	issue, detail := c.doc.Components.Schemas["Issue"].Value, c.doc.Components.Schemas["IssueDetail"].Value
	for _, schema := range []*openapi3.Schema{issue, detail} {
		link := schema.Properties["link_count"]
		if !slices.Contains(schema.Required, "link_count") || link == nil || link.Value == nil || !link.Value.Type.Is("integer") || link.Value.Format != "int64" || link.Value.Min == nil || *link.Value.Min != 0 {
			t.Fatal("Issue link_count must be required nonnegative int64")
		}
	}
	properties := maps.Clone(detail.Properties)
	if _, ok := properties["description_html"]; !ok {
		t.Fatal("detail lacks description_html")
	}
	delete(properties, "description_html")
	// Compare serialized schemas, excluding parser-origin metadata that differs by source location.
	gotProperties, err := json.Marshal(properties)
	if err != nil {
		t.Fatal(err)
	}
	wantProperties, err := json.Marshal(issue.Properties)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotProperties, wantProperties) {
		t.Fatal("IssueDetail properties must equal Issue plus description_html")
	}
	required := slices.Clone(detail.Required)
	if !slices.Contains(required, "description_html") {
		t.Fatal("description must be required")
	}
	required = slices.DeleteFunc(required, func(s string) bool { return s == "description_html" })
	slices.Sort(required)
	want := slices.Clone(issue.Required)
	slices.Sort(want)
	if !slices.Equal(required, want) {
		t.Fatal("IssueDetail required must equal Issue plus description_html")
	}
	if len(detail.AllOf) > 0 {
		t.Fatal("detail must be explicit")
	}
}
