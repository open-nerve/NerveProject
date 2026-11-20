package app_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// assetProjects is the asset path's project read (app.AssetProjectReader)
// over the world's project: its archive and its intake's toggle as its
// lock answers them (fakeProject.locked), none when it has none. Each read
// is logged.
type assetProjects struct{ p *fakeProject }

func (f assetProjects) AssetProject(ctx context.Context, id uuid.UUID) (app.AssetProject, bool, error) {
	if err := f.p.log.add(ctx, "AssetProject", id); err != nil {
		return app.AssetProject{}, false, err
	}
	if f.p.locked == nil {
		return app.AssetProject{}, false, nil
	}
	return app.AssetProject{Archived: f.p.locked.Archived, IntakeView: f.p.locked.IntakeView}, true, nil
}

// intakeAssets is the triage's world (newTriage): web's intake turned on,
// carol's work item 7 in it, pending, each read of it carrying its intake
// row; the asset hosts' access over its Locks, alice's comment on the work
// item, and web read as assetProjects reads it.
type intakeAssets struct {
	*triageWorld
	access   *app.AssetAccess
	comments *writableComments
}

func newIntakeAssets(t *testing.T) *intakeAssets {
	w := newTriage(t)
	c := &writableComments{fakeComments: &fakeComments{fakeRelations: &fakeRelations{fakeIssues: w.issues}}, found: true,
		current: domain.Comment{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, IssueID: w.id, ActorID: alice}}
	a := app.NewAssetAccess(app.AssetAccessDeps{Locks: w.locks, Comments: c, Projects: assetProjects{p: w.project}})
	return &intakeAssets{triageWorld: w, access: a, comments: c}
}

// host is the host of the work item w.id as the access answers it.
func (w *intakeAssets) host() app.AssetHost {
	return app.AssetHost{WorkspaceID: acme, ProjectID: web, IssueID: w.id}
}

// decided are the actions w's decisions were asked, in order.
func (w *intakeAssets) decided() []shared.Action {
	var got []shared.Action
	for _, c := range w.log.calls {
		if f := strings.Fields(c); f[0] == "Authorize" {
			got = append(got, shared.Action(f[2]))
		}
	}
	return got
}

// projectRead reports whether w's project was read for the asset path.
func (w *intakeAssets) projectRead() bool {
	return slices.ContainsFunc(w.log.calls, func(c string) bool { return strings.HasPrefix(c, "AssetProject") })
}

