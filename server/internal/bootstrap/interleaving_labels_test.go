package bootstrap

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Interleaving 11, the two levels of labels (M3 design 9.3, 3.16), on
// memberWorld's real database, in both orders: project's CreateLabel,
// UpdateLabel and DeleteLabel over projectLocks, with the Authorizer as
// bootstrap wires them, the first held at a gate past its write. Every
// wait has a deadline.

// labelStore is what a write on a label reads and writes through:
// project's store, or one held at a gate.
type labelStore interface {
	projectapp.LabelCreator
	projectapp.LabelUpdater
	projectapp.LabelDeleter
}

// labelWrittenHolding stops a write on a label past its write: once it
// holds the workspace FOR SHARE, the project FOR NO KEY UPDATE and the rows
// it wrote, before its commit.
type labelWrittenHolding struct {
	*projectpg.Store
	gate *gate
}

func (s labelWrittenHolding) CreateLabel(ctx context.Context, r projectapp.LabelRow) (projectdomain.Label, error) {
	created, err := s.Store.CreateLabel(ctx, r)
	if err != nil {
		return projectdomain.Label{}, err
	}
	return created, s.gate.wait(ctx)
}

func (s labelWrittenHolding) UpdateLabel(ctx context.Context, id uuid.UUID, p projectdomain.LabelPatch, by uuid.UUID,
	now time.Time) (projectdomain.Label, error) {
	changed, err := s.Store.UpdateLabel(ctx, id, p, by, now)
	if err != nil {
		return projectdomain.Label{}, err
	}
	return changed, s.gate.wait(ctx)
}

func (s labelWrittenHolding) DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := s.Store.DeleteLabel(ctx, id, by, now); err != nil {
		return err
	}
	return s.gate.wait(ctx)
}

// worldLabels are the labels seedLabelsOf gives Web and Ops, by name, each
// with the name of the label it is under, "" at the top: Bug with UI under
// it, Feature and Docs at the top. Ops's are Web's names in another project,
// which no write on Web's touches.
var worldLabels = map[string]string{"Bug": "", "UI": "Bug", "Feature": "", "Docs": ""}

