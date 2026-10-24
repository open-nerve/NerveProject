package bootstrap

import (
	"context"
	"strings"
	"uuid"

	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	issuedomain "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Every seeded work item carries one comment, so a successful list proves it
// reached the correct issue; deleted projects must hide their comments too.
func (s issueSeed) comments(sd seeded) {
	for _, i := range seededItems() {
		slug, _, _ := strings.Cut(i.key, "/")
		by := s.ids[i.by]
		c := issuedomain.Comment{ID: sd.comment(i.key, i.name), WorkspaceID: s.workspaces[slug], ProjectID: s.projects[i.key], IssueID: sd.issue(i.key, i.name), ActorID: by, HTML: "<p>Seeded comment</p>", Text: "Seeded comment"}
		if err := s.store.InsertComment(context.Background(), issueapp.CommentWrite{Comment: c, By: by, At: s.now}); err != nil {
			s.t.Fatal(err)
		}
	}
}
func (s issueSeed) seededCommentRows(sd seeded) {
	for _, i := range seededItems() {
		got, err := s.store.Comments(context.Background(), sd.issue(i.key, i.name), nil, 100)
		gone := strings.HasPrefix(i.key, "gone/")
		if err != nil || !gone && (len(got) != 1 || got[0].HTML != "<p>Seeded comment</p>" || got[0].ActorID != s.ids[i.by]) || gone && len(got) != 0 {
			s.t.Fatalf("seeded comment %s/%s: %+v, %v", i.key, i.name, got, err)
		}
	}
}

// comment is the live comment on the named seeded issue.
func (s seeded) comment(key, name string) uuid.UUID {
	id, ok := s.comments[key+"|"+name]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no comment on %s/%s is seeded", key, name)
	}
	return id
}