// sameAnswer checks the access's answer got: none for a nil want, else
// want's code.
func sameAnswer(t *testing.T, what string, got, want error) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Errorf("%s = %v, want it allowed", what, got)
		}
		return
	}
	var se, wantSE *shared.Error
	if !errors.As(got, &se) || !errors.As(want, &wantSE) || se.Code != wantSE.Code {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

// assetWrites are the intents of an asset's writes on its host, by name.
var assetWrites = map[string]app.AssetIntent{"prepare": app.AssetPrepare, "remove": app.AssetRemove}

// An asset's write, a preparation or a removal, on a work item in its
// project's intake (M7 P2c spec 2.2): intake.update decides it, the
// submitter rule after it, the project's admin passing, before the lock
// (Inspect) and again under it (Check), where alone, for such a host, an
// archived project (409 project.archived), then its intake turned off (409
// issue.intake_disabled), then the item declined or a duplicate (409
// issue.intake_closed), by its stored status, refuse it; Inspect reads no
// project. Each refusal of Inspect is Check's too, each 403 before a 409.
// One accepted is issue.update's again, toggle and status aside; one
// accepted while the lock waited is decided so under it. Every case is
// asked of Inspect, then of Check on the host as it is, as a lock that
// waited would: grants are each decision's, answers too (nil allows).
func TestAnAssetWriteOnAWorkItemInTheIntake(t *testing.T) {
	gone := clockNow.Add(-24 * time.Hour)
	status := func(s domain.IntakeFacts) func(*intakeAssets) {
		return func(w *intakeAssets) { w.intake.facts[w.id] = s }
	}
	submitted := func(w *intakeAssets) { w.issues.issue.By = alice }
	unknown := func(w *intakeAssets) { w.issues.issue.By = uuid.UUID{} }
	off := func(w *intakeAssets) { w.project.locked.IntakeView = false }
	archived := func(w *intakeAssets) { w.project.locked.Archived = true }
	declined, duplicate := status(domain.IntakeFacts{Status: domain.IntakeDeclined}), status(domain.IntakeFacts{Status: domain.IntakeDuplicate})
	accepted := status(domain.IntakeFacts{Status: domain.IntakeAccepted})
	all := func(sets ...func(*intakeAssets)) func(*intakeAssets) {
		return func(w *intakeAssets) {
			for _, set := range sets {
				set(w)
			}
		}
	}
	intake, update := []shared.Action{domain.ActionUpdateIntake, domain.ActionUpdateIntake}, []shared.Action{domain.ActionUpdate, domain.ActionUpdate}
	twice := func(g shared.Grant) []shared.Grant { return []shared.Grant{g, g} }
	forbidden := shared.Forbidden()
	for _, tt := range []struct {
		name    string
		set     func(*intakeAssets)
		waited  func(*intakeAssets) // between Inspect and Check
		grants  []shared.Grant
		answers []error
		inspect error
		check   error
		actions []shared.Action
	}{
		{"web's admin", nil, nil, twice(webAdmin), nil, nil, nil, intake},
		{"web's admin, of no known creator", unknown, nil, twice(webAdmin), nil, nil, nil, intake},
		{"a member who submitted it", submitted, nil, twice(webMember), nil, nil, nil, intake},
		{"a guest who submitted it, who sees only his own", submitted, nil, twice(ownOnly), nil, nil, nil, intake},
		{"a guest who submitted it, who sees every work item", submitted, nil, twice(everyItem), nil, nil, nil, intake},
		{"a snooze whose time has come", all(submitted, status(domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &gone})), nil,
			twice(webMember), nil, nil, nil, intake},
		{"a member who did not submit it", nil, nil, twice(webMember), nil, forbidden, forbidden, intake},
		{"a guest who sees every work item, did not submit it", nil, nil, twice(everyItem), nil, forbidden, forbidden, intake},
		{"a member, of no known creator", unknown, nil, twice(webMember), nil, forbidden, forbidden, intake},
		{"its intake turned off", off, nil, twice(webAdmin), nil, nil, domain.ErrIntakeDisabled, intake},
		{"declined", declined, nil, twice(webAdmin), nil, nil, domain.ErrIntakeClosed, intake},
		{"a duplicate", duplicate, nil, twice(webAdmin), nil, nil, domain.ErrIntakeClosed, intake},
		{"its project archived", archived, nil, twice(webAdmin), nil, nil, domain.ErrProjectArchived, intake},
		{"archived, its intake off", all(archived, off), nil, twice(webAdmin), nil, nil, domain.ErrProjectArchived, intake},
		{"its intake off, declined", all(off, declined), nil, twice(webAdmin), nil, nil, domain.ErrIntakeDisabled, intake},
		{"a member who did not submit it, declined", declined, nil, twice(webMember), nil, forbidden, forbidden, intake},
		{"a member who did not submit it, its intake off", off, nil, twice(webMember), nil, forbidden, forbidden, intake},
		{"a member since the lock waited, archived, its intake off, declined", all(archived, off, declined), nil,
			[]shared.Grant{webAdmin, webMember}, nil, nil, forbidden, intake},
		{"accepted, a guest who submitted it", all(accepted, submitted), nil, twice(ownOnly), []error{forbidden, forbidden}, forbidden, forbidden,
			update},
		{"accepted, a member who did not submit it, its intake off", all(accepted, off), nil, twice(webMember), nil, nil, nil, update},
		{"accepted while the lock waited, a guest who submitted it", submitted, accepted, twice(ownOnly), []error{nil, forbidden}, nil, forbidden,
			[]shared.Action{domain.ActionUpdateIntake, domain.ActionUpdate}},
	} {
		for write, intent := range assetWrites {
			t.Run(line(tt.name, write), func(t *testing.T) {
				w := newIntakeAssets(t)
				if tt.set != nil {
					tt.set(w)
				}
				w.auth.grants, w.auth.answers = tt.grants, tt.answers
				q := app.AssetRequest{ID: w.id, Kind: app.AssetIssue, Intent: intent}
				h, err := w.access.Inspect(t.Context(), shared.Actor{UserID: alice}, q)
				sameAnswer(t, "Inspect()", err, tt.inspect)
				if err == nil && h != w.host() {
					t.Errorf("Inspect() = %+v, want %+v", h, w.host())
				}
				if w.projectRead() {
					t.Errorf("Inspect() read the project: %q", w.log.calls)
				}
				if tt.waited != nil {
					tt.waited(w)
				}
				sameAnswer(t, "Check()", w.access.Check(t.Context(), shared.Actor{UserID: alice}, q, w.host()), tt.check)
				if got := w.decided(); !slices.Equal(got, tt.actions) {
					t.Errorf("decided %q, want %q", got, tt.actions)
				}
			})
		}
	}
}

