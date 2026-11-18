package bootstrap

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"uuid"

	issuedomain "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	notificationdomain "github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// madeIn is the work item the account by creates in project, whose
// identifier is identifier, through the API with body, as a change's other
// end names it (issueRefOf).
func (w issueWorld) madeIn(t *testing.T, by string, project uuid.UUID, identifier, body string) notificationdomain.IssueSnapshot {
	t.Helper()
	var answer struct {
		ID         uuid.UUID `json:"id"`
		SequenceID int       `json:"sequence_id"`
		Name       string    `json:"name"`
	}
	decodeAnswer(t, w.send(t, by, http.MethodPost, "/api/v0/projects/"+project.String()+"/issues", body, http.StatusCreated), &answer)
	return notificationdomain.IssueSnapshot{ID: answer.ID, Name: answer.Name, SequenceID: answer.SequenceID, Identifier: identifier}
}

// endsOf are the other ends a notification's change is answered with: its
// old_id, old_issue, new_id and new_issue, each a key it must have.
func endsOf(t *testing.T, row map[string]any) map[string]any {
	t.Helper()
	change, ends := noteChange(row), map[string]any{}
	for _, key := range []string{"old_id", "old_issue", "new_id", "new_issue"} {
		value, ok := change[key]
		if !ok {
			t.Errorf("notification %v answers its change without %s", row["id"], key)
		}
		ends[key] = value
	}
	return ends
}

// namedEnds are the other ends of a change from before to after as
// answered (spec 2.4, 2.6): each the work item's id and the work item
// issueRefOf names, both nil for nil.
func namedEnds(before, after *notificationdomain.IssueSnapshot) map[string]any {
	ends := map[string]any{"old_id": nil, "old_issue": nil, "new_id": nil, "new_issue": nil}
	if before != nil {
		ends["old_id"], ends["old_issue"] = before.ID.String(), issueRefOf(*before)
	}
	if after != nil {
		ends["new_id"], ends["new_issue"] = after.ID.String(), issueRefOf(*after)
	}
	return ends
}

// noteOn is the one of rows of the work item issue whose change is of
// field, with verb, but those of the ids except.
func noteOn(t *testing.T, rows []map[string]any, issue uuid.UUID, field, verb string, except ...string) map[string]any {
	t.Helper()
	var found []map[string]any
	for _, r := range rows {
		if c := noteChange(r); r["issue_id"] == issue.String() && c["field"] == field && c["verb"] == verb && !slices.Contains(except, fmt.Sprint(r["id"])) {
			found = append(found, r)
		}
	}
	if len(found) != 1 {
		t.Fatalf("the notifications of %s's %s %s: %v; want one", issue, field, verb, noteIDs(found))
	}
	return found[0]
}

