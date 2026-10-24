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
		v := issuedomain.DescriptionVersion{ID: sd.descriptionVersion(i.key, i.name), WorkspaceID: c.WorkspaceID, ProjectID: c.ProjectID, IssueID: c.IssueID, OwnedByID: by, CreatedByID: by, UpdatedByID: by, HTML: "<p>Seeded description</p>", Text: "Seeded description", LastSavedAt: s.now, CreatedAt: s.now, UpdatedAt: s.now}
		if err := s.store.InsertDescriptionVersion(context.Background(), v); err != nil {
			s.t.Fatal(err)
		}
		if err := s.store.InsertIssueReaction(context.Background(), issueapp.IssueReactionWrite{Reaction: issuedomain.IssueReaction{ID: uuid.NewV7(), WorkspaceID: c.WorkspaceID, ProjectID: c.ProjectID, IssueID: c.IssueID, ActorID: by, Reaction: "1"}, By: by, At: s.now}); err != nil {
			s.t.Fatal(err)
		}
		if err := s.store.InsertComment(context.Background(), issueapp.CommentWrite{Comment: c, By: by, At: s.now}); err != nil {
			s.t.Fatal(err)
		}
		if err := s.store.InsertCommentReaction(context.Background(), issueapp.CommentReactionWrite{Reaction: issuedomain.CommentReaction{ID: uuid.NewV7(), WorkspaceID: c.WorkspaceID, ProjectID: c.ProjectID, CommentID: c.ID, ActorID: by, Reaction: "1"}, By: by, At: s.now}); err != nil {
			s.t.Fatal(err)
		}
	}
}
func (s issueSeed) seededCommentRows(sd seeded) {
	for _, i := range seededItems() {
		versions, versionErr := s.store.DescriptionVersions(context.Background(), sd.issue(i.key, i.name), nil, 100)
		versionGone := strings.HasPrefix(i.key, "gone/")
		if versionErr != nil || !versionGone && (len(versions) != 1 || versions[0].HTML != "<p>Seeded description</p>" || versions[0].OwnedByID != s.ids[i.by]) || versionGone && len(versions) != 0 {
			s.t.Fatal("seeded description versions", versions, versionErr)
		}

		reactions, e := s.store.IssueReactions(context.Background(), sd.issue(i.key, i.name))
		goneReaction := strings.HasPrefix(i.key, "gone/")
		if e != nil || !goneReaction && (len(reactions) != 1 || reactions[0].Reaction != "1") || goneReaction && len(reactions) != 0 {
			s.t.Fatal("seeded reactions", reactions, e)
		}
		cr, ce := s.store.CommentReactions(context.Background(), []uuid.UUID{sd.comment(i.key, i.name)})
		if ce != nil || !goneReaction && (len(cr) != 1 || cr[0].Reaction != "1") || goneReaction && len(cr) != 0 {
			s.t.Fatal("seeded comment reactions", cr, ce)
		}
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

// descriptionVersion is the saved description on the named seeded work item.
func (s seeded) descriptionVersion(key, name string) uuid.UUID {
	id, ok := s.descriptionVersions[key+"|"+name]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no description version on %s/%s is seeded", key, name)
	}
	return id
}
