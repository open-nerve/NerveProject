package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestDescriptionChanges(t *testing.T) {
	t.Run("input and length", func(t *testing.T) {
		n, err := CheckNewIssue(NewIssue{Name: "work"})
		if err != nil || n.DescriptionHTML == nil || *n.DescriptionHTML != "<p></p>" {
			t.Fatalf("default description = %v, %v", n.DescriptionHTML, err)
		}
		for _, text := range []string{"", strings.Repeat("界", 500000), strings.Repeat("界", 500001)} {
			_, createErr := CheckNewIssue(NewIssue{Name: "work", DescriptionHTML: &text})
			_, patchErr := CheckIssuePatch(IssuePatch{DescriptionHTML: &text})
			for _, err := range []error{createErr, patchErr} {
				if len([]rune(text)) <= 500000 {
					if err != nil {
						t.Fatalf("accepted length rejected: %v", err)
					}
					continue
				}
				var p *shared.Error
				if !errors.As(err, &p) || p.ProblemStatus() != 422 || len(p.Fields) != 1 || p.Fields[0].Field != "description_html" || p.Fields[0].Code != shared.FieldTooLong {
					t.Fatalf("length error: %v", err)
				}
			}
		}
	})
	t.Run("skip presence and all other fields", func(t *testing.T) {
		id := uuid.NewV7()
		empty := []uuid.UUID{}
		for _, skip := range []bool{false, true} {
			_, err := CheckIssuePatch(IssuePatch{SkipActivity: &skip})
			assertSkipProblem(t, err)
			for name, other := range map[string]IssuePatch{
				"name": {Name: ptr("x")}, "state": {StateID: &id}, "priority": {Priority: ptr(PriorityLow)},
				"assignees": {AssigneeIDs: &empty}, "labels": {LabelIDs: &empty},
				"start null": {SetStartDate: true}, "target null": {SetTargetDate: true}, "parent null": {SetParentID: true},
				"start": {SetStartDate: true, StartDate: october(1)}, "target": {SetTargetDate: true, TargetDate: october(2)}, "parent": {SetParentID: true, ParentID: &id},
			} {
				t.Run(name+" skip="+map[bool]string{true: "true", false: "false"}[skip], func(t *testing.T) {
					other.SkipActivity = &skip
					other.DescriptionHTML = ptr("")
					_, err := CheckIssuePatch(other)
					assertSkipProblem(t, err)
				})
			}
			p := IssuePatch{SkipActivity: &skip, DescriptionHTML: ptr("")}
			got, err := CheckIssuePatch(p)
			if err != nil || !reflect.DeepEqual(got, p) {
				t.Fatalf("description alone: %+v %v", got, err)
			}
		}
		_, err := CheckIssuePatch(IssuePatch{Name: ptr(""), DescriptionHTML: ptr(strings.Repeat("x", 500001)), SkipActivity: ptr(false)})
		var p *shared.Error
		if !errors.As(err, &p) || len(p.Fields) != 3 {
			t.Fatalf("must aggregate name, description and skip: %v", err)
		}
	})
	t.Run("comparison", func(t *testing.T) {
		a, b := uuid.NewV7(), uuid.NewV7()
		old := RichText{HTML: "<p>one</p>", Text: "one", Identity: ContentIdentity{Text: "one", MentionIDs: []uuid.UUID{a, b}, Links: []string{"https://a.test"}, Checked: []bool{false}}}
		same := CompareDescription(old, old, false)
		if same.Write || same.Changed {
			t.Fatal("unchanged ordinary save wrote")
		}
		p, err := CheckIssuePatch(IssuePatch{DescriptionHTML: &old.HTML, SkipActivity: ptr(false)})
		if err != nil {
			t.Fatal(err)
		}
		explicit := old
		explicit.HTML = *p.DescriptionHTML
		same = CompareDescription(old, explicit, *p.SkipActivity)
		if same.Write || same.Changed {
			t.Fatal("explicit skip=false wrote unchanged HTML")
		}
		withIDs := old
		withIDs.HTML = `<p data-id="a">one</p>`
		c := CompareDescription(old, withIDs, true)
		if !c.Write || c.Changed || !reflect.DeepEqual(c.Content, withIDs) {
			t.Fatal("migration update was audited or lost", c)
		}
		c = CompareDescription(old, withIDs, false)
		if !c.Write || !c.Changed {
			t.Fatal("ordinary different HTML not audited", c)
		}
		for name, next := range map[string]RichText{
			"text":     {HTML: "changed", Identity: ContentIdentity{Text: "two", MentionIDs: []uuid.UUID{a, b}, Links: old.Identity.Links, Checked: old.Identity.Checked}},
			"mentions": {HTML: "changed", Identity: ContentIdentity{Text: "one", MentionIDs: []uuid.UUID{b, a}, Links: old.Identity.Links, Checked: old.Identity.Checked}},
			"href":     {HTML: "changed", Identity: ContentIdentity{Text: "one", MentionIDs: old.Identity.MentionIDs, Links: []string{"https://b.test"}, Checked: old.Identity.Checked}},
			"checked":  {HTML: "changed", Identity: ContentIdentity{Text: "one", MentionIDs: old.Identity.MentionIDs, Links: old.Identity.Links, Checked: []bool{true}}},
		} {
			c := CompareDescription(old, next, true)
			if !c.Write || !c.Changed {
				t.Errorf("%s hidden: %+v", name, c)
			}
		}
		c = CompareDescription(old, old, true)
		if !c.Write || c.Changed {
			t.Fatal("same migration must write without audit", c)
		}
	})
}

func assertSkipProblem(t *testing.T, err error) {
	t.Helper()
	var p *shared.Error
	if !errors.As(err, &p) || p.ProblemStatus() != 422 {
		t.Fatalf("skip must be rejected: %v", err)
	}
	for _, f := range p.Fields {
		if f.Field == "skip_activity" && f.Code == shared.FieldNotAllowed {
			return
		}
	}
	t.Fatalf("missing skip_activity/not_allowed: %v", err)
}
