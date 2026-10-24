package app_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func TestDescriptionWriteIsAtomic(t *testing.T) {
	create, w := newCreate()
	html := "<p>Saved description</p>"
	got, err := create.Execute(as(alice), web, domain.NewIssue{Name: "Description", DescriptionHTML: &html})
	if err != nil {
		t.Fatal(err)
	}
	if got.DescriptionHTML != html || w.issues.issue.DescriptionHTML != html {
		t.Fatalf("saved description = %q; want %q", got.DescriptionHTML, html)
	}
}

func TestSkipDescriptionPreservesAudit(t *testing.T) {
	update, w, _ := newUpdate()
	html := "<p>New description</p>"
	got, err := update.Execute(as(alice), w.issues.issue.ID, domain.IssuePatch{DescriptionHTML: &html})
	if err != nil {
		t.Fatal(err)
	}
	if got.DescriptionHTML != html {
		t.Fatalf("description patch was ignored: %q", got.DescriptionHTML)
	}
}

func TestDescriptionCreateHasDefaultAndWriterVersion(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		create, w := newCreate()
		in := domain.NewIssue{Name: "Default"}
		html := "<p>Saved</p>"
		if explicit {
			in.DescriptionHTML = &html
		} else {
			html = "<p></p>"
		}
		got, err := create.Execute(as(alice), web, in)
		if err != nil {
			t.Fatal(err)
		}
		if len(w.issues.versions) != 1 {
			t.Fatal("no initial version")
		}
		v := w.issues.versions[0]
		if v.HTML != html || got.DescriptionHTML != html || v.OwnedByID != alice || v.CreatedByID != alice || v.UpdatedByID != alice || !v.LastSavedAt.Equal(clockNow) {
			t.Fatal(v, got)
		}
	}
}

func TestDescriptionAbsentAndOrdinaryNoopAvoidMentionDirectory(t *testing.T) {
	for _, withDescription := range []bool{false, true} {
		update, w, id := newUpdate()
		w.issues.issue.DescriptionHTML = "<p>Same</p>"
		p := domain.IssuePatch{}
		if withDescription {
			p.DescriptionHTML = name("<p>Same</p>")
		}
		if _, err := update.Execute(as(alice), id, p); err != nil {
			t.Fatal(err)
		}
		for _, call := range w.log.calls {
			if strings.HasPrefix(call, "MentionMembers") || strings.HasPrefix(call, "MentionIDs") || call == "Now" {
				t.Fatal("no-op read description dependencies", w.log.calls)
			}
		}
	}
}

func TestDescriptionMigrationWritesWithoutAudit(t *testing.T) {
	update, w, id := newUpdate()
	w.issues.issue.DescriptionHTML = "<p>Same</p>"
	skip := true
	got, err := update.Execute(as(alice), id, domain.IssuePatch{DescriptionHTML: name("<p><strong>Same</strong></p>"), SkipActivity: &skip})
	if err != nil {
		t.Fatal(err)
	}
	if got.DescriptionHTML != "<p><strong>Same</strong></p>" || w.issues.changed != nil || len(w.issues.activities) != 0 || len(w.issues.versions) != 0 || w.issues.descriptionWrites != 1 {
		t.Fatal("migration left audit", got, w.issues.changed, w.issues.versions)
	}
}

func TestDescriptionVersionErrorPropagates(t *testing.T) {
	create, w := newCreate()
	w.log.fail = "InsertDescriptionVersion"
	if _, err := create.Execute(as(alice), web, domain.NewIssue{Name: "Fail"}); !errors.Is(err, errDisk) {
		t.Fatal(err)
	}
	update, w, id := newUpdate()
	w.log.fail = "InsertDescriptionVersion"
	if _, err := update.Execute(as(alice), id, domain.IssuePatch{DescriptionHTML: name("<p>Changed</p>")}); !errors.Is(err, errDisk) {
		t.Fatal(err)
	}
}
