package app_test

import (
	"context"
	"regexp"
	"strings"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// descriptionSanitizer is a pure port fake; real HTML behavior is tested through HTTP.
type descriptionSanitizer struct{}

func (descriptionSanitizer) Clean(input string) domain.RichText {
	if input == "" {
		input = "<p></p>"
	}
	text := strings.TrimSpace(regexp.MustCompile(`<[^>]*>`).ReplaceAllString(input, ""))
	return domain.RichText{HTML: input, Text: text, Identity: domain.ContentIdentity{Text: text}}
}
func (f *fakeIssues) MentionIDs(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, "MentionIDs", id); err != nil {
		return nil, err
	}
	return f.mentions, nil
}
func (f *fakeIssues) InsertMentions(ctx context.Context, r app.LinkRows) error {
	ids, err := f.link(ctx, "InsertMentions", r)
	if err == nil {
		f.mentions = append(f.mentions, ids...)
	}
	return err
}
func (f *fakeIssues) RemoveMentions(ctx context.Context, r app.UnlinkRows) error {
	ids, err := f.unlink(ctx, "RemoveMentions", r, f.mentions)
	if err == nil {
		f.mentions = ids
	}
	return err
}
func (f *fakeIssues) InsertDescriptionVersion(_ context.Context, v domain.DescriptionVersion) error {
	f.versions = append(f.versions, v)
	if f.log.fail == "InsertDescriptionVersion" {
		return errDisk
	}
	return nil
}
func (f *fakeIssues) UpdateDescriptionVersion(_ context.Context, v domain.DescriptionVersion) error {
	f.versions[len(f.versions)-1] = v
	return nil
}
func (f *fakeIssues) LatestDescriptionVersion(_ context.Context, _ uuid.UUID) (domain.DescriptionVersion, bool, error) {
	if len(f.versions) == 0 {
		return domain.DescriptionVersion{}, false, nil
	}
	return f.versions[len(f.versions)-1], true, nil
}
func (f *fakeIssues) WriteDescription(ctx context.Context, w app.DescriptionWrite) error {
	if err := f.log.add(ctx, "WriteDescription", w.IssueID); err != nil {
		return err
	}
	f.descriptionWrites++
	f.issue.DescriptionHTML = w.HTML
	f.issue.DescriptionStripped = &w.Text
	return nil
}
func (f *fakeIssues) LatestActivity(_ context.Context, _ uuid.UUID) (app.ActivityRecord, bool, error) {
	return app.ActivityRecord{}, false, nil
}
func (f *fakeIssues) MergeDescriptionActivity(_ context.Context, _ app.DescriptionActivityMerge) error {
	return nil
}