// A guest who sees only his own work items finds another's in the intake,
// carol's, as unknown as a random id, before Inspect and under the lock:
// the guest rule's 404 comes before intake.update's submitter rule, and it
// is the work item's very issue.not_found, never the intake's (M7 P2c spec
// 2.2): AssetHostNotFound turns that one into asset.not_found.
func TestAnAssetWriteOnAnothersWorkItemInTheIntakeIsNotFound(t *testing.T) {
	for write, intent := range assetWrites {
		for _, random := range []bool{false, true} {
			t.Run(line(write, "random", random), func(t *testing.T) {
				w := newIntakeAssets(t)
				w.auth.grants = []shared.Grant{ownOnly, ownOnly}
				id := w.id
				if random {
					id = uuid.NewV7()
				}
				q := app.AssetRequest{ID: id, Kind: app.AssetIssue, Intent: intent}
				_, inspected := w.access.Inspect(t.Context(), shared.Actor{UserID: alice}, q)
				checked := w.access.Check(t.Context(), shared.Actor{UserID: alice}, q, w.host())
				for _, err := range []error{inspected, checked} {
					if se := (*shared.Error)(nil); !errors.As(err, &se) || se != domain.ErrIssueNotFound {
						t.Errorf("Inspect(), Check() = %#v, %#v; want the very value a random id answers, %#v", inspected, checked,
							domain.ErrIssueNotFound)
					}
				}
			})
		}
	}
}

// What is not an asset's write on the work item in the intake keeps its
// rules (M7 P2c spec 1, 2.2, R5): a read is issue.read's, the comment's
// and the new comment's hosts their own, with neither the intake's toggle
// nor its item's status judged, for a member who did not submit it, of an
// item declined in an intake turned off.
func TestAssetReadsAndCommentsOfAWorkItemInTheIntakeKeepTheirRules(t *testing.T) {
	for _, tt := range []struct {
		name   string
		kind   app.AssetHostKind
		intent app.AssetIntent
		action shared.Action
	}{
		{"the work item's read", app.AssetIssue, app.AssetRead, domain.ActionRead},
		{"a comment's read", app.AssetComment, app.AssetRead, domain.ActionRead},
		{"a comment's preparation", app.AssetComment, app.AssetPrepare, domain.ActionWriteComment},
		{"a comment's removal", app.AssetComment, app.AssetRemove, domain.ActionWriteComment},
		{"a new comment's read", app.AssetNewComment, app.AssetRead, domain.ActionRead},
		{"a new comment's preparation", app.AssetNewComment, app.AssetPrepare, domain.ActionCreateComment},
		{"a new comment's removal", app.AssetNewComment, app.AssetRemove, domain.ActionCreateComment},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newIntakeAssets(t)
			w.intake.facts[w.id] = domain.IntakeFacts{Status: domain.IntakeDeclined}
			w.project.locked.IntakeView = false
			w.auth.grants = []shared.Grant{webMember, webMember}
			q, want := app.AssetRequest{ID: w.id, Kind: tt.kind, Intent: tt.intent}, w.host()
			if tt.kind == app.AssetComment {
				q.ID, want.CommentID = w.comments.current.ID, w.comments.current.ID
			}
			h, err := w.access.Inspect(t.Context(), shared.Actor{UserID: alice}, q)
			if err != nil || h != want {
				t.Fatalf("Inspect() = %+v, %v; want %+v", h, err, want)
			}
			sameAnswer(t, "Check()", w.access.Check(t.Context(), shared.Actor{UserID: alice}, q, h), nil)
			if got := w.decided(); !slices.Equal(got, []shared.Action{tt.action, tt.action}) {
				t.Errorf("decided %q, want %q twice", got, tt.action)
			}
		})
	}
}