// seedLabelsOf stores worldLabels in project through the project store, by
// alice, an admin of Web and of Ops, whom the rules let write their labels,
// each parent before the labels under it, reads them back (labelsOf), and
// answers their ids by name.
func (w memberWorld) seedLabelsOf(t *testing.T, project uuid.UUID) map[string]uuid.UUID {
	t.Helper()
	var workspace uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT workspace_id FROM projects WHERE id = $1", project).Scan(&workspace); err != nil {
		t.Fatal(err)
	}
	ids, store := map[string]uuid.UUID{}, projectpg.New(w.pool)
	for _, name := range []string{"Bug", "UI", "Feature", "Docs"} {
		ids[name] = uuid.NewV7()
		var parent *uuid.UUID
		if p := worldLabels[name]; p != "" {
			id := ids[p]
			parent = &id
		}
		if _, err := store.CreateLabel(pgtest.Soon(t), projectapp.LabelRow{ID: ids[name], WorkspaceID: workspace, ProjectID: project, ParentID: parent,
			Name: name, SortOrder: float64(len(ids)), CreatedBy: w.ids["alice"], Now: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	if got := w.labelsOf(t, project); !maps.Equal(got, worldLabels) {
		t.Fatalf("the labels seeded in %s read back as %v; want %v", project, got, worldLabels)
	}
	return ids
}

// labelsOf is project's undeleted labels by name, each with the name of the
// label it is under, "" at the top.
func (w memberWorld) labelsOf(t *testing.T, project uuid.UUID) map[string]string {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT l.name, coalesce(p.name, '') FROM labels l LEFT JOIN labels p ON p.id = l.parent_id
		WHERE l.project_id = $1 AND l.deleted_at IS NULL`, project)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	labels := map[string]string{}
	for rows.Next() {
		var name, parent string
		if err := rows.Scan(&name, &parent); err != nil {
			t.Fatal(err)
		}
		labels[name] = parent
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return labels
}

// labelWrite is bob's write on Web's labels, through project's use case
// over a store, with projectLocks, as project.New wires it, given the ids
// of Web's labels by name; apply is what it does to Web's labelsOf.
type labelWrite struct {
	name  string
	run   func(ctx context.Context, w memberWorld, labels labelStore, ids map[string]uuid.UUID) error
	apply func(labels map[string]string)
}

// labelCreates is the creation of the label name under parent, "" at the
// top.
func labelCreates(name, parent string) labelWrite {
	where := "under " + parent
	if parent == "" {
		where = "at the top"
	}
	return labelWrite{"creates " + name + " " + where, func(ctx context.Context, w memberWorld, labels labelStore,
		ids map[string]uuid.UUID) error {
		in := projectdomain.LabelCreate{Name: name}
		if parent != "" {
			id := ids[parent]
			in.ParentID = &id
		}
		_, err := projectapp.NewCreateLabel(w.projectLocks(), labels, w.tx(), clock.System{}).Execute(w.asBob(ctx), w.web, in)
		return err
	}, func(labels map[string]string) { labels[name] = parent }}
}

// labelMoves puts the label name under parent.
func labelMoves(name, parent string) labelWrite {
	return labelWrite{"moves " + name + " under " + parent, func(ctx context.Context, w memberWorld, labels labelStore,
		ids map[string]uuid.UUID) error {
		id := ids[parent]
		_, err := projectapp.NewUpdateLabel(w.projectLocks(), labels, w.tx(), clock.System{}).Execute(w.asBob(ctx), ids[name],
			projectdomain.LabelPatch{SetParent: true, ParentID: &id})
		return err
	}, func(labels map[string]string) { labels[name] = parent }}
}

// labelDeletes deletes the label name, and the labels under it with it.
func labelDeletes(name string) labelWrite {
	return labelWrite{"deletes " + name, func(ctx context.Context, w memberWorld, labels labelStore, ids map[string]uuid.UUID) error {
		return projectapp.NewDeleteLabel(w.projectLocks(), labels, w.tx(), clock.System{}).Execute(w.asBob(ctx), ids[name])
	}, func(labels map[string]string) {
		for n, parent := range labels {
			if n == name || parent == name {
				delete(labels, n)
			}
		}
	}}
}

// parentRefused is the 422 a parent the rules of 3.16 refuse answers, for
// why.
func parentRefused(why string) error {
	return shared.Invalid(shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed, Message: why})
}

// sameLabelOutcome is sameOutcome, the refused fields too.
func sameLabelOutcome(err, want error) bool {
	var got, refused *shared.Error
	return sameOutcome(err, want) && (want == nil || errors.As(err, &got) && errors.As(want, &refused) && slices.Equal(got.Fields, refused.Fields))
}

// Two writes on Web's labels at once serialize on Web's row (M3 design
// 3.6, 3.16; 9.3, interleaving 11), in both orders. The first holds acme
// FOR SHARE, Web FOR NO KEY UPDATE and the rows it wrote, and waits at its
// gate past its write. The second shares acme and waits for Web's row:
// neither writes a row of projects, so only that lock's wait satisfies the
// probe. Once the first has committed, the second decides on what it
// committed, its parent and the label's children read under the lock:
//   - Feature under Docs and Docs under Feature: the second's parent has a
//     parent now; no label is ever under a label under another, and none
//     under itself;
//   - Icons created under Feature and Feature moved under Docs: created
//     first, Feature has a label under it and takes no parent; moved
//     first, Feature has a parent and takes no label under it;
//   - QA and qa created: the second's name is taken, in any case;
//   - Feature moved under Docs and under Bug: the second's own label moved
//     while it waited, and the second moves it again, from where the first
//     put it, to where it asks;
//   - Bug deleted, and Feature moved or Icons created under it (convention
//     5's probe): deleted first, Bug is no label of the project any more;
//     the other first, the deletion's one statement takes the label just
//     put under Bug with Bug and UI, under the lock that every write of
//     Web's labels takes;
//   - QA created at the top and Icons under Bug: both are made, the second
//     at the time its clock gives once it holds the lock.
//
// Web's labels are then as the first left them, and the second's too when
// it succeeded, at a moment no earlier than the gate's opening, each at a
// sort order of its own: a new label's place is read under the project's
// lock (M3 design 4.10). Ops's are as they were.
func TestLabelWritesOnOneProjectSerialize(t *testing.T) {
	ofTheProject := parentRefused("must be a label of the project")
	withoutParent := parentRefused("must be a label without a parent: labels have two levels")
	childless := parentRefused("must be null: the label has labels under it, and labels have two levels")
	for _, tt := range []struct {
		a, b               labelWrite
		ifAFirst, ifBFirst error // the second's answer
	}{
		{labelMoves("Feature", "Docs"), labelMoves("Docs", "Feature"), withoutParent, withoutParent},
		{labelCreates("Icons", "Feature"), labelMoves("Feature", "Docs"), childless, withoutParent},
		{labelCreates("QA", ""), labelCreates("qa", ""), projectdomain.ErrLabelNameTaken, projectdomain.ErrLabelNameTaken},
		{labelMoves("Feature", "Docs"), labelMoves("Feature", "Bug"), nil, nil},
		{labelDeletes("Bug"), labelMoves("Feature", "Bug"), ofTheProject, nil},
		{labelDeletes("Bug"), labelCreates("Icons", "Bug"), ofTheProject, nil},
		{labelCreates("QA", ""), labelCreates("Icons", "Bug"), nil, nil},
	} {
		for _, aFirst := range []bool{true, false} {
			first, second, want := tt.a, tt.b, tt.ifAFirst
			if !aFirst {
				first, second, want = tt.b, tt.a, tt.ifBFirst
			}
			t.Run(first.name+", then "+second.name, func(t *testing.T) {
				w := newMemberWorld(t)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				ids := w.seedLabelsOf(t, w.web)
				w.seedLabelsOf(t, w.ops)
				labels, ops := w.labelsOf(t, w.web), w.labelsOf(t, w.ops)
				g := newGate()
				done := run(func() error { return first.run(ctx, w, labelWrittenHolding{projectpg.New(w.pool), g}, ids) })
				held(t, ctx, g, done, "the first write")
				answered := run(func() error { return second.run(ctx, w, projectpg.New(w.pool), ids) })
				pgtest.WaitForLockWaitOn(t, w.pool, "projects", 5*time.Second)
				opened := time.Now().Truncate(time.Microsecond)
				close(g.open)

				if err := result(t, ctx, done, "the first write"); err != nil {
					t.Errorf("%s = %v, want it done", first.name, err)
				}
				if err := result(t, ctx, answered, "the second write"); !sameLabelOutcome(err, want) {
					t.Errorf("%s = %v, want %v as its first problem", second.name, err, want)
				}
				first.apply(labels)
				if want == nil {
					second.apply(labels)
					var latest time.Time
					err := w.pool.QueryRow(pgtest.Soon(t), "SELECT max(updated_at) FROM labels WHERE project_id = $1", w.web).Scan(&latest)
					if err != nil || latest.Before(opened) {
						t.Errorf("Web's last write at %v (%v); want one no earlier than the gate's opening, %v", latest, err, opened)
					}
				}
				if got := w.labelsOf(t, w.web); !maps.Equal(got, labels) {
					t.Errorf("Web's labels after both: %v; want %v", got, labels)
				}
				if got := w.labelsOf(t, w.ops); !maps.Equal(got, ops) {
					t.Errorf("Ops's labels after both: %v; want them as they were, %v", got, ops)
				}
				var sharing int
				if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*) - count(DISTINCT sort_order) FROM labels
					WHERE project_id = $1 AND deleted_at IS NULL`, w.web).Scan(&sharing); err != nil || sharing != 0 {
					t.Errorf("%d of Web's labels share a sort order (%v); want each its own: a new label's is read under the project's lock",
						sharing, err)
				}
			})
		}
	}
}
