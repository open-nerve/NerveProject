package app_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

type listAttachmentCatalogFunc func(context.Context, []uuid.UUID, uuid.UUID, int) ([]app.AttachmentCount, error)

func (f listAttachmentCatalogFunc) ProjectAttachmentCounts(ctx context.Context, ids []uuid.UUID, after uuid.UUID, size int) ([]app.AttachmentCount, error) {
	return f(ctx, ids, after, size)
}

func TestListAttachmentCountDirectory(t *testing.T) {
	_, f, in := listingFixture(t)
	in.Query.Order = "-attachment_count"
	p2 := uuid.NewV7()
	in.Projects = append(in.Projects, app.VisibleProject{ID: p2}, in.Projects[0])
	f.pids = []uuid.UUID{web, p2, web}
	wantProjects := []uuid.UUID{web, p2}
	slices.SortFunc(wantProjects, func(a, b uuid.UUID) int { return a.Compare(b) })
	counts := make([]app.AttachmentCount, 1001)
	for n := range counts {
		counts[n] = app.AttachmentCount{IssueID: uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012x", n+1)), Count: int64(n + 1)}
	}
	for _, fault := range []string{"", "first error", "second error", "duplicate", "negative", "zero count", "zero ID", "unsorted", "oversized", "cross-page duplicate"} {
		t.Run(fault, func(t *testing.T) {
			f.reads = nil
			calls := 0
			catalog := listAttachmentCatalogFunc(func(ctx context.Context, projects []uuid.UUID, after uuid.UUID, size int) ([]app.AttachmentCount, error) {
				if ctx != f.ctx || size != 1000 || !slices.Equal(projects, wantProjects) {
					t.Fatal("directory scope/context/batch", projects, after, size)
				}
				calls++
				start := (calls - 1) * 1000
				if calls > 2 || (calls == 1 && after != (uuid.UUID{})) || (calls == 2 && after != counts[999].IssueID) {
					t.Fatal("global after", calls, after)
				}
				if fault == "first error" && calls == 1 || fault == "second error" && calls == 2 {
					return nil, errDisk
				}
				page := slices.Clone(counts[start:min(start+size, len(counts))])
				if calls == 1 {
					switch fault {
					case "duplicate":
						page[1] = page[0]
					case "negative":
						page[0].Count = -1
					case "zero count":
						page[0].Count = 0
					case "zero ID":
						page[0].IssueID = uuid.UUID{}
					case "unsorted":
						page[0], page[1] = page[1], page[0]
					case "oversized":
						page = slices.Clone(counts)
					}
				}
				if calls == 2 && fault == "cross-page duplicate" {
					page[0] = counts[999]
				}
				return page, nil
			})
			listing := app.NewListing(app.ListingDeps{Attachments: noIssueAttachments{}, AttachmentCatalog: catalog, Rows: f, States: f, Labels: f, Assignees: f, Profiles: f, Issues: f})
			_, err := listing.Read(f.ctx, in)
			if fault == "" {
				if err != nil || calls != 2 || len(f.reads) != 1 || !reflect.DeepEqual(f.reads[0].Directories.AttachmentCounts, counts) {
					t.Fatalf("incomplete directory: calls=%d reads=%d err=%v", calls, len(f.reads), err)
				}
			} else {
				if err == nil || len(f.reads) != 0 {
					t.Fatal("malformed/failed directory reached list SQL", fault, err)
				}
				if (fault == "first error" || fault == "second error") && !errors.Is(err, errDisk) {
					t.Fatal("lost directory error", err)
				}
			}
		})
	}
}
