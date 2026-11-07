package postgresadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var _ app.LinkMetadataWriter = (*postgresadapter.Store)(nil)

// A crawl's metadata is written only while the link it names is undeleted and has the address it crawled, as it is
// spelt (M6 P2 spec R11): not against the link's old address, nor in another case, nor on a deleted link whose
// address a live one has since, nor on another id. It writes the metadata alone: every other column of the links, and
// the modules, as they were, in a deleted project too. Its failure keeps its chain.
func TestSetModuleLinkMetadata(t *testing.T) {
	w := newLinksWorld(t)
	ctx := context.Background()
	at := now.Truncate(time.Microsecond)
	const oldURL, newURL = "HTTP://Example.com/A", "http://Example.com/B"
	title := "User title"
	live, deleted, replacement := w.link(w.auth, oldURL, at), w.link(w.auth, oldURL, at), w.link(w.auth, oldURL, at)
	live.Title = &title
	updater := account(t, w.pool, "upd")
	if err := w.s.InsertModuleLink(ctx, live); err != nil {
		t.Fatal(err)
	}
	live.URL, live.UpdatedAt = newURL, at.Add(time.Minute)
	if _, err := w.s.UpdateModuleLink(ctx, live, updater); err != nil {
		t.Fatal(err)
	}
	if err := w.s.InsertModuleLink(ctx, deleted); err != nil {
		t.Fatal(err)
	}
	if err := w.s.DeleteModuleLink(ctx, deleted.ID, at.Add(time.Minute), updater); err != nil {
		t.Fatal(err)
	}
	if err := w.s.InsertModuleLink(ctx, replacement); err != nil {
		t.Fatal(err)
	}
	// A deleted project's link, still live, may yet be written.
	exec(t, w.pool, "UPDATE projects SET deleted_at = $1 WHERE id = $2", at, w.web)
	snapshot := func() string {
		var got string
		if err := w.pool.QueryRow(ctx, `SELECT jsonb_build_object(
			'links', (SELECT jsonb_agg(to_jsonb(l) - 'metadata' ORDER BY id) FROM module_links l),
			'modules', (SELECT jsonb_agg(to_jsonb(m) ORDER BY id) FROM modules m))::text`).Scan(&got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	before := snapshot()
	crawled, resultURL := "Fetched title", newURL
	metadata := shared.LinkMetadata{"title": &crawled, "url": &resultURL, "favicon": nil, "favicon_url": nil}
	for _, tc := range []struct {
		name string
		id   uuid.UUID
		url  string
		want bool
	}{
		{"the live link at its address", live.ID, newURL, true},
		{"its old address", live.ID, oldURL, false},
		{"its address in another case", live.ID, "HTTP://Example.com/B", false},
		{"a deleted link whose address a live one has", deleted.ID, oldURL, false},
		{"another id", uuid.NewV7(), oldURL, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			matched, err := w.s.SetModuleLinkMetadata(ctx, tc.id, tc.url, metadata)
			if err != nil || matched != tc.want {
				t.Fatalf("SetModuleLinkMetadata = %t, %v; want %t", matched, err, tc.want)
			}
			if got := snapshot(); got != before {
				t.Fatalf("the metadata's write changed another column: %s", got)
			}
			for _, id := range []uuid.UUID{live.ID, deleted.ID, replacement.ID} {
				var data []byte
				if err := w.pool.QueryRow(ctx, "SELECT metadata FROM module_links WHERE id = $1", id).Scan(&data); err != nil {
					t.Fatal(err)
				}
				var got shared.LinkMetadata
				if err := json.Unmarshal(data, &got); err != nil {
					t.Fatal(err)
				}
				want := shared.LinkMetadata{}
				if id == live.ID {
					want = metadata
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("link %s's metadata = %s; want %v", id, data, want)
				}
			}
		})
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if matched, err := w.s.SetModuleLinkMetadata(cancelled, live.ID, newURL, metadata); matched || !errors.Is(err, context.Canceled) {
		t.Fatalf("on a cancelled context = %t, %v; want context.Canceled", matched, err)
	}
}