// A change's other end, a parent or a relation's other work item, is
// answered as its reader reads it now (M7 P1c spec 2.4, 2.6, R4, R7;
// acceptance "可见"): with its id, number, identifier and name when he
// reads it, an archived work item and one of an archived project too;
// with the id and the work item both null when he does not, one of a
// project he is no member of, one his project's guest rule hides, one of
// another workspace, whose work items he reads there; and any id of a
// field outside the stored and the work-item fields never. bob, the
// assignee of alice's Fix, is sent her change of its parent from Shelf to
// Top, her relations of it to Shelf and Near, to Far, of Ops, which she
// removes, and to Hid, of the private Secret. walt, the creator of Mine,
// is sent her change of its parent from his own My top to Top and her
// relation of it to Top, both named while he is Web's member and Top not
// once he is its guest. Shelf is then archived, and so is Ops; SQL, as no
// write can, points bob's relation to Near at Away, of beta, and turns
// Fix's creation into a change of its cycle from Shelf to Top.
func TestNotificationsNameTheOtherEndsTheReaderReads(t *testing.T) {
	w := newIssueWorld(t)
	alice, bob := w.ids["alice"], w.ids["bob"]
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	secret := createdProjectFrom(t, w.contract, w.base, w.tokens["alice"], "acme", `{"name":"Secret","identifier":"SEC","network":0}`)
	w.makeMember(t, "bob", ops, shared.RoleMember)
	w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Beta","slug":"beta"}`, http.StatusCreated)
	site := createdProject(t, w.contract, w.base, w.tokens["alice"], "beta", "Site", "SITE")
	inWorkspaceOf(t, w.pool, site, bob, alice, shared.RoleMember)
	w.makeMember(t, "bob", site, shared.RoleMember)

	top, near := w.madeIn(t, "alice", w.web, "WEB", `{"name":"Top"}`), w.madeIn(t, "alice", w.web, "WEB", `{"name":"Near"}`)
	shelf := w.madeIn(t, "alice", w.web, "WEB", fmt.Sprintf(`{"name":"Shelf","state_id":%q}`, w.done))
	far, hid := w.madeIn(t, "alice", ops, "OPS", `{"name":"Far"}`), w.madeIn(t, "alice", secret, "SEC", `{"name":"Hid"}`)
	away := w.madeIn(t, "alice", site, "SITE", `{"name":"Away"}`)
	fix := w.madeIn(t, "alice", w.web, "WEB", fmt.Sprintf(`{"name":"Fix","parent_id":%q,"assignee_ids":[%q]}`, shelf.ID, bob))
	myTop := w.madeIn(t, "walt", w.web, "WEB", `{"name":"My top"}`)
	mine := w.madeIn(t, "walt", w.web, "WEB", fmt.Sprintf(`{"name":"Mine","parent_id":%q}`, myTop.ID))
	for _, item := range []uuid.UUID{fix.ID, mine.ID} {
		w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+item.String(), fmt.Sprintf(`{"parent_id":%q}`, top.ID), http.StatusOK)
	}
	w.send(t, "alice", http.MethodPost, relationsOf(fix.ID), relating("relates_to", shelf.ID, near.ID), http.StatusCreated)
	removed := listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost, relationsOf(fix.ID), relating("blocked_by", far.ID), http.StatusCreated))
	w.send(t, "alice", http.MethodDelete, relationPath(removed[0].ID), "", http.StatusNoContent)
	w.send(t, "alice", http.MethodPost, relationsOf(fix.ID), relating("blocking", hid.ID), http.StatusCreated)
	w.send(t, "alice", http.MethodPost, relationsOf(mine.ID), relating("duplicate", top.ID), http.StatusCreated)
	waitForFanouts(t, w.pool)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelf.ID.String()+"/archive", "", http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/archive", "", http.StatusOK)
	w.send(t, "bob", http.MethodGet, "/api/v0/issues/"+away.ID.String(), "", http.StatusOK)

	t.Run("a guest's", func(t *testing.T) {
		for _, phase := range []struct {
			role   string
			parent map[string]any
			same   map[string]any
		}{{"member", namedEnds(&myTop, &top), namedEnds(nil, &top)}, {"guest", namedEnds(&myTop, nil), namedEnds(nil, nil)}} {
			if phase.role == "guest" {
				w.send(t, "alice", http.MethodPatch, "/api/v0/project-members/"+w.memberRow(t, "project_members", "project_id", w.web, "walt"), `{"role":5}`,
					http.StatusOK)
			}
			walts := w.inbox(t, "walt", "").rows
			if got := endsOf(t, noteOn(t, walts, mine.ID, "parent", "updated")); !reflect.DeepEqual(got, phase.parent) {
				t.Errorf("as Web's %s, walt's Mine's parent ends %v; want %v", phase.role, got, phase.parent)
			}
			if got := endsOf(t, noteOn(t, walts, mine.ID, "duplicate", "created")); !reflect.DeepEqual(got, phase.same) {
				t.Errorf("as Web's %s, walt's Mine's duplicate ends %v; want %v", phase.role, got, phase.same)
			}
		}
	})

	pointed := queryIDs(t, w.pool, `SELECT id FROM notifications WHERE receiver_id = $1 AND data->'change'->>'new_id' = $2`, bob, near.ID.String())
	created := queryIDs(t, w.pool, `SELECT id FROM notifications WHERE receiver_id = $1 AND issue_id = $2 AND data->'change'->>'field' = 'issue'`, bob, fix.ID)
	if len(pointed) != 1 || len(created) != 1 {
		t.Fatalf("bob's notifications of the relation to Near %v and of Fix's creation %v; want one each", pointed, created)
	}
	for _, s := range []struct {
		id   uuid.UUID
		sql  string
		args []any
	}{
		{pointed[0], `jsonb_set(data, '{change,new_id}', to_jsonb($2::text))`, []any{away.ID.String()}},
		{created[0], `jsonb_set(data, '{change}', jsonb_build_object('field', 'cycle', 'verb', 'updated', 'old_id', $2::text, 'new_id', $3::text))`,
			[]any{shelf.ID.String(), top.ID.String()}},
	} {
		if tag, err := w.pool.Exec(pgtest.Soon(t), `UPDATE notifications SET data = `+s.sql+` WHERE id = $1`, append([]any{s.id}, s.args...)...); err != nil ||
			tag.RowsAffected() != 1 {
			t.Fatalf("rewriting notification %s: %v, %v", s.id, tag, err)
		}
	}
	bobs := w.inbox(t, "bob", "").rows
	byID := func(id uuid.UUID) map[string]any {
		at := slices.IndexFunc(bobs, func(r map[string]any) bool { return r["id"] == id.String() })
		if at < 0 {
			t.Fatalf("bob's list %v lacks notification %s", noteIDs(bobs), id)
		}
		return bobs[at]
	}
	for _, c := range []struct {
		what string
		row  map[string]any
		want map[string]any
	}{
		{"Fix's parent, from Shelf, archived, to Top", noteOn(t, bobs, fix.ID, "parent", "updated"), namedEnds(&shelf, &top)},
		{"Fix's relation to Shelf, archived", noteOn(t, bobs, fix.ID, "relates_to", "created", pointed[0].String()), namedEnds(nil, &shelf)},
		{"Fix's relation to Far, of the archived Ops", noteOn(t, bobs, fix.ID, "blocked_by", "created"), namedEnds(nil, &far)},
		{"its removal", noteOn(t, bobs, fix.ID, "blocked_by", "deleted"), namedEnds(&far, nil)},
		{"Fix's relation to Hid, of Secret, which bob is no member of", noteOn(t, bobs, fix.ID, "blocking", "created"), namedEnds(nil, nil)},
		{"Fix's relation pointed at Away, of beta", byID(pointed[0]), namedEnds(nil, nil)},
		{"a change of Fix's cycle, from Shelf to Top", byID(created[0]), namedEnds(nil, nil)},
	} {
		if got := endsOf(t, c.row); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: bob's notification %v ends %v; want %v", c.what, c.row["id"], got, c.want)
		}
	}
}

// relationTypesIn are the values of the RelationType constants file
// declares, as `X RelationType = "x"` or `X = RelationType("x")`.
func relationTypesIn(t *testing.T, file string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var types []string
	for _, decl := range parsed.Decls {
		d, ok := decl.(*ast.GenDecl)
		if !ok || d.Tok != token.CONST {
			continue
		}
		for _, spec := range d.Specs {
			value := spec.(*ast.ValueSpec)
			declared, ok := value.Type.(*ast.Ident)
			for _, v := range value.Values {
				typed := ok && declared.Name == "RelationType"
				if call, isCall := v.(*ast.CallExpr); isCall && len(call.Args) == 1 {
					if fun, isIdent := call.Fun.(*ast.Ident); isIdent && fun.Name == "RelationType" {
						typed, v = true, call.Args[0]
					}
				}
				if !typed {
					continue
				}
				literal, isLiteral := v.(*ast.BasicLit)
				if !isLiteral || literal.Kind != token.STRING {
					t.Fatalf("%s: the RelationType constant %v is no string literal", file, value.Names)
				}
				typ, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
				types = append(types, typ)
			}
		}
	}
	return types
}

// Every relation type the issue module defines is a field whose ids a
// reader is answered only as the work items he reads (M7 P1c spec 2.6, R7;
// pre-flight re-84 I7): one it adds and the notification module does not
// list would be answered as any other field, its ids never, rather than
// leak. The types are read from issue/domain's source, each constant of
// RelationType, so that a new one is among them without a change here;
// the four the module exports show the reading finds them.
func TestEveryRelationTypeIsAWorkItemField(t *testing.T) {
	files, err := filepath.Glob("../modules/issue/domain/*.go")
	if err != nil {
		t.Fatal(err)
	}
	var types []string
	for _, file := range files {
		if !strings.HasSuffix(file, "_test.go") {
			types = append(types, relationTypesIn(t, file)...)
		}
	}
	for _, exported := range []issuedomain.RelationType{issuedomain.RelatesTo, issuedomain.Duplicate, issuedomain.BlockedBy, issuedomain.Blocking} {
		if !slices.Contains(types, string(exported)) {
			t.Fatalf("the relation types read from issue/domain %v lack %s", types, exported)
		}
	}
	fields := notificationdomain.WorkItemFields()
	for _, typ := range types {
		if !slices.Contains(fields, typ) {
			t.Errorf("the relation type %s is not among the notification module's work-item fields %v", typ, fields)
		}
	}
}
